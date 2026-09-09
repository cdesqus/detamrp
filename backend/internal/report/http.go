package report

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

func RegisterRoutes(router *gin.Engine, store *Store, authn Authenticator) {
	middleware := func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		user, err := authn.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Set("report_actor", Actor{TenantID: user.TenantID, UserID: user.ID})
		c.Set(rbac.ContextPermissionsKey, user.Permissions)
		c.Next()
	}
	group := router.Group("/reports", middleware, rbac.RequirePermissions("receiving.view"))
	handler := func(c *gin.Context, pdf bool) {
		filter, fields := ParseFilter(c.Request.URL.Query())
		if len(fields) > 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "validation failed", "fields": fields})
			return
		}
		actor, _ := c.Get("report_actor")
		result, err := store.ListReceiving(c, actor.(Actor), filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "report could not be loaded"})
			return
		}
		if !pdf {
			c.JSON(http.StatusOK, result)
			return
		}
		data, err := RenderReceivingPDF(result, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "report could not be exported"})
			return
		}
		c.Header("Content-Disposition", `inline; filename="receiving-report.pdf"`)
		c.Data(http.StatusOK, "application/pdf", data)
	}
	group.GET("/receiving", func(c *gin.Context) { handler(c, false) })
	group.GET("/receiving.pdf", func(c *gin.Context) { handler(c, true) })
	group.GET("/sales-orders", func(c *gin.Context) {
		actor, _ := c.Get("report_actor")
		items, err := store.ListSalesOrders(c, actor.(Actor))
		if err != nil {
			c.JSON(500, gin.H{"error": "report could not be loaded"})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	group.GET("/material-requirements", func(c *gin.Context) {
		actor, _ := c.Get("report_actor")
		items, err := store.ListMaterialRequirements(c, actor.(Actor))
		if err != nil {
			c.JSON(500, gin.H{"error": "report could not be loaded"})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	group.GET("/customer-deliveries", func(c *gin.Context) {
		actor, _ := c.Get("report_actor")
		items, err := store.ListCustomerDeliveries(c, actor.(Actor))
		if err != nil {
			c.JSON(500, gin.H{"error": "report could not be loaded"})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
}
