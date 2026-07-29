package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationAddsIdentityAndApproverContracts(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "migrations", "004_settings_identity.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{"add column if not exists email", "users_tenant_email_ci_key", "add column if not exists active", "default_approver_user_id", "foreign key (tenant_id, default_approver_user_id)", "references users(tenant_id, id)"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}

func TestDashboardPermissionMigrationRestoresExistingDashboardAccess(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "migrations", "011_dashboard_permission.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{"dashboard.view", "insert into permissions", "insert into role_permissions", "inventory.view", "on conflict"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("dashboard migration missing %q", fragment)
		}
	}
}
