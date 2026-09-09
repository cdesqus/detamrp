package salesorder

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

const actorKey = "sales_order_actor"

func RegisterRoutes(router *gin.Engine, service *Service, authenticator Authenticator) {
	g := router.Group("", authenticate(authenticator))
	g.POST("/sales-orders", rbac.RequirePermissions("sales_order.create"), func(c *gin.Context) {
		var input Input
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, gin.H{"message": "Invalid JSON body"})
			return
		}
		item, err := service.Create(c, actor(c), input)
		if err != nil {
			c.JSON(422, gin.H{"message": err.Error()})
			return
		}
		c.JSON(201, item)
	})
	g.POST("/sales-orders/:id/submit", rbac.RequirePermissions("sales_order.submit"), func(c *gin.Context) {
		id, e := uuid.Parse(c.Param("id"))
		if e != nil {
			c.JSON(400, gin.H{"message": "Invalid ID"})
			return
		}
		item, e := service.Submit(c, actor(c), id)
		if e != nil {
			c.JSON(409, gin.H{"message": e.Error()})
			return
		}
		c.JSON(200, item)
	})
}
func authenticate(authenticator Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, e := c.Cookie("session")
		if e != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication is required"})
			c.Abort()
			return
		}
		user, e := authenticator.Authenticate(c, cookie)
		if e != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication is required"})
			c.Abort()
			return
		}
		c.Set(actorKey, Actor{TenantID: user.TenantID, UserID: user.ID})
		c.Next()
	}
}
func actor(c *gin.Context) Actor { return c.MustGet(actorKey).(Actor) }
