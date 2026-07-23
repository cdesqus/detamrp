package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/inventory"
	"order-stock/backend/internal/masterdata"
	"order-stock/backend/internal/outgoing"
	"order-stock/backend/internal/purchaseorder"
	"order-stock/backend/internal/receiving"
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
	measurementService   *masterdata.MeasurementService
	supplierService      *masterdata.SupplierService
	rawMaterialService   *masterdata.RawMaterialService
	settingsService      *settings.Service
	purchaseOrderService *purchaseorder.Service
	receivingStore       *receiving.Store
	outgoingStore        *outgoing.Store
	inventoryStore       *inventory.Store
}

type ServerOption func(*serverConfig)

func WithAuthenticator(authenticator Authenticator) ServerOption {
	return func(config *serverConfig) { config.authenticator = authenticator }
}

func WithSecureCookies() ServerOption {
	return func(config *serverConfig) { config.cookieSecure = true }
}

func WithMeasurementService(service *masterdata.MeasurementService) ServerOption {
	return func(config *serverConfig) { config.measurementService = service }
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
		if config.measurementService != nil {
			masterdata.RegisterMeasurementRoutes(router, config.measurementService, config.authenticator)
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
	}
	return router
}
