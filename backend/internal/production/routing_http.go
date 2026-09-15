package production

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"order-stock/backend/internal/rbac"
)

type RoutingRepository interface {
	ListRoutings(context.Context, Actor) ([]Routing, error)
	GetRouting(context.Context, Actor, uuid.UUID) (Routing, error)
	RoutingOptions(context.Context, Actor) (RoutingOptions, error)
	CreateRouting(context.Context, Actor, RoutingInput) (Routing, error)
	UpdateRouting(context.Context, Actor, uuid.UUID, RoutingInput) (Routing, error)
	RoutingAction(context.Context, Actor, uuid.UUID, string) (Routing, error)
}

func RegisterRoutingRoutes(router *gin.Engine, s RoutingRepository, authenticator Authenticator) {
	g := router.Group("/production-routings", executionAuth(authenticator))
	g.GET("", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		items, e := s.ListRoutings(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, gin.H{"items": items}))
	})
	g.GET("/options", rbac.RequirePermissions("production.routing", "production.report"), func(c *gin.Context) {
		options, e := s.RoutingOptions(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, options))
	})
	g.GET("/:id", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		r, e := s.GetRouting(c, executionActor(c), id)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, r))
	})
	save := func(update bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			var input RoutingInput
			if !bindExecution(c, &input) {
				return
			}
			input.Normalize()
			if e := input.Validate(!update); e != nil {
				respondError(c, e)
				return
			}
			var r Routing
			var e error
			code := 201
			if update {
				id, ok := planID(c)
				if !ok {
					return
				}
				r, e = s.UpdateRouting(c, executionActor(c), id, input)
				code = 200
			} else {
				r, e = s.CreateRouting(c, executionActor(c), input)
			}
			if e != nil {
				respondError(c, e)
				return
			}
			c.JSON(code, productionEntryPayload(c, r))
		}
	}
	g.POST("", rbac.RequirePermissions("production.routing", "production.report"), save(false))
	g.PUT("/:id", rbac.RequirePermissions("production.routing", "production.report"), save(true))
	for _, action := range []string{"activate", "deactivate"} {
		g.POST("/:id/"+action, rbac.RequirePermissions("production.routing", "production.report"), routingAction(s, action))
	}
	g.DELETE("/:id", rbac.RequirePermissions("production.routing", "production.report"), routingAction(s, "delete"))
}

func routingAction(s RoutingRepository, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		r, e := s.RoutingAction(c, executionActor(c), id, action)
		if e != nil {
			respondError(c, e)
			return
		}
		if action == "delete" {
			c.Status(204)
			return
		}
		c.JSON(200, productionEntryPayload(c, r))
	}
}
