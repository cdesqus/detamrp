package salesmaster

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

const actorContextKey = "sales_master_actor"

func RegisterRoutes(router *gin.Engine, service *Service, authenticator Authenticator) {
	group := router.Group("", authenticate(authenticator))
	registerCustomerRoutes(group, service)
	registerFinishedGoodRoutes(group, service)
}

func registerCustomerRoutes(group *gin.RouterGroup, service *Service) {
	group.GET("/customers", rbac.RequirePermissions("customer.view"), func(c *gin.Context) {
		query, ok := parseListQuery(c)
		if !ok {
			return
		}
		items, total, err := service.ListCustomers(c, actorFrom(c), query)
		if writeError(c, err) {
			return
		}
		if items == nil {
			items = []Customer{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})
	group.GET("/customers/:id", rbac.RequirePermissions("customer.view"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		item, err := service.GetCustomer(c, actorFrom(c), id)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusOK, item)
	})
	group.POST("/customers", rbac.RequirePermissions("customer.manage"), func(c *gin.Context) {
		var input CustomerInput
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
			return
		}
		item, err := service.CreateCustomer(c, actorFrom(c), input)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusCreated, item)
	})
	update := func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var input CustomerInput
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
			return
		}
		item, err := service.UpdateCustomer(c, actorFrom(c), id, input)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusOK, item)
	}
	group.PUT("/customers/:id", rbac.RequirePermissions("customer.manage"), update)
	group.PATCH("/customers/:id", rbac.RequirePermissions("customer.manage"), update)
}

func registerFinishedGoodRoutes(group *gin.RouterGroup, service *Service) {
	group.GET("/finished-goods", rbac.RequirePermissions("fg.view"), func(c *gin.Context) {
		query, ok := parseListQuery(c)
		if !ok {
			return
		}
		items, total, err := service.ListFinishedGoods(c, actorFrom(c), query)
		if writeError(c, err) {
			return
		}
		if items == nil {
			items = []FinishedGood{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})
	group.GET("/finished-goods/:id", rbac.RequirePermissions("fg.view"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		item, err := service.GetFinishedGood(c, actorFrom(c), id)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusOK, item)
	})
	group.POST("/finished-goods", rbac.RequirePermissions("fg.manage", "fg.price.manage"), func(c *gin.Context) {
		var input FinishedGoodInput
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
			return
		}
		item, err := service.CreateFinishedGood(c, actorFrom(c), input)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusCreated, item)
	})
	update := func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var input FinishedGoodInput
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
			return
		}
		item, err := service.UpdateFinishedGood(c, actorFrom(c), id, input)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusOK, item)
	}
	group.PUT("/finished-goods/:id", rbac.RequirePermissions("fg.manage"), update)
	group.PATCH("/finished-goods/:id", rbac.RequirePermissions("fg.manage"), update)
}

func authenticate(authenticator Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		user, err := authenticator.Authenticate(c, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Set(actorContextKey, Actor{TenantID: user.TenantID, UserID: user.ID, Permissions: user.Permissions})
		c.Set(rbac.ContextPermissionsKey, user.Permissions)
		c.Next()
	}
}

func actorFrom(c *gin.Context) Actor {
	value, _ := c.Get(actorContextKey)
	return value.(Actor)
}

func routeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
		return uuid.Nil, false
	}
	return id, true
}

func parseListQuery(c *gin.Context) (ListQuery, bool) {
	query := ListQuery{Search: c.Query("search")}
	if raw, exists := c.GetQuery("active"); exists {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_filter", "fields": FieldErrors{"active": "Must be true or false"}})
			return query, false
		}
		query.Active = &value
	}
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_filter"})
			return query, false
		}
		query.Limit = value
	}
	if raw := c.Query("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_filter"})
			return query, false
		}
		query.Offset = value
	}
	return query, true
}

func writeError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var validation ValidationError
	var conflict ConflictError
	var missing NotFoundError
	var forbidden ForbiddenError
	switch {
	case errors.As(err, &validation):
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": "Please correct the highlighted fields", "fields": validation.Fields})
	case errors.As(err, &conflict):
		c.JSON(http.StatusConflict, gin.H{"error": "conflict", "message": "Data is already in use", "fields": conflict.Fields})
	case errors.As(err, &missing):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": missing.Error()})
	case errors.As(err, &forbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "permission_denied", "message": forbidden.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "Request could not be completed"})
	}
	return true
}
