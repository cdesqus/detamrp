package dashboard

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

const actorKey = "dashboard_actor"

func RegisterRoutes(router *gin.Engine, store *Store, authenticator Authenticator) {
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
	router.GET("/dashboard", authenticate, rbac.RequirePermissions("dashboard.view"), func(c *gin.Context) {
		filter, fields := ParseFilter(c.Request.URL.Query(), time.Now())
		if len(fields) > 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "validation failed", "fields": fields})
			return
		}
		value, _ := c.Get(actorKey)
		snapshot, err := store.Snapshot(c, value.(Actor), filter)
		if errors.Is(err, ErrSupplierNotFound) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "validation failed", "fields": gin.H{"supplierId": "Select a valid supplier."}})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "dashboard could not be loaded"})
			return
		}
		c.JSON(http.StatusOK, snapshot)
	})
}
