package inventory

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

const actorKey = "inventory_actor"

func RegisterRoutes(router *gin.Engine, store *Store, authenticator Authenticator) {
	authenticate := func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		user, err := authenticator.Authenticate(c, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Set(actorKey, Actor{TenantID: user.TenantID, UserID: user.ID})
		c.Set(rbac.ContextPermissionsKey, user.Permissions)
		c.Next()
	}
	actor := func(c *gin.Context) Actor {
		value, _ := c.Get(actorKey)
		return value.(Actor)
	}
	writeError := func(c *gin.Context, err error) {
		switch {
		case errors.Is(err, ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, ErrValidation):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "inventory request failed"})
		}
	}

	group := router.Group("/inventory", authenticate, rbac.RequirePermissions("inventory.view"))
	group.GET("/stock", func(c *gin.Context) {
		filters, err := parseFilters(c)
		if err != nil {
			writeError(c, err)
			return
		}
		response, err := store.ListStock(c, actor(c), filters)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
	group.GET("/stock/:id/kanbans", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid raw material id"})
			return
		}
		response, err := store.ListKanbans(c, actor(c), id)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func parseFilters(c *gin.Context) (Filters, error) {
	filters := Filters{Search: strings.TrimSpace(c.Query("search"))}
	if supplier := strings.TrimSpace(c.Query("supplierId")); supplier != "" {
		id, err := uuid.Parse(supplier)
		if err != nil {
			return Filters{}, ErrValidation
		}
		filters.SupplierID = &id
	}
	if status := strings.ToUpper(strings.TrimSpace(c.Query("status"))); status != "" {
		switch status {
		case "IN_STOCK", "LOW_STOCK", "OUT_OF_STOCK":
			filters.Status = status
		default:
			return Filters{}, ErrValidation
		}
	}
	return filters, nil
}
