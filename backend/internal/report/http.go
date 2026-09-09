package report

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
	group := router.Group("/reports", middleware)
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
	group.GET("/receiving", rbac.RequirePermissions("receiving.view"), func(c *gin.Context) { handler(c, false) })
	group.GET("/receiving.pdf", rbac.RequirePermissions("receiving.view"), func(c *gin.Context) { handler(c, true) })
	group.GET("/sales-orders", rbac.RequirePermissions("sales_report.view"), func(c *gin.Context) {
		actor, _ := c.Get("report_actor")
		items, err := store.ListSalesOrders(c, actor.(Actor), Filter{Search: c.Query("search")})
		if err != nil {
			c.JSON(500, gin.H{"error": "report could not be loaded"})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	group.GET("/material-requirements", rbac.RequirePermissions("sales_report.view"), func(c *gin.Context) {
		actor, _ := c.Get("report_actor")
		items, err := store.ListMaterialRequirements(c, actor.(Actor))
		if err != nil {
			c.JSON(500, gin.H{"error": "report could not be loaded"})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	group.GET("/customer-deliveries", rbac.RequirePermissions("customer_delivery.view"), func(c *gin.Context) {
		actor, _ := c.Get("report_actor")
		filter := Filter{Search: c.Query("search")}
		for _, field := range []struct {
			key    string
			target **time.Time
		}{{"fromDate", &filter.FromDate}, {"toDate", &filter.ToDate}} {
			if value := c.Query(field.key); value != "" {
				date, e := time.Parse("2006-01-02", value)
				if e != nil {
					c.JSON(422, gin.H{"fields": gin.H{field.key: "Use YYYY-MM-DD"}})
					return
				}
				*field.target = &date
			}
		}
		if filter.FromDate != nil && filter.ToDate != nil && filter.FromDate.After(*filter.ToDate) {
			c.JSON(422, gin.H{"fields": gin.H{"toDate": "To Date must be on or after From Date"}})
			return
		}
		items, err := store.ListCustomerDeliveries(c, actor.(Actor), filter)
		if err != nil {
			c.JSON(500, gin.H{"error": "report could not be loaded"})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
}
