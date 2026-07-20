package rbac

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const ContextPermissionsKey = "permissions"

func RequirePermissions(required ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(ContextPermissionsKey)
		permissions, valid := value.([]string)
		if !exists || !valid || !Allows(permissions, required...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		c.Next()
	}
}
