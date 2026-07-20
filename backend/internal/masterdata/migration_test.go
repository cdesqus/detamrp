package masterdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMasterDataMigrationContainsTenantAndKanbanConstraints(t *testing.T) {
	path := filepath.Join("..", "..", "..", "database", "migrations", "002_master_data.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read master data migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	required := []string{
		"create table measurements", "create table suppliers", "create table raw_materials",
		"create table warehouses", "create table warehouse_locations", "qty_per_kanban > 0",
		"foreign key (tenant_id, supplier_id)", "force row level security",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
