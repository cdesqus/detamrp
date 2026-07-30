package purchaseorder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationContainsTenantScopedPurchaseOrderContracts(t *testing.T) {
	path := filepath.Join("..", "..", "..", "database", "migrations", "005_purchase_orders.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read purchase order migration: %v", err)
	}

	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"create table if not exists purchase_order_number_sequences",
		"create table if not exists purchase_orders",
		"create table if not exists purchase_order_lines",
		"create table if not exists purchase_order_approvals",
		"unique (tenant_id, id)",
		"unique (tenant_id, po_number)",
		"unique (tenant_id, purchase_order_id, raw_material_id)",
		"unique (tenant_id, purchase_order_id, version)",
		"qty_per_kanban_snapshot numeric(20,6)",
		"total_kanban numeric(20,6)",
		"ordered_base_qty numeric(20,6)",
		"unit_price_snapshot numeric(20,6)",
		"line_total numeric(20,6)",
		"total_amount numeric(20,6)",
		"expected_delivery_date >= order_date",
		"foreign key (tenant_id, supplier_id)",
		"foreign key (tenant_id, raw_material_id)",
		"foreign key (tenant_id, base_unit_id)",
		"force row level security",
		"purchase_order_supplier_matches_material",
		"grant select, insert, update, delete on purchase_order_lines to nextgen_app",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}

func TestMigrationCanReconcileAnUntrackedExistingPurchaseOrderSchema(t *testing.T) {
	path := filepath.Join("..", "..", "..", "database", "migrations", "005_purchase_orders.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read purchase order migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"create table if not exists purchase_order_number_sequences",
		"create table if not exists purchase_orders",
		"create table if not exists purchase_order_lines",
		"create table if not exists purchase_order_approvals",
		"create index if not exists purchase_orders_tenant_status_order_date_idx",
		"if not exists (select 1 from pg_trigger",
		"if not exists (select 1 from pg_policies",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("repeat-safe migration contract missing %q", fragment)
		}
	}
}

func TestReferenceSnapshotMigrationAddsPlantCategoryAndPackingHistory(t *testing.T) {
	path := filepath.Join("..", "..", "..", "database", "migrations", "013_material_po_reference_snapshots.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read reference snapshot migration: %v", err)
	}
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{
		"add column plant_id uuid",
		"plant_code_snapshot text not null default ''",
		"plant_name_snapshot text not null default ''",
		"plant_address_snapshot text not null default ''",
		"category_code_snapshot text not null default ''",
		"category_name_snapshot text not null default ''",
		"packing_code_snapshot text not null default ''",
		"packing_name_snapshot text not null default ''",
		"foreign key (tenant_id, plant_id) references plants(tenant_id, id)",
		"purchase_orders_tenant_plant_idx",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}
