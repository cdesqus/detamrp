package emailing

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/purchaseorder"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

const actorKey = "email_actor"

func RegisterRoutes(router *gin.Engine, s *Service, po *purchaseorder.Service, a Authenticator) {
	g := router.Group("/email", authenticate(a))
	g.GET("/smtp-settings", rbac.RequirePermissions("smtp_settings.view"), func(c *gin.Context) { x, e := s.GetSettings(c, actor(c)); write(c, x, e) })
	g.PUT("/smtp-settings", rbac.RequirePermissions("smtp_settings.manage"), func(c *gin.Context) {
		var in SMTPSettingsInput
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "invalid_request"})
			return
		}
		x, fields, e := s.UpdateSettings(c, actor(c), in)
		if len(fields) > 0 {
			c.JSON(400, gin.H{"error": "validation_failed", "fields": fields})
			return
		}
		write(c, x, e)
	})
	g.POST("/smtp-test", rbac.RequirePermissions("smtp_settings.test"), func(c *gin.Context) {
		var in struct {
			To string `json:"to"`
		}
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "invalid_request"})
			return
		}
		if e := s.Test(c, actor(c), in.To); e != nil {
			c.JSON(422, gin.H{"error": "delivery_failed", "message": e.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "SENT"})
	})
	g.GET("/logs", rbac.RequirePermissions("email_log.view"), func(c *gin.Context) {
		items, e := s.ListLogs(c, actor(c), c.Query("search"))
		if items == nil {
			items = []EmailLog{}
		}
		if e != nil {
			c.JSON(500, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(200, gin.H{"items": items, "total": len(items)})
	})
	g.POST("/purchase-orders/:id/approval", rbac.RequirePermissions("po.submit"), func(c *gin.Context) {
		id, ok := id(c)
		if !ok {
			return
		}
		if e := s.SendApproval(c, actor(c), id); e != nil {
			c.JSON(422, gin.H{"error": "delivery_failed", "message": e.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "SENT"})
	})
	g.POST("/purchase-orders/:id/supplier", rbac.RequirePermissions("po.view"), func(c *gin.Context) {
		id, ok := id(c)
		if !ok {
			return
		}
		pa := purchaseorder.Actor{TenantID: actor(c).TenantID, UserID: actor(c).UserID}
		order, e := po.Get(c, pa, id)
		if e != nil {
			c.JSON(422, gin.H{"error": "delivery_failed", "message": e.Error()})
			return
		}
		if order.Status != purchaseorder.StatusApproved && order.Status != purchaseorder.StatusPartiallyReceived && order.Status != purchaseorder.StatusFullyReceived {
			c.JSON(409, gin.H{"error": "conflict", "message": "Only approved orders can be sent to suppliers"})
			return
		}
		pdfPO, e := po.PurchaseOrderPDF(c, pa, id, true)
		if e == nil {
			var dn, labels purchaseorder.PDFDocument
			dn, e = po.DeliveryNotePDF(c, pa, id)
			if e == nil {
				labels, e = po.KanbanLabelsPDF(c, pa, id)
				if e == nil {
					e = s.SendSupplier(c, actor(c), order, pdfPO, dn, labels)
				}
			}
		}
		if e != nil {
			c.JSON(422, gin.H{"error": "delivery_failed", "message": e.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "SENT"})
	})
	router.GET("/public/email-approval", func(c *gin.Context) { publicDecisionPage(c, s, po) })
	router.POST("/public/email-approval/reject", func(c *gin.Context) { publicReject(c, s, po) })
}
func authenticate(a Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, e := c.Cookie("session")
		if e != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
			return
		}
		u, e := a.Authenticate(c, token)
		if e != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
			return
		}
		c.Set(actorKey, Actor{TenantID: u.TenantID, UserID: u.ID})
		c.Set(rbac.ContextPermissionsKey, u.Permissions)
		c.Next()
	}
}
func actor(c *gin.Context) Actor { v, _ := c.Get(actorKey); return v.(Actor) }
func id(c *gin.Context) (uuid.UUID, bool) {
	x, e := uuid.Parse(c.Param("id"))
	if e != nil {
		c.JSON(400, gin.H{"error": "invalid_id"})
		return uuid.Nil, false
	}
	return x, true
}
func write(c *gin.Context, value any, e error) {
	if e != nil {
		c.JSON(500, gin.H{"error": "internal_error", "message": "Request could not be completed"})
		return
	}
	c.JSON(200, value)
}
func tokenContext(c *gin.Context, s *Service) (TokenContext, []byte, error) {
	tenant, e := uuid.Parse(c.Query("tenant"))
	if e != nil {
		return TokenContext{}, nil, e
	}
	token := c.Query("token")
	x, e := s.store.ResolveToken(c, tenant, TokenHash(token))
	return x, TokenHash(token), e
}
func publicDecisionPage(c *gin.Context, s *Service, po *purchaseorder.Service) {
	x, hash, e := tokenContext(c, s)
	if e != nil {
		publicHTML(c, "Approval link unavailable", e.Error())
		return
	}
	if c.Query("decision") == "reject" {
		action := fmt.Sprintf("/api/public/email-approval/reject?tenant=%s&token=%s", x.TenantID, c.Query("token"))
		c.Data(200, "text/html; charset=utf-8", []byte(`<!doctype html><html><body style="font-family:Arial;padding:40px;max-width:520px;margin:auto"><h1>Reject Purchase Order</h1><form method="post" action="`+html.EscapeString(action)+`"><label>Rejection reason</label><br><textarea name="reason" required style="width:100%;height:110px;margin:10px 0"></textarea><br><button style="padding:10px 18px">Confirm Rejection</button></form></body></html>`))
		return
	}
	_, e = po.Approve(c, purchaseorder.Actor{TenantID: x.TenantID, UserID: x.ApproverUserID, DisplayName: x.ApproverName}, x.ApprovalID, purchaseorder.DecisionInput{})
	if e == nil {
		e = s.store.MarkTokenUsed(c, x.TenantID, hash)
	}
	if e != nil {
		publicHTML(c, "Approval could not be completed", e.Error())
		return
	}
	publicHTML(c, "Purchase Order Approved", "The approval has been recorded successfully.")
}
func publicReject(c *gin.Context, s *Service, po *purchaseorder.Service) {
	x, hash, e := tokenContext(c, s)
	if e == nil {
		e = c.Request.ParseForm()
	}
	reason := strings.TrimSpace(c.Request.FormValue("reason"))
	if e == nil && reason == "" {
		e = fmt.Errorf("rejection reason is required")
	}
	if e == nil {
		_, e = po.Reject(c, purchaseorder.Actor{TenantID: x.TenantID, UserID: x.ApproverUserID, DisplayName: x.ApproverName}, x.ApprovalID, purchaseorder.DecisionInput{Reason: reason})
	}
	if e == nil {
		e = s.store.MarkTokenUsed(c, x.TenantID, hash)
	}
	if e != nil {
		publicHTML(c, "Rejection could not be completed", e.Error())
		return
	}
	publicHTML(c, "Purchase Order Rejected", "The rejection has been recorded successfully.")
}
func publicHTML(c *gin.Context, title, message string) {
	c.Data(200, "text/html; charset=utf-8", []byte(`<!doctype html><html><body style="margin:0;background:#f4f4f5;font-family:Arial"><div style="max-width:520px;margin:80px auto;background:white;border:1px solid #e4e4e7;border-radius:12px;padding:32px"><b>ORDER STOCK</b><h1>`+html.EscapeString(title)+`</h1><p>`+html.EscapeString(message)+`</p></div></body></html>`))
}
