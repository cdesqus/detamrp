package bom

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

const actorKey = "bom_actor"

func RegisterRoutes(router *gin.Engine, service *Service, authenticator Authenticator) {
	g := router.Group("", authenticate(authenticator))
	g.GET("/boms", rbac.RequirePermissions("bom.view"), func(c *gin.Context) {
		items, err := service.List(c, actorFrom(c))
		if err != nil {
			c.JSON(500, gin.H{"message": "BOMs could not be loaded"})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/boms/:id", rbac.RequirePermissions("bom.view"), func(c *gin.Context) {
		id, ok := idFrom(c)
		if !ok {
			return
		}
		item, err := service.Get(c, actorFrom(c), id)
		if err != nil {
			c.JSON(404, gin.H{"message": "BOM not found"})
			return
		}
		c.JSON(200, item)
	})
	g.POST("/boms", rbac.RequirePermissions("bom.manage"), func(c *gin.Context) {
		var input Input
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, gin.H{"message": "Invalid JSON body"})
			return
		}
		item, err := service.Create(c, actorFrom(c), input)
		if err != nil {
			c.JSON(422, gin.H{"message": err.Error()})
			return
		}
		c.JSON(201, item)
	})
	g.PUT("/boms/:id", rbac.RequirePermissions("bom.manage"), func(c *gin.Context) {
		id, ok := idFrom(c); if !ok { return }; var input Input
		if c.ShouldBindJSON(&input) != nil { c.JSON(400, gin.H{"message":"Invalid JSON body"}); return }
		item, err := service.Update(c, actorFrom(c), id, input); if err != nil { c.JSON(422, gin.H{"message":err.Error()}); return }; c.JSON(200, item)
	})
	g.POST("/boms/:id/activate", rbac.RequirePermissions("bom.activate"), func(c *gin.Context) {
		id, ok := idFrom(c)
		if !ok {
			return
		}
		item, err := service.Activate(c, actorFrom(c), id)
		if err != nil {
			c.JSON(409, gin.H{"message": err.Error()})
			return
		}
		c.JSON(200, item)
	})
}
func authenticate(authenticator Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("session")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication is required"})
			c.Abort()
			return
		}
		user, err := authenticator.Authenticate(c, cookie)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication is required"})
			c.Abort()
			return
		}
		c.Set(actorKey, Actor{TenantID: user.TenantID, UserID: user.ID})
		c.Set(rbac.ContextPermissionsKey, user.Permissions)
		c.Next()
	}
}
func actorFrom(c *gin.Context) Actor { return c.MustGet(actorKey).(Actor) }
func idFrom(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID"})
		return uuid.Nil, false
	}
	return id, true
}
