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
	"order-stock/backend/internal/database"
	"order-stock/backend/internal/masterdata"
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
	measurementService := masterdata.NewMeasurementService(masterDataStore)
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8091"
	}
	log.Printf("API listening on %s", address)
	if err := http.ListenAndServe(address, api.NewServer(api.WithAuthenticator(authService), api.WithMeasurementService(measurementService))); err != nil {
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
