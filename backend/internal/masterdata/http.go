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

func RegisterMeasurementRoutes(router *gin.Engine, service *MeasurementService, authenticator Authenticator) {
	g := router.Group("/master-data", authenticateMasterData(authenticator))
	g.GET("/measurements", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		q, ok := listQuery(c)
		if !ok {
			return
		}
		items, total, err := service.List(c, actorFrom(c), q)
		if writeError(c, err) {
			return
		}
		if items == nil {
			items = []Measurement{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})
	g.GET("/measurements/:id", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
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
	g.POST("/measurements", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		var input MeasurementInput
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
	g.PUT("/measurements/:id", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var input MeasurementInput
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
