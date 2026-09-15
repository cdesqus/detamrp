package production

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"order-stock/backend/internal/rbac"
)

type OrderRepository interface {
	ListOrders(context.Context, Actor) ([]Order, error)
	GetOrder(context.Context, Actor, uuid.UUID) (Order, error)
	OrderOptions(context.Context, Actor) (OrderOptions, error)
	CreateOrder(context.Context, Actor, OrderInput) (Order, error)
	UpdateOrder(context.Context, Actor, uuid.UUID, OrderInput) (Order, error)
	OrderAction(context.Context, Actor, uuid.UUID, string, string) (Order, error)
}

func executionAuth(authenticator Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, e := c.Cookie("session")
		if e != nil {
			c.AbortWithStatusJSON(401, gin.H{"message": "Authentication is required"})
			return
		}
		u, e := authenticator.Authenticate(c, token)
		if e != nil {
			c.AbortWithStatusJSON(401, gin.H{"message": "Authentication is required"})
			return
		}
		c.Set("production_actor", Actor{u.TenantID, u.ID})
		c.Set(rbac.ContextPermissionsKey, u.Permissions)
		c.Next()
	}
}
func executionActor(c *gin.Context) Actor { return c.MustGet("production_actor").(Actor) }
func bindExecution(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024*1024)
	if c.ShouldBindJSON(v) != nil {
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return false
	}
	return true
}
func RegisterOrderRoutes(router *gin.Engine, s OrderRepository, authenticator Authenticator) {
	g := router.Group("/production-orders", executionAuth(authenticator))
	g.GET("", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		v, e := s.ListOrders(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, gin.H{"items": v}))
	})
	g.GET("/options", rbac.RequirePermissions("production.order"), func(c *gin.Context) {
		v, e := s.OrderOptions(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, v))
	})
	g.GET("/:id", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		v, e := s.GetOrder(c, executionActor(c), id)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, v))
	})
	save := func(update bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			var input OrderInput
			if !bindExecution(c, &input) {
				return
			}
			if e := input.Validate(!update); e != nil {
				respondError(c, e)
				return
			}
			var v Order
			var e error
			code := 201
			if update {
				id, ok := planID(c)
				if !ok {
					return
				}
				v, e = s.UpdateOrder(c, executionActor(c), id, input)
				code = 200
			} else {
				v, e = s.CreateOrder(c, executionActor(c), input)
			}
			if e != nil {
				respondError(c, e)
				return
			}
			c.JSON(code, productionEntryPayload(c, v))
		}
	}
	g.POST("", rbac.RequirePermissions("production.order"), save(false))
	g.PUT("/:id", rbac.RequirePermissions("production.order"), save(true))
	for _, action := range []string{"start", "cancel", "delete"} {
		h := func(c *gin.Context) {
			id, ok := planID(c)
			if !ok {
				return
			}
			var input struct {
				Reason string `json:"reason"`
			}
			if action == "cancel" && !bindExecution(c, &input) {
				return
			}
			v, e := s.OrderAction(c, executionActor(c), id, action, input.Reason)
			if e != nil {
				respondError(c, e)
				return
			}
			if action == "delete" {
				c.Status(204)
			} else {
				c.JSON(200, productionEntryPayload(c, v))
			}
		}
		if action == "delete" {
			g.DELETE("/:id", rbac.RequirePermissions("production.order"), h)
		} else {
			g.POST("/:id/"+action, rbac.RequirePermissions("production.order"), h)
		}
	}
}
