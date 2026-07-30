package masterdata

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/rbac"
)

type referenceListFunc[T any] func(context.Context, Actor, ListQuery) ([]T, int, error)
type referenceGetFunc[T any] func(context.Context, Actor, uuid.UUID) (T, error)
type referenceCreateFunc[T any, I any] func(context.Context, Actor, I) (T, error)
type referenceUpdateFunc[T any, I any] func(context.Context, Actor, uuid.UUID, I) (T, error)

func RegisterCategoryRoutes(router *gin.Engine, service *CategoryService, authenticator Authenticator) {
	registerReferenceRoutes(router, authenticator, "/categories", service.List, service.Get, service.Create, service.Update)
}

func RegisterPackingRoutes(router *gin.Engine, service *PackingService, authenticator Authenticator) {
	registerReferenceRoutes(router, authenticator, "/packings", service.List, service.Get, service.Create, service.Update)
}

func RegisterPlantRoutes(router *gin.Engine, service *PlantService, authenticator Authenticator) {
	registerReferenceRoutes(router, authenticator, "/plants", service.List, service.Get, service.Create, service.Update)
}

func registerReferenceRoutes[T any, I any](
	router *gin.Engine,
	authenticator Authenticator,
	path string,
	list referenceListFunc[T],
	get referenceGetFunc[T],
	create referenceCreateFunc[T, I],
	update referenceUpdateFunc[T, I],
) {
	group := router.Group("/master-data", authenticateMasterData(authenticator))
	group.GET(path, rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		query, ok := listQuery(c)
		if !ok {
			return
		}
		items, total, err := list(c, actorFrom(c), query)
		if writeError(c, err) {
			return
		}
		if items == nil {
			items = []T{}
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})
	group.GET(path+"/:id", rbac.RequirePermissions("master_data.view"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		item, err := get(c, actorFrom(c), id)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusOK, item)
	})
	group.POST(path, rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		var input I
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
			return
		}
		item, err := create(c, actorFrom(c), input)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusCreated, item)
	})
	group.PUT(path+"/:id", rbac.RequirePermissions("master_data.manage"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var input I
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "Invalid JSON body"})
			return
		}
		item, err := update(c, actorFrom(c), id, input)
		if writeError(c, err) {
			return
		}
		c.JSON(http.StatusOK, item)
	})
}
