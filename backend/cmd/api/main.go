package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"order-stock/backend/internal/api"
	"order-stock/backend/internal/auth"
	"order-stock/backend/internal/dashboard"
	"order-stock/backend/internal/database"
	"order-stock/backend/internal/emailing"
	"order-stock/backend/internal/inventory"
	"order-stock/backend/internal/masterdata"
	"order-stock/backend/internal/outgoing"
	"order-stock/backend/internal/purchaseorder"
	"order-stock/backend/internal/receiving"
	"order-stock/backend/internal/report"
	"order-stock/backend/internal/settings"
)

func main() {
	ctx := context.Background()
	db, err := database.Open(ctx, requiredEnv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	tenantID, err := uuid.Parse(requiredEnv("DEFAULT_TENANT_ID"))
	if err != nil {
		log.Fatalf("invalid DEFAULT_TENANT_ID: %v", err)
	}
	store := auth.NewSQLStore(db, tenantID)
	if err := store.EnsureInitialAdmin(ctx, requiredEnv("INITIAL_ADMIN_USERNAME"), requiredEnv("INITIAL_ADMIN_PASSWORD")); err != nil {
		log.Fatal(err)
	}
	authService := auth.NewService(store, 12*time.Hour)
	masterDataStore := masterdata.NewSQLStore(db)
	unitService := masterdata.NewUnitService(masterDataStore)
	categoryService := masterdata.NewCategoryService(masterDataStore)
	packingService := masterdata.NewPackingService(masterDataStore)
	plantService := masterdata.NewPlantService(masterDataStore)
	supplierService := masterdata.NewSupplierService(masterDataStore)
	rawMaterialService := masterdata.NewRawMaterialService(masterDataStore)
	settingsService := settings.NewService(settings.NewSQLStore(db))
	purchaseOrderService := purchaseorder.NewService(purchaseorder.NewSQLStore(db))
	receivingStore := receiving.NewStore(db)
	outgoingStore := outgoing.NewStore(db)
	inventoryStore := inventory.NewStore(db)
	reportStore := report.NewStore(db)
	dashboardStore := dashboard.NewStore(db)
	secretBox, err := emailing.NewSecretBox(requiredEnv("EMAIL_ENCRYPTION_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	appBaseURL := os.Getenv("APP_BASE_URL")
	if appBaseURL == "" {
		appBaseURL = "http://localhost:3019"
	}
	emailService := emailing.NewService(emailing.NewStore(db), secretBox, appBaseURL)
	purchaseOrderService.SetDecisionNotifier(func(ctx context.Context, actor purchaseorder.Actor, approvalID uuid.UUID) error {
		return emailService.SendDecisionResult(ctx, emailing.Actor{TenantID: actor.TenantID, UserID: actor.UserID}, approvalID)
	})
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8091"
	}
	log.Printf("API listening on %s", address)
	if err := http.ListenAndServe(address, api.NewServer(api.WithAuthenticator(authService), api.WithUnitService(unitService), api.WithCategoryService(categoryService), api.WithPackingService(packingService), api.WithPlantService(plantService), api.WithSupplierService(supplierService), api.WithRawMaterialService(rawMaterialService), api.WithSettingsService(settingsService), api.WithPurchaseOrderService(purchaseOrderService), api.WithReceivingStore(receivingStore), api.WithOutgoingStore(outgoingStore), api.WithInventoryStore(inventoryStore), api.WithReportStore(reportStore), api.WithEmailService(emailService), api.WithDashboardStore(dashboardStore))); err != nil {
		log.Fatal(err)
	}
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}
