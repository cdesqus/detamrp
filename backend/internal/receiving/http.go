package receiving

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

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
		if errors.Is(e, ErrDeliveryNoteInvalid) {
			code = 422
		} else if errors.Is(e, ErrDeliveryNoteFullyReceived) || errors.Is(e, ErrDeliveryNoteInProgress) {
			code = 409
		} else if errors.Is(e, ErrNotFound) {
			code = 404
		} else if errors.Is(e, ErrConflict) {
			code = 409
		} else if errors.Is(e, ErrValidation) {
			code = 422
		}
		body := gin.H{"error": e.Error()}
		if businessCode := receivingErrorCode(e); businessCode != "" {
			body["code"] = businessCode
		}
		c.JSON(code, body)
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
			DeliveryNoteNumber string `json:"deliveryNoteNumber" binding:"required"`
		}
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "deliveryNoteNumber is required", "code": "DN_INVALID"})
			return
		}
		x, e := store.CreateSession(c, actor(c), in.DeliveryNoteNumber)
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
		query, ok := receivingListQuery(c)
		if !ok {
			return
		}
		x, e := store.List(c, actor(c), query)
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

func receivingListQuery(c *gin.Context) (ListQuery, bool) {
	var query ListQuery
	if raw := c.Query("supplierId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_filter", "fields": gin.H{"supplierId": "Invalid supplier"}})
			return query, false
		}
		query.SupplierID = id
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	if jakarta == nil {
		jakarta = time.FixedZone("Asia/Jakarta", 7*60*60)
	}
	for key, destination := range map[string]*time.Time{"createdFrom": &query.CreatedFrom, "createdTo": &query.CreatedToExclusive} {
		raw := c.Query(key)
		if raw == "" {
			continue
		}
		value, err := time.ParseInLocation("2006-01-02", raw, jakarta)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_filter", "fields": gin.H{key: "Use YYYY-MM-DD"}})
			return query, false
		}
		if key == "createdTo" {
			value = value.AddDate(0, 0, 1)
		}
		*destination = value
	}
	if !query.CreatedFrom.IsZero() && !query.CreatedToExclusive.IsZero() && !query.CreatedFrom.Before(query.CreatedToExclusive) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_filter", "fields": gin.H{"createdTo": "Must be on or after createdFrom"}})
		return query, false
	}
	return query, true
}
