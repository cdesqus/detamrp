package production

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/rbac"
)

type EntryRepository interface {
	ListEntries(context.Context, Actor) ([]Entry, error)
	GetEntry(context.Context, Actor, uuid.UUID) (Entry, error)
	EntryOptions(context.Context, Actor) (EntryOptions, error)
	CreateEntry(context.Context, Actor, EntryInput) (Entry, error)
	UpdateEntry(context.Context, Actor, uuid.UUID, EntryInput) (Entry, error)
	VoidEntry(context.Context, Actor, uuid.UUID, int, string) (Entry, error)
	ListProductionPeriods(context.Context, Actor) ([]ProductionPeriod, error)
	CloseProductionPeriod(context.Context, Actor, string, string) error
}

// Operators can post quantities without receiving restricted prices or costs.
func productionEntryPayload(c *gin.Context, value any) any {
	permissions, _ := c.Get(rbac.ContextPermissionsKey)
	granted, _ := permissions.([]string)
	if rbac.Allows(granted, "production.report") {
		return value
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var payload any
	if json.Unmarshal(raw, &payload) != nil {
		return nil
	}
	var redact func(any)
	redact = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			for _, key := range []string{"materialCost", "processCost", "processRate", "unitPrice", "cost", "rate", "totalRate", "materialEstimate", "processEstimate", "unitCost", "totalCost", "value", "totalValue"} {
				delete(typed, key)
			}
			for _, child := range typed {
				redact(child)
			}
		case []any:
			for _, child := range typed {
				redact(child)
			}
		}
	}
	redact(payload)
	return payload
}

func RegisterEntryRoutes(router *gin.Engine, s EntryRepository, authenticator Authenticator) {
	g := router.Group("/production-entries", executionAuth(authenticator))
	g.GET("", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		v, e := s.ListEntries(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, gin.H{"items": v}))
	})
	g.GET("/options", rbac.RequirePermissions("production.entry"), func(c *gin.Context) {
		v, e := s.EntryOptions(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, v))
	})
	g.GET("/:id", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		v, e := s.GetEntry(c, executionActor(c), id)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, v))
	})
	save := func(edit bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			var input EntryInput
			if !bindExecution(c, &input) {
				return
			}
			if e := input.Validate(edit); e != nil {
				respondError(c, e)
				return
			}
			var v Entry
			var e error
			code := 201
			if edit {
				id, ok := planID(c)
				if !ok {
					return
				}
				v, e = s.UpdateEntry(c, executionActor(c), id, input)
				code = 200
			} else {
				v, e = s.CreateEntry(c, executionActor(c), input)
			}
			if e != nil {
				respondError(c, e)
				return
			}
			c.JSON(code, productionEntryPayload(c, v))
		}
	}
	g.POST("", rbac.RequirePermissions("production.entry"), save(false))
	g.PUT("/:id", rbac.RequirePermissions("production.entry"), save(true))
	g.POST("/:id/void", rbac.RequirePermissions("production.void"), func(c *gin.Context) {
		id, ok := planID(c)
		if !ok {
			return
		}
		var input struct {
			Version int    `json:"version"`
			Reason  string `json:"reason"`
		}
		if !bindExecution(c, &input) {
			return
		}
		v, e := s.VoidEntry(c, executionActor(c), id, input.Version, input.Reason)
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, v))
	})
	p := router.Group("/production-periods", executionAuth(authenticator))
	p.GET("", rbac.RequirePermissions("production.view"), func(c *gin.Context) {
		v, e := s.ListProductionPeriods(c, executionActor(c))
		if e != nil {
			respondError(c, e)
			return
		}
		c.JSON(200, productionEntryPayload(c, gin.H{"items": v}))
	})
	p.POST("/:period/close", rbac.RequirePermissions("production.period"), func(c *gin.Context) {
		var input struct {
			Reason string `json:"reason"`
		}
		if !bindExecution(c, &input) {
			return
		}
		if e := s.CloseProductionPeriod(c, executionActor(c), c.Param("period"), input.Reason); e != nil {
			respondError(c, e)
			return
		}
		c.Status(204)
	})
}
