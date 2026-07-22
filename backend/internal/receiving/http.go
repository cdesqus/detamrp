package receiving

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

const actorKey = "receiving_actor"

func RegisterRoutes(router *gin.Engine, store *Store, authn Authenticator) {
	mw := func(c *gin.Context) {
		token, e := c.Cookie("session")
		if e != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
			return
		}
		u, e := authn.Authenticate(c.Request.Context(), token)
		if e != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "authentication required"})
			return
		}
		c.Set(actorKey, Actor{TenantID: u.TenantID, UserID: u.ID, DisplayName: u.DisplayName})
		c.Set(rbac.ContextPermissionsKey, u.Permissions)
		c.Next()
	}
	actor := func(c *gin.Context) Actor { v, _ := c.Get(actorKey); return v.(Actor) }
	parse := func(c *gin.Context) (uuid.UUID, bool) {
		id, e := uuid.Parse(c.Param("id"))
		if e != nil {
			c.JSON(400, gin.H{"error": "invalid id"})
			return uuid.Nil, false
		}
		return id, true
	}
	write := func(c *gin.Context, e error) bool {
		if e == nil {
			return false
		}
		code := 500
		if errors.Is(e, ErrNotFound) {
			code = 404
		} else if errors.Is(e, ErrConflict) {
			code = 409
		} else if errors.Is(e, ErrValidation) {
			code = 422
		}
		c.JSON(code, gin.H{"error": e.Error()})
		return true
	}
	options := router.Group("/receiving-options", mw, rbac.RequirePermissions("receiving.view"))
	options.GET("", func(c *gin.Context) {
		x, e := store.Options(c, actor(c), strings.TrimSpace(c.Query("search")))
		if write(c, e) {
			return
		}
		if x == nil {
			x = []Option{}
		}
		c.JSON(200, gin.H{"items": x})
	})
	sessions := router.Group("/receiving-sessions", mw)
	sessions.GET("", rbac.RequirePermissions("receiving.view"), func(c *gin.Context) {
		x, e := store.ListOpenSessions(c, actor(c))
		if write(c, e) {
			return
		}
		if x == nil {
			x = []Session{}
		}
		c.JSON(http.StatusOK, gin.H{"items": x})
	})
	sessions.POST("", rbac.RequirePermissions("receiving.create"), func(c *gin.Context) {
		var in struct {
			DeliveryNoteID uuid.UUID `json:"deliveryNoteId" binding:"required"`
		}
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "deliveryNoteId is required"})
			return
		}
		x, e := store.CreateSession(c, actor(c), in.DeliveryNoteID)
		if write(c, e) {
			return
		}
		c.JSON(201, x)
	})
	sessions.GET("/:id", rbac.RequirePermissions("receiving.view"), func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		x, e := store.GetSession(c, actor(c), id)
		if write(c, e) {
			return
		}
		c.JSON(200, x)
	})
	sessions.POST("/:id/scans", rbac.RequirePermissions("receiving.create"), func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		var in struct {
			KanbanID string `json:"kanbanId"`
		}
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "kanbanId is required"})
			return
		}
		x, e := store.Scan(c, actor(c), id, in.KanbanID)
		if write(c, e) {
			return
		}
		c.JSON(200, x)
	})
	sessions.DELETE("/:id/scans/:lot", rbac.RequirePermissions("receiving.create"), func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		lot, e := uuid.Parse(c.Param("lot"))
		if e != nil {
			c.JSON(400, gin.H{"error": "invalid lot"})
			return
		}
		if write(c, store.RemoveScan(c, actor(c), id, lot)) {
			return
		}
		c.Status(204)
	})
	for _, action := range []string{"pause", "resume", "cancel"} {
		action := action
		sessions.POST("/:id/"+action, rbac.RequirePermissions("receiving.create"), func(c *gin.Context) {
			id, ok := parse(c)
			if !ok {
				return
			}
			status := "PAUSED"
			reason := ""
			if action == "resume" {
				status = "ACTIVE"
			}
			if action == "cancel" {
				status = "CANCELLED"
				var in struct {
					Reason string `json:"reason"`
				}
				_ = c.ShouldBindJSON(&in)
				reason = strings.TrimSpace(in.Reason)
			}
			if write(c, store.SetStatus(c, actor(c), id, status, reason)) {
				return
			}
			x, e := store.GetSession(c, actor(c), id)
			if write(c, e) {
				return
			}
			c.JSON(200, x)
		})
	}
	sessions.POST("/:id/complete", rbac.RequirePermissions("receiving.submit"), func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		x, e := store.Complete(c, actor(c), id)
		if write(c, e) {
			return
		}
		c.JSON(200, x)
	})
	receivings := router.Group("/receivings", mw, rbac.RequirePermissions("receiving.view"))
	receivings.GET("", func(c *gin.Context) {
		x, e := store.List(c, actor(c))
		if write(c, e) {
			return
		}
		if x == nil {
			x = []Receiving{}
		}
		c.JSON(200, gin.H{"items": x})
	})
	receivings.GET("/:id", func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		x, e := store.GetReceiving(c, actor(c), id)
		if write(c, e) {
			return
		}
		c.JSON(200, x)
	})
	receivings.GET("/:id/document.pdf", func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		x, e := store.GetReceiving(c, actor(c), id)
		if write(c, e) {
			return
		}
		b, e := RenderPDF(x)
		if write(c, e) {
			return
		}
		c.Header("Cache-Control", "private, no-store")
		c.Header("Content-Disposition", `inline; filename="`+x.ReceivingNumber+`.pdf"`)
		c.Data(http.StatusOK, "application/pdf", b)
	})
}
