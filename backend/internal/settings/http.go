package settings

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
	"strconv"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

const actorKey = "settings_actor"

func RegisterRoutes(router *gin.Engine, s *Service, a Authenticator) {
	g := router.Group("/settings", authenticate(a))
	g.GET("/users", rbac.RequirePermissions("user.manage"), func(c *gin.Context) {
		q, ok := query(c)
		if !ok {
			return
		}
		items, total, e := s.ListUsers(c, actor(c), q)
		if fail(c, e) {
			return
		}
		if items == nil {
			items = []User{}
		}
		c.JSON(200, gin.H{"items": items, "total": total})
	})
	g.POST("/users", rbac.RequirePermissions("user.manage"), func(c *gin.Context) {
		var in UserInput
		if c.ShouldBindJSON(&in) != nil {
			badJSON(c)
			return
		}
		item, e := s.CreateUser(c, actor(c), in)
		if fail(c, e) {
			return
		}
		c.JSON(201, item)
	})
	g.PUT("/users/:id", rbac.RequirePermissions("user.manage"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var in UserInput
		if c.ShouldBindJSON(&in) != nil {
			badJSON(c)
			return
		}
		item, e := s.UpdateUser(c, actor(c), id, in)
		if fail(c, e) {
			return
		}
		c.JSON(200, item)
	})
	g.GET("/roles", rbac.RequirePermissions("role.manage"), func(c *gin.Context) {
		q, ok := query(c)
		if !ok {
			return
		}
		items, total, e := s.ListRoles(c, actor(c), q)
		if fail(c, e) {
			return
		}
		if items == nil {
			items = []Role{}
		}
		c.JSON(200, gin.H{"items": items, "total": total})
	})
	g.POST("/roles", rbac.RequirePermissions("role.manage"), func(c *gin.Context) {
		var in RoleInput
		if c.ShouldBindJSON(&in) != nil {
			badJSON(c)
			return
		}
		item, e := s.CreateRole(c, actor(c), in)
		if fail(c, e) {
			return
		}
		c.JSON(201, item)
	})
	g.PUT("/roles/:id", rbac.RequirePermissions("role.manage"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var in RoleInput
		if c.ShouldBindJSON(&in) != nil {
			badJSON(c)
			return
		}
		item, e := s.UpdateRole(c, actor(c), id, in)
		if fail(c, e) {
			return
		}
		c.JSON(200, item)
	})
	g.GET("/permissions", rbac.RequirePermissions("role.manage"), func(c *gin.Context) {
		items, e := s.ListPermissions(c, actor(c))
		if fail(c, e) {
			return
		}
		if items == nil {
			items = []Permission{}
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/approval-config", rbac.RequirePermissions("configuration.manage"), func(c *gin.Context) {
		item, e := s.GetApprovalConfig(c, actor(c))
		if fail(c, e) {
			return
		}
		c.JSON(200, item)
	})
	g.PUT("/approval-config", rbac.RequirePermissions("configuration.manage"), func(c *gin.Context) {
		var in ApprovalConfigInput
		if c.ShouldBindJSON(&in) != nil {
			badJSON(c)
			return
		}
		item, e := s.UpdateApprovalConfig(c, actor(c), in)
		if fail(c, e) {
			return
		}
		c.JSON(200, item)
	})
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
func routeID(c *gin.Context) (uuid.UUID, bool) {
	v, e := uuid.Parse(c.Param("id"))
	if e != nil {
		c.JSON(400, gin.H{"error": "invalid_id"})
		return uuid.Nil, false
	}
	return v, true
}
func query(c *gin.Context) (ListQuery, bool) {
	q := ListQuery{Search: c.Query("search")}
	if raw, ok := c.GetQuery("active"); ok {
		v, e := strconv.ParseBool(raw)
		if e != nil {
			c.JSON(400, gin.H{"error": "invalid_filter"})
			return q, false
		}
		q.Active = &v
	}
	if raw := c.Query("limit"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil {
			c.JSON(400, gin.H{"error": "invalid_filter"})
			return q, false
		}
		q.Limit = v
	}
	return q, true
}
func badJSON(c *gin.Context) {
	c.JSON(400, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
}
func fail(c *gin.Context, e error) bool {
	if e == nil {
		return false
	}
	var v ValidationError
	var x ConflictError
	var n NotFoundError
	switch {
	case errors.As(e, &v):
		c.JSON(400, gin.H{"error": "validation_failed", "message": "Please correct the highlighted fields", "fields": v.Fields})
	case errors.As(e, &x):
		c.JSON(409, gin.H{"error": "conflict", "message": "Data conflicts with an existing setting", "fields": x.Fields})
	case errors.As(e, &n):
		c.JSON(404, gin.H{"error": "not_found", "message": n.Error()})
	default:
		c.JSON(500, gin.H{"error": "internal_error", "message": "Request could not be completed"})
	}
	return true
}

var _ = http.StatusOK
