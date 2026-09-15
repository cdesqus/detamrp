package production

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"order-stock/backend/internal/rbac"
)

type CostRepository interface {
	ListOrderCosts(context.Context, Actor) ([]OrderCost, error)
	GetOrderCost(context.Context, Actor, uuid.UUID) (OrderCost, error)
	ListPeriodCosts(context.Context, Actor) ([]PeriodCost, error)
}

// Actual costs are restricted to production.report; the routes carry no
// operator-facing variant, so no redaction is needed here.
func RegisterCostRoutes(router *gin.Engine, s CostRepository, authenticator Authenticator) {
	g := router.Group("/production-costs", executionAuth(authenticator), rbac.RequirePermissions("production.report"))
	g.GET("", func(c *gin.Context) {
		items, e := s.ListOrderCosts(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/periods", func(c *gin.Context) {
		items, e := s.ListPeriodCosts(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/:id", func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		v, e := s.GetOrderCost(c, executionActor(c), id)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, v)
	})
}
