package purchaseorder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readMigration(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "..", "database", "migrations", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}

	return string(content)
}

func TestInboundMigrationContainsGenerationAndIsolationContracts(t *testing.T) {
	sql := readMigration(t, "006_inbound_documents.sql")
	required := []string{
		"create table if not exists delivery_notes",
		"unique (tenant_id, purchase_order_id)",
		"create table if not exists delivery_note_lines",
		"unique (tenant_id, delivery_note_id, purchase_order_line_id)",
		"create table if not exists kanban_lots",
		"unique (tenant_id, kanban_id)",
		"force row level security",
		"where p.status = 'approved'",
		"generate_series",
		"new.lot_number > expected_total_kanban",
		"before update of qty_per_kanban_snapshot, total_kanban",
		"inbound document capacity preflight failed",
	}
	for _, fragment := range required {
		if !strings.Contains(strings.ToLower(sql), fragment) {
			t.Errorf("missing %q", fragment)
		}
	}
}
