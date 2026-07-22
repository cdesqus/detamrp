package outgoing

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

const actorKey = "outgoing_actor"

func RegisterRoutes(r *gin.Engine, s *Store, a Authenticator) {
	mw := func(c *gin.Context) {
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
	sessions := r.Group("/outgoing-sessions", mw)
	sessions.POST("", rbac.RequirePermissions("inventory.consume"), func(c *gin.Context) {
		var in struct{ Destination, Notes string }
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "invalid input"})
			return
		}
		x, e := s.Create(c, actor(c), in.Destination, in.Notes)
		if write(c, e) {
			return
		}
		c.JSON(201, x)
	})
	sessions.GET("/:id", rbac.RequirePermissions("inventory.view"), func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		x, e := s.GetSession(c, actor(c), id)
		if write(c, e) {
			return
		}
		c.JSON(200, x)
	})
	sessions.POST("/:id/scans", rbac.RequirePermissions("inventory.consume"), func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		var in struct {
			KanbanID string `json:"kanbanId"`
		}
		if c.ShouldBindJSON(&in) != nil {
			c.JSON(400, gin.H{"error": "kanbanId required"})
			return
		}
		x, e := s.Scan(c, actor(c), id, in.KanbanID)
		if write(c, e) {
			return
		}
		c.JSON(200, x)
	})
	sessions.DELETE("/:id/scans/:lot", rbac.RequirePermissions("inventory.consume"), func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		lot, e := uuid.Parse(c.Param("lot"))
		if e != nil {
			c.JSON(400, gin.H{"error": "invalid lot"})
			return
		}
		if write(c, s.Remove(c, actor(c), id, lot)) {
			return
		}
		c.Status(204)
	})
	sessions.POST("/:id/complete", rbac.RequirePermissions("inventory.consume"), func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		x, e := s.Complete(c, actor(c), id)
		if write(c, e) {
			return
		}
		c.JSON(200, x)
	})
	docs := r.Group("/outgoing-material", mw, rbac.RequirePermissions("inventory.view"))
	docs.GET("", func(c *gin.Context) {
		x, e := s.List(c, actor(c))
		if write(c, e) {
			return
		}
		if x == nil {
			x = []Document{}
		}
		c.JSON(200, gin.H{"items": x})
	})
	docs.GET("/:id", func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		x, e := s.GetDocument(c, actor(c), id)
		if write(c, e) {
			return
		}
		c.JSON(200, x)
	})
	docs.GET("/:id/document.pdf", func(c *gin.Context) {
		id, ok := parse(c)
		if !ok {
			return
		}
		x, e := s.GetDocument(c, actor(c), id)
		if write(c, e) {
			return
		}
		b, e := RenderPDF(x)
		if write(c, e) {
			return
		}
		c.Header("Cache-Control", "private, no-store")
		c.Header("Content-Disposition", `inline; filename="`+x.DocumentNumber+`.pdf"`)
		c.Data(http.StatusOK, "application/pdf", b)
	})
}
