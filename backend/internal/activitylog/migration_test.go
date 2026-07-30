package activitylog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActivityLogMigrationContract(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "migrations", "016_activity_log.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(content))

	for _, fragment := range []string{
		"create table activity_logs",
		"enable row level security",
		"force row level security",
		"activity_logs_isolation",
		"activity_logs_trigger_insert",
		"reject_activity_log_mutation",
		"before update or delete on activity_logs",
		"sanitize_activity_snapshot",
		"password_hash",
		"smtp_password_encrypted",
		"company_logo",
		"login_background",
		"token_hash",
		"activity_log.view",
		"insert into role_permissions",
		"configuration.manage",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("activity log migration missing %q", fragment)
		}
	}
}

func TestActivityLogMigrationCoversBusinessTablesAndActions(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "migrations", "016_activity_log.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(content))

	for _, table := range []string{
		"tenant_settings", "users", "roles", "user_roles", "role_permissions", "units", "categories", "packings",
		"plants", "suppliers", "raw_materials", "purchase_orders", "delivery_notes",
		"receiving_sessions", "receivings", "outgoing_sessions", "outgoing_documents",
		"inventory_ledger_entries", "kanban_lots",
	} {
		if !strings.Contains(sql, "'"+table+"'") {
			t.Fatalf("activity trigger coverage missing table %q", table)
		}
	}

	for _, action := range []string{
		"created", "updated", "activated", "deactivated", "submitted", "approved",
		"rejected", "cancelled", "issued", "completed", "received", "moved",
		"company_logo_updated", "login_background_updated",
	} {
		if !strings.Contains(sql, "'"+action+"'") {
			t.Fatalf("activity action inference missing %q", action)
		}
	}
}
