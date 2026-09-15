package production

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Production must move raw material out of stock, and put it back when the
// booking is corrected or voided.
func TestProductionMovesMaterialThroughInventorySQL(t *testing.T) {
	ctx, admin, s, a, o, material := newWIPSQLFixture(t)
	issued := func() decimal.Decimal {
		t.Helper()
		var net decimal.Decimal
		if e := admin.QueryRow(ctx, `SELECT COALESCE(sum(quantity_delta),0) FROM inventory_ledger_entries
 WHERE tenant_id=$1 AND raw_material_id=$2 AND event_type IN ('PRODUCTION_ISSUE','PRODUCTION_RETURN')`, a.TenantID, material).Scan(&net); e != nil {
			t.Fatal(e)
		}
		return net
	}
	rows := func() int {
		t.Helper()
		var count int
		if e := admin.QueryRow(ctx, `SELECT count(*) FROM inventory_ledger_entries WHERE tenant_id=$1 AND reference_type='PRODUCTION_ENTRY'`, a.TenantID).Scan(&count); e != nil {
			t.Fatal(e)
		}
		return count
	}
	input := EntryInput{OrderID: o.ID, OperationID: o.Operations[0].ID, OperatorID: a.UserID, ProductionDate: "2026-09-10", Shift: "1",
		Processed: decimal.NewFromInt(20), Good: decimal.NewFromInt(20),
		Materials: []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(20)}}}
	entry, e := s.CreateEntry(ctx, a, input)
	if e != nil {
		t.Fatal("create:", e)
	}
	if !issued().Equal(decimal.NewFromInt(-20)) {
		t.Fatalf("material did not leave stock: %s", issued())
	}

	// A correction returns the old usage and issues the new one.
	corrected := entry.EntryInput
	corrected.Materials = []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(12)}}
	corrected.Version = entry.Version
	entry, e = s.UpdateEntry(ctx, a, entry.ID, corrected)
	if e != nil {
		t.Fatal("edit:", e)
	}
	if !issued().Equal(decimal.NewFromInt(-12)) {
		t.Fatalf("correction did not reach stock: %s", issued())
	}

	// Voiding puts everything back and leaves the ledger append-only.
	before := rows()
	if _, e = s.VoidEntry(ctx, a, entry.ID, entry.Version, "Wrong material"); e != nil {
		t.Fatal("void:", e)
	}
	if !issued().IsZero() {
		t.Fatalf("void did not return material: %s", issued())
	}
	if rows() <= before {
		t.Fatal("void rewrote history instead of appending a return")
	}
	if _, e = admin.Exec(ctx, `DELETE FROM inventory_ledger_entries WHERE tenant_id=$1 AND reference_type='PRODUCTION_ENTRY'`, a.TenantID); e == nil {
		t.Fatal("production ledger rows can be deleted")
	}

	// Later operations consume WIP, not warehouse stock.
	second := EntryInput{OrderID: o.ID, OperationID: o.Operations[0].ID, OperatorID: a.UserID, ProductionDate: "2026-09-11", Shift: "1",
		Processed: decimal.NewFromInt(10), Good: decimal.NewFromInt(10),
		Materials: []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(10)}}}
	if _, e = s.CreateEntry(ctx, a, second); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: o.Operations[0].ID, Quantity: decimal.NewFromInt(10), MovementDate: "2026-09-11"}); e != nil {
		t.Fatal(e)
	}
	beforeWelding := issued()
	if _, e = s.CreateEntry(ctx, a, EntryInput{OrderID: o.ID, OperationID: o.Operations[1].ID, OperatorID: a.UserID,
		ProductionDate: "2026-09-12", Shift: "1", Processed: decimal.NewFromInt(10), Good: decimal.NewFromInt(9), Rejected: decimal.NewFromInt(1)}); e != nil {
		t.Fatal(e)
	}
	if !issued().Equal(beforeWelding) {
		t.Fatalf("a later operation touched warehouse stock: %s", issued())
	}
	if material == uuid.Nil {
		t.Fatal("fixture lost its material")
	}
}
