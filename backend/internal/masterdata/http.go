package masterdata

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

const actorContextKey = "master_data_actor"

func RegisterUnitRoutes(router *gin.Engine, service *UnitService, authenticator Authenticator) {
	g := router.Group("/master-data", authenticateMasterData(authenticator))
	g.GET("/units", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		q, ok := listQuery(c)
		if !ok {
			return
		}
		items, total, err := service.List(c, actorFrom(c), q)
		if writeError(c, err) {
			return
		}
		if items == nil {
			items = []Unit{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})
	g.GET("/units/:id", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		item, err := service.Get(c, actorFrom(c), id)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusOK, item)
	})
	g.POST("/units", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		var input UnitInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
			return
		}
		item, err := service.Create(c, actorFrom(c), input)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusCreated, item)
	})
	g.PUT("/units/:id", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var input UnitInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
			return
		}
		item, err := service.Update(c, actorFrom(c), id, input)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusOK, item)
	})
}

func RegisterSupplierRoutes(router *gin.Engine, service *SupplierService, authenticator Authenticator) {
	g := router.Group("/master-data", authenticateMasterData(authenticator))
	g.GET("/suppliers", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		q, ok := listQuery(c)
		if !ok {
			return
		}
		items, total, err := service.List(c, actorFrom(c), q)
		if writeError(c, err) {
			return
		}
		if items == nil {
			items = []Supplier{}
		}
		c.JSON(200, gin.H{"items": items, "total": total})
	})
	g.GET("/suppliers/:id", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		item, err := service.Get(c, actorFrom(c), id)
		if writeError(c, err) {
			return
		}
		c.JSON(200, item)
	})
	g.POST("/suppliers", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		var in SupplierInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "invalid_request"})
			return
		}
		item, err := service.Create(c, actorFrom(c), in)
		if writeError(c, err) {
			return
		}
		c.JSON(201, item)
	})
	g.PUT("/suppliers/:id", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var in SupplierInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "invalid_request"})
			return
		}
		item, err := service.Update(c, actorFrom(c), id, in)
		if writeError(c, err) {
			return
		}
		c.JSON(200, item)
	})
}

func RegisterRawMaterialRoutes(router *gin.Engine, service *RawMaterialService, authenticator Authenticator) {
	g := router.Group("/master-data", authenticateMasterData(authenticator))
	g.GET("/raw-materials", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		q, ok := listQuery(c)
		if !ok {
			return
		}
		if raw := c.Query("supplierId"); raw != "" {
			id, e := uuid.Parse(raw)
			if e != nil {
				c.JSON(400, gin.H{"error": "invalid_filter", "fields": FieldErrors{"supplierId": "Invalid supplier"}})
				return
			}
			q.SupplierID = id
		}
		items, total, err := service.List(c, actorFrom(c), q)
		if writeError(c, err) {
			return
		}
		if items == nil {
			items = []RawMaterial{}
		}
		c.JSON(200, gin.H{"items": items, "total": total})
	})
	g.GET("/raw-materials/:id", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		item, err := service.Get(c, actorFrom(c), id)
		if writeError(c, err) {
			return
		}
		c.JSON(200, item)
	})
	g.POST("/raw-materials", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		var in RawMaterialInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "invalid_request"})
			return
		}
		item, err := service.Create(c, actorFrom(c), in)
		if writeError(c, err) {
			return
		}
		c.JSON(201, item)
	})
	g.PUT("/raw-materials/:id", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var in RawMaterialInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "invalid_request"})
			return
		}
		item, err := service.Update(c, actorFrom(c), id, in)
		if writeError(c, err) {
			return
		}
		c.JSON(200, item)
	})
}

func authenticateMasterData(a Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
			return
		}
		user, err := a.Authenticate(c, token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
			return
		}
		c.Set(actorContextKey, Actor{TenantID: user.TenantID, UserID: user.ID})
		c.Set(rbac.ContextPermissionsKey, user.Permissions)
		c.Next()
	}
}

func actorFrom(c *gin.Context) Actor { value, _ := c.Get(actorContextKey); return value.(Actor) }
func routeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid_id"})
		return uuid.Nil, false
	}
	return id, true
}
func listQuery(c *gin.Context) (ListQuery, bool) {
	q := ListQuery{Search: c.Query("search")}
	if raw, exists := c.GetQuery("active"); exists {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid_filter", "fields": FieldErrors{"active": "Must be true or false"}})
			return q, false
		}
		q.Active = &v
	}
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid_filter"})
			return q, false
		}
		q.Limit = v
	}
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid_filter"})
			return q, false
		}
		q.Offset = v
	}
	return q, true
}
func writeError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var validation ValidationError
	var conflict ConflictError
	var missing NotFoundError
	switch {
	case errors.As(err, &validation):
		c.JSON(400, gin.H{"error": "validation_failed", "message": "Please correct the highlighted fields", "fields": validation.Fields})
	case errors.As(err, &conflict):
		c.JSON(409, gin.H{"error": "conflict", "message": "Data is already in use", "fields": conflict.Fields})
	case errors.As(err, &missing):
		c.JSON(404, gin.H{"error": "not_found", "message": missing.Error()})
	default:
		c.JSON(500, gin.H{"error": "internal_error", "message": "Request could not be completed"})
	}
	return true
}
