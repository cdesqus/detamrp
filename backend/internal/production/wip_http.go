package production

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"order-stock/backend/internal/rbac"
)

type WIPRepository interface {
	ListWIP(context.Context, Actor) ([]WIPOrderBalance, error)
	GetWIP(context.Context, Actor, uuid.UUID) (WIPOrderBalance, error)
	CreateTransfer(context.Context, Actor, TransferInput) (WIPOrderBalance, error)
	ReverseMovement(context.Context, Actor, uuid.UUID, string) (WIPOrderBalance, error)
}

func RegisterWIPRoutes(router *gin.Engine, s WIPRepository, authenticator Authenticator) {
	g := router.Group("/production-wip", executionAuth(authenticator))
	g.GET("", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		items, e := s.ListWIP(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, gin.H{"items": items}))
	})
	g.GET("/:id", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		v, e := s.GetWIP(c, executionActor(c), id)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, v))
	})
	g.POST("/transfers", rbac.RequirePermissions("production.wip"), func(c *gin.Context) {
		var input TransferInput
		if !bindExecution(c, &input) {
			return
		}
		if e := input.Validate(); e != nil {
			respondError(c, e)
			return
		}
		v, e := s.CreateTransfer(c, executionActor(c), input)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(201, productionEntryPayload(c, v))
	})
	g.POST("/movements/:id/reverse", rbac.RequirePermissions("production.wip"), func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		var input struct {
			Reason string `json:"reason"`
		}
		if !bindExecution(c, &input) {
			return
		}
		v, e := s.ReverseMovement(c, executionActor(c), id, input.Reason)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, v))
	})
}
