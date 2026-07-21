package auth

import (
	"os"
	"strings"
	"testing"
)

func TestInitialAdminInsertSuppliesRequiredEmail(t *testing.T) {
	content, err := os.ReadFile("sql_store.go")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ToLower(string(content))
	if !strings.Contains(source, "insert into users (tenant_id, username, display_name, email, password_hash)") {
		t.Fatal("initial admin insert must supply required users.email")
	}
}
