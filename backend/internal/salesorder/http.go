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
	g.GET("/sales-orders", rbac.RequirePermissions("sales_order.view"), func(c *gin.Context) {
		items, err := service.List(c, actor(c))
		if err != nil {
			c.JSON(500, gin.H{"message": err.Error()})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/sales-orders/:id", rbac.RequirePermissions("sales_order.view"), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"message": "Invalid ID"})
			return
		}
		item, err := service.Get(c, actor(c), id)
		if err != nil {
			c.JSON(404, gin.H{"message": "Sales order not found"})
			return
		}
		c.JSON(200, item)
	})
	g.GET("/sales-orders/:id/requirements", rbac.RequirePermissions("sales_order.view"), func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"message": "Invalid ID"})
			return
		}
		item, err := service.Get(c, actor(c), id)
		if err != nil {
			c.JSON(404, gin.H{"message": "Sales order not found"})
			return
		}
		if item.Status != StatusSubmitted {
			c.JSON(409, gin.H{"message": "Submit the sales order before viewing calculation"})
			return
		}
		result, err := CalculateRequirements(item.Lines)
		if err != nil {
			c.JSON(422, gin.H{"message": err.Error()})
			return
		}
		c.JSON(200, gin.H{"lines": result})
	})
	g.GET("/sales-orders/:id/requirements.pdf", rbac.RequirePermissions("sales_order.view"), func(c *gin.Context) {
		id, e := uuid.Parse(c.Param("id"))
		if e != nil {
			c.JSON(400, gin.H{"message": "Invalid ID"})
			return
		}
		item, e := service.Get(c, actor(c), id)
		if e != nil {
			c.JSON(404, gin.H{"message": "Sales order not found"})
			return
		}
		if item.Status != StatusSubmitted {
			c.JSON(409, gin.H{"message": "Submit the sales order before exporting requirements"})
			return
		}
		result, e := CalculateRequirements(item.Lines)
		if e != nil {
			c.JSON(422, gin.H{"message": e.Error()})
			return
		}
		data, e := RenderRequirementsPDF(item, result)
		if e != nil {
			c.JSON(500, gin.H{"message": "Requirements PDF could not be generated"})
			return
		}
		c.Header("Content-Disposition", `attachment; filename="`+item.Number+`-requirements.pdf"`)
		c.Data(http.StatusOK, "application/pdf", data)
	})
	g.GET("/sales-orders/:id/deliveries", rbac.RequirePermissions("customer_delivery.view"), func(c *gin.Context) {
		id, e := uuid.Parse(c.Param("id"))
		if e != nil {
			c.JSON(400, gin.H{"message": "Invalid ID"})
			return
		}
		items, e := service.ListDeliveries(c, actor(c), id)
		if e != nil {
			c.JSON(500, gin.H{"message": e.Error()})
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/customer-deliveries/:id", rbac.RequirePermissions("customer_delivery.view"), func(c *gin.Context) {
		id, e := uuid.Parse(c.Param("id"))
		if e != nil {
			c.JSON(400, gin.H{"message": "Invalid ID"})
			return
		}
		item, e := service.GetDelivery(c, actor(c), id)
		if e != nil {
			c.JSON(404, gin.H{"message": "Customer delivery not found"})
			return
		}
		c.JSON(200, item)
	})
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
	g.DELETE("/sales-orders/:id", rbac.RequirePermissions("sales_order.edit_draft"), func(c *gin.Context) {
		id, e := uuid.Parse(c.Param("id"))
		if e != nil {
			c.JSON(400, gin.H{"message": "Invalid ID"})
			return
		}
		if e = service.Delete(c, actor(c), id); e != nil {
			c.JSON(409, gin.H{"message": e.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
	g.POST("/sales-orders/:id/deliveries", rbac.RequirePermissions("customer_delivery.create"), func(c *gin.Context) {
		id, e := uuid.Parse(c.Param("id"))
		if e != nil {
			c.JSON(400, gin.H{"message": "Invalid ID"})
			return
		}
		var input DeliveryInput
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, gin.H{"message": "Invalid JSON body"})
			return
		}
		item, e := service.CreateDelivery(c, actor(c), id, input)
		if e != nil {
			c.JSON(422, gin.H{"message": e.Error()})
			return
		}
		c.JSON(201, item)
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
		c.Set(rbac.ContextPermissionsKey, user.Permissions)
		c.Next()
	}
}
func actor(c *gin.Context) Actor { return c.MustGet(actorKey).(Actor) }
