package production

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"net/http"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/rbac"
)

type Authenticator interface {
	Authenticate(context.Context, string) (auth.User, error)
}

func RegisterRoutes(router *gin.Engine, s *PlanService, authenticator Authenticator) {
	g := router.Group("/production-plans", func(c *gin.Context) {
		token, e := c.Cookie("session")
		if e != nil {
			c.AbortWithStatusJSON(401, gin.H{"message": "Authentication is required"})
			return
		}
		u, e := authenticator.Authenticate(c, token)
		if e != nil {
			c.AbortWithStatusJSON(401, gin.H{"message": "Authentication is required"})
			return
		}
		c.Set("production_actor", Actor{u.TenantID, u.ID})
		c.Set(rbac.ContextPermissionsKey, u.Permissions)
		c.Next()
	})
	actor := func(c *gin.Context) Actor { return c.MustGet("production_actor").(Actor) }
	g.GET("", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		items, e := s.List(c, actor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, gin.H{"items": items})
	})
	g.GET("/options", rbac.RequirePermissions("production.plan"), func(c *gin.Context) {
		items, e := s.Options(c, actor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, items)
	})
	g.GET("/:id", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		p, e := s.Get(c, actor(c), id)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, p)
	})
	save := func(update bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			var id uuid.UUID
			if update {
				var ok bool
				id, ok = planID(c)
				if !ok {
					return
				}
			}
			var p Plan
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024*1024)
			if c.ShouldBindJSON(&p) != nil {
				c.JSON(400, gin.H{"message": "Invalid planning body"})
				return
			}
			if e := ValidatePlanInput(p); e != nil {
				c.JSON(422, gin.H{"message": e.Error()})
				return
			}
			var result Plan
			var e error
			code := 201
			if update {
				result, e = s.Update(c, actor(c), id, p)
				code = 200
			} else {
				result, e = s.Create(c, actor(c), p)
			}
			if e != nil {
				respondError(c, e)
				return
			}
			c.JSON(code, result)
		}
	}
	g.POST("", rbac.RequirePermissions("production.plan"), save(false))
	g.PUT("/:id", rbac.RequirePermissions("production.plan"), save(true))
	g.DELETE("/:id", rbac.RequirePermissions("production.plan"), func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		if e := s.Delete(c, actor(c), id); e != nil {
			respondError(c, e)
			return
		}
		c.Status(204)
	})
	for _, action := range []struct {
		path, permission string
		run              func(context.Context, Actor, uuid.UUID) (Plan, error)
	}{{"approve", "production.plan", s.Approve}, {"close", "production.plan", s.Close}, {"orders", "production.order", s.CreateOrders}} {
		g.POST("/:id/"+action.path, rbac.RequirePermissions(action.permission), func(c *gin.Context) {
			id, ok := planID(c)
			if !ok {
				return
			}
			p, e := action.run(c, actor(c), id)
			if e != nil {
				respondError(c, e)
				return
			}
			c.JSON(200, p)
		})
	}
}
func planID(c *gin.Context) (uuid.UUID, bool) {
	id, e := uuid.Parse(c.Param("id"))
	if e != nil {
		c.JSON(400, gin.H{"message": "Invalid planning ID"})
		return uuid.Nil, false
	}
	return id, true
}
func respondError(c *gin.Context, e error) {
	var validation ValidationError
	switch {
	case errors.As(e, &validation):
		c.JSON(422, gin.H{"message": e.Error()})
	case errors.Is(e, ErrNotFound):
		c.JSON(404, gin.H{"message": e.Error()})
	case errors.Is(e, ErrConflict):
		c.JSON(409, gin.H{"message": e.Error()})
	case errors.Is(e, ErrInvalidReference):
		c.JSON(422, gin.H{"message": e.Error()})
	default:
		// The cause stays in the server log; the client gets a stable message
		// that names the module it came from.
		log.Printf("production %s %s: %v", c.Request.Method, c.Request.URL.Path, e)
		c.JSON(500, gin.H{"message": "This production request could not be processed"})
	}
}
