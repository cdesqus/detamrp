package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/dashboard"
	"order-stock/backend/internal/emailing"
	"order-stock/backend/internal/inventory"
	"order-stock/backend/internal/masterdata"
	"order-stock/backend/internal/outgoing"
	"order-stock/backend/internal/purchaseorder"
	"order-stock/backend/internal/receiving"
	"order-stock/backend/internal/report"
	"order-stock/backend/internal/settings"
)

type Authenticator interface {
	Login(ctx context.Context, username, password string) (auth.LoginResult, error)
	Authenticate(ctx context.Context, token string) (auth.User, error)
	Logout(ctx context.Context, token string) error
}

type serverConfig struct {
	authenticator        Authenticator
	cookieSecure         bool
	unitService          *masterdata.UnitService
	categoryService      *masterdata.CategoryService
	packingService       *masterdata.PackingService
	plantService         *masterdata.PlantService
	supplierService      *masterdata.SupplierService
	rawMaterialService   *masterdata.RawMaterialService
	settingsService      *settings.Service
	purchaseOrderService *purchaseorder.Service
	receivingStore       *receiving.Store
	outgoingStore        *outgoing.Store
	inventoryStore       *inventory.Store
	reportStore          *report.Store
	emailService         *emailing.Service
	dashboardStore       *dashboard.Store
}

type ServerOption func(*serverConfig)

func WithAuthenticator(authenticator Authenticator) ServerOption {
	return func(config *serverConfig) { config.authenticator = authenticator }
}

func WithSecureCookies() ServerOption {
	return func(config *serverConfig) { config.cookieSecure = true }
}

func WithUnitService(service *masterdata.UnitService) ServerOption {
	return func(config *serverConfig) { config.unitService = service }
}
func WithCategoryService(service *masterdata.CategoryService) ServerOption {
	return func(config *serverConfig) { config.categoryService = service }
}
func WithPackingService(service *masterdata.PackingService) ServerOption {
	return func(config *serverConfig) { config.packingService = service }
}
func WithPlantService(service *masterdata.PlantService) ServerOption {
	return func(config *serverConfig) { config.plantService = service }
}
func WithSupplierService(service *masterdata.SupplierService) ServerOption {
	return func(config *serverConfig) { config.supplierService = service }
}
func WithRawMaterialService(service *masterdata.RawMaterialService) ServerOption {
	return func(c *serverConfig) { c.rawMaterialService = service }
}
func WithSettingsService(service *settings.Service) ServerOption {
	return func(c *serverConfig) { c.settingsService = service }
}
func WithPurchaseOrderService(service *purchaseorder.Service) ServerOption {
	return func(c *serverConfig) { c.purchaseOrderService = service }
}
func WithReceivingStore(store *receiving.Store) ServerOption {
	return func(c *serverConfig) { c.receivingStore = store }
}
func WithOutgoingStore(store *outgoing.Store) ServerOption {
	return func(c *serverConfig) { c.outgoingStore = store }
}
func WithInventoryStore(store *inventory.Store) ServerOption {
	return func(c *serverConfig) { c.inventoryStore = store }
}
func WithReportStore(store *report.Store) ServerOption {
	return func(c *serverConfig) { c.reportStore = store }
}
func WithEmailService(service *emailing.Service) ServerOption {
	return func(c *serverConfig) { c.emailService = service }
}
func WithDashboardStore(store *dashboard.Store) ServerOption {
	return func(c *serverConfig) { c.dashboardStore = store }
}

func NewServer(options ...ServerOption) http.Handler {
	config := serverConfig{}
	for _, option := range options {
		option(&config)
	}
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	if config.authenticator != nil {
		registerAuthRoutes(router, config)
		if config.unitService != nil {
			masterdata.RegisterUnitRoutes(router, config.unitService, config.authenticator)
		}
		if config.categoryService != nil {
			masterdata.RegisterCategoryRoutes(router, config.categoryService, config.authenticator)
		}
		if config.packingService != nil {
			masterdata.RegisterPackingRoutes(router, config.packingService, config.authenticator)
		}
		if config.plantService != nil {
			masterdata.RegisterPlantRoutes(router, config.plantService, config.authenticator)
		}
		if config.supplierService != nil {
			masterdata.RegisterSupplierRoutes(router, config.supplierService, config.authenticator)
		}
		if config.rawMaterialService != nil {
			masterdata.RegisterRawMaterialRoutes(router, config.rawMaterialService, config.authenticator)
		}
		if config.settingsService != nil {
			settings.RegisterRoutes(router, config.settingsService, config.authenticator)
		}
		if config.purchaseOrderService != nil {
			purchaseorder.RegisterRoutes(router, config.purchaseOrderService, config.authenticator)
		}
		if config.receivingStore != nil {
			receiving.RegisterRoutes(router, config.receivingStore, config.authenticator)
		}
		if config.outgoingStore != nil {
			outgoing.RegisterRoutes(router, config.outgoingStore, config.authenticator)
		}
		if config.inventoryStore != nil {
			inventory.RegisterRoutes(router, config.inventoryStore, config.authenticator)
		}
		if config.reportStore != nil {
			report.RegisterRoutes(router, config.reportStore, config.authenticator)
		}
		if config.emailService != nil && config.purchaseOrderService != nil {
			emailing.RegisterRoutes(router, config.emailService, config.purchaseOrderService, config.authenticator)
		}
		if config.dashboardStore != nil {
			dashboard.RegisterRoutes(router, config.dashboardStore, config.authenticator)
		}
	}
	return router
}
