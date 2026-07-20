package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"order-stock/backend/internal/database"
)

func TestSQLStoreCanAuthenticateSeededAdministrator(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := database.Open(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	store := NewSQLStore(db, uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	user, err := store.FindUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatalf("find seeded administrator: %v", err)
	}
	if !VerifyPassword(user.PasswordHash, "change-me-before-production") {
		t.Fatal("stored administrator password hash does not verify")
	}

	result, err := NewService(store, time.Hour).Login(context.Background(), "admin", "change-me-before-production")
	if err != nil {
		t.Fatalf("login seeded administrator: %v", err)
	}
	if result.User.Username != "admin" {
		t.Fatalf("unexpected user %q", result.User.Username)
	}
	if len(result.User.Permissions) == 0 {
		t.Fatal("administrator has no permissions")
	}
}
