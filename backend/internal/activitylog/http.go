package activitylog

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

type Lister interface {
	List(context.Context, Actor, Query) (Page, error)
}

const actorKey = "activity_log_actor"

func RegisterRoutes(router *gin.Engine, lister Lister, authenticator Authenticator) {
	authenticate := func(c *gin.Context) {
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
		c.Set(actorKey, Actor{TenantID: user.TenantID, UserID: user.ID})
		c.Set(rbac.ContextPermissionsKey, user.Permissions)
		c.Next()
	}

	router.GET("/activity-logs", authenticate, rbac.RequirePermissions("activity_log.view"), func(c *gin.Context) {
		query, fields := ParseQuery(c.Request.URL.Query(), time.Now())
		if len(fields) > 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "validation failed", "fields": fields})
			return
		}
		value, _ := c.Get(actorKey)
		page, err := lister.List(c, value.(Actor), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "activity log could not be loaded"})
			return
		}
		c.JSON(http.StatusOK, page)
	})
}
