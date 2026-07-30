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

func TestRawMaterialPriceMigration(t *testing.T) {
	path := filepath.Join("..", "..", "..", "database", "migrations", "003_raw_material_price.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read price migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{"standard_unit_price numeric(20,6)", "standard_unit_price >= 0", "currency char(3)", "from suppliers", "alter column currency set not null"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}

func TestMasterReferenceMigrationDefinesUnitsCategoriesPackingsAndPlants(t *testing.T) {
	path := filepath.Join("..", "..", "..", "database", "migrations", "012_master_reference_data.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read master reference migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	required := []string{
		"alter table measurements rename to units",
		"create table categories",
		"create table packings",
		"create table plants",
		"unique (tenant_id, code)",
		"alter table categories enable row level security",
		"alter table packings enable row level security",
		"alter table plants enable row level security",
		"alter table categories force row level security",
		"alter table packings force row level security",
		"alter table plants force row level security",
		"create policy categories_isolation",
		"create policy packings_isolation",
		"create policy plants_isolation",
		"grant select, insert, update on categories, packings, plants to nextgen_app",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}

func TestMaterialReferenceSnapshotMigrationLinksCategoryAndPacking(t *testing.T) {
	path := filepath.Join("..", "..", "..", "database", "migrations", "013_material_po_reference_snapshots.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read material reference snapshot migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"add column category_id uuid",
		"add column packing_id uuid",
		"foreign key (tenant_id, category_id) references categories(tenant_id, id)",
		"foreign key (tenant_id, packing_id) references packings(tenant_id, id)",
		"raw_materials_tenant_category_idx",
		"raw_materials_tenant_packing_idx",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}
