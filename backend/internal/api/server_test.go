package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"order-stock/backend/internal/inventory"
)

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewServer().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("expected healthy response, got %q", response["status"])
	}
}

func TestWithInventoryStoreRegistersConfiguration(t *testing.T) {
	store := &inventory.Store{}
	config := serverConfig{}

	WithInventoryStore(store)(&config)

	if config.inventoryStore != store {
		t.Fatal("inventory store was not registered")
	}
}
