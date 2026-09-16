package production

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"order-stock/backend/internal/database"
)

// Requires a migrated disposable database through migration 032; the ledger and
// its audit rows are append-only, so fixtures are retained.
func newWIPSQLFixture(t *testing.T) (context.Context, *pgx.Conn, *Store, Actor, Order, uuid.UUID) {
	t.Helper()
	url := os.Getenv("PLANNING_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PLANNING_TEST_DATABASE_URL must point to a disposable database with migrations through 032")
	}
	ctx := context.Background()
	admin, e := pgx.Connect(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { admin.Close(ctx) })
	a := Actor{uuid.New(), uuid.New()}
	plant, unit, fg, material, supplier, bom := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`INSERT INTO tenants(id,code,name) VALUES($1,$2,'WIP ledger test')`, a.TenantID, a.TenantID.String())
	exec(`INSERT INTO users(id,tenant_id,username,display_name,email,password_hash) VALUES($1,$2,'wip','WIP operator','wip@test.invalid','test')`, a.UserID, a.TenantID)
	exec(`SELECT set_config('app.tenant_id',$1,false)`, a.TenantID.String())
	exec(`SELECT set_config('app.user_id',$1,false)`, a.UserID.String())
	exec(`INSERT INTO units(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES($1,$2,'PCS','Pieces',$3,$3)`, unit, a.TenantID, a.UserID)
	exec(`INSERT INTO plants(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES($1,$2,'MAIN','Main plant',$3,$3)`, plant, a.TenantID, a.UserID)
	exec(`INSERT INTO suppliers(id,tenant_id,code,sage_supplier_code,name,email,currency,created_by_user_id,updated_by_user_id) VALUES($1,$2,'SUP','SUP','Supplier','s@test.invalid','IDR',$3,$3)`, supplier, a.TenantID, a.UserID)
	exec(`INSERT INTO raw_materials(id,tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,currency,standard_unit_price,created_by_user_id,updated_by_user_id) VALUES($1,$2,'RM','RM','Steel',$3,$4,1,'IDR',10,$5,$5)`, material, a.TenantID, supplier, unit, a.UserID)
	exec(`INSERT INTO finished_goods(id,tenant_id,item_code,name,base_unit_id,sales_price,currency,created_by_user_id,updated_by_user_id) VALUES($1,$2,'FG','Panel',$3,100,'IDR',$4,$4)`, fg, a.TenantID, unit, a.UserID)
	exec(`INSERT INTO boms(id,tenant_id,output_kind,finished_good_id,revision,status,created_by_user_id,updated_by_user_id) VALUES($1,$2,'FG',$3,1,'ACTIVE',$4,$4)`, bom, a.TenantID, fg, a.UserID)
	exec(`INSERT INTO bom_components(tenant_id,bom_id,raw_material_id,usage_qty,sort_position) VALUES($1,$2,$3,1,0)`, a.TenantID, bom, material)
	db, e := database.Open(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(db.Close)
	s := NewStore(applicationDB{db})
	if _, e = s.CreateRouting(ctx, a, RoutingInput{PartID: fg, Kind: "FG", Name: "Stamp and weld", Currency: "IDR",
		Steps: []RoutingStep{{Code: "STAMPING", Name: "Stamping", Rate: decimal.NewFromInt(5)}, {Code: "WELDING", Name: "Welding", Rate: decimal.NewFromInt(8)}}}); e != nil {
		t.Fatal(e)
	}
	p := validPlan()
	p.PlantID = plant
	p.Lines = []PlanLine{{FinishedGoodID: fg, PlannedQty: decimal.NewFromInt(100)}}
	if p, e = s.Create(ctx, a, p); e != nil {
		t.Fatal(e)
	}
	if p, e = s.Approve(ctx, a, p.ID); e != nil {
		t.Fatal(e)
	}
	o, e := s.CreateOrder(ctx, a, OrderInput{PlanLineID: p.Lines[0].ID, PlannedQty: decimal.NewFromInt(100), DueDate: "2026-09-30"})
	if e != nil {
		t.Fatal(e)
	}
	return ctx, admin, s, a, o, material
}

func TestWIPLedgerSQLWorkflow(t *testing.T) {
	ctx, admin, s, a, o, material := newWIPSQLFixture(t)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	stamping, welding := o.Operations[0], o.Operations[1]
	post := func(operation uuid.UUID, date string, processed, good int64, materials []MaterialUsage) Entry {
		t.Helper()
		input := EntryInput{OrderID: o.ID, OperationID: operation, OperatorID: a.UserID, ProductionDate: date, Shift: "1",
			Processed: decimal.NewFromInt(processed), Good: decimal.NewFromInt(good), Rejected: decimal.NewFromInt(processed - good), Materials: materials}
		entry, e := s.CreateEntry(ctx, a, input)
		if e != nil {
			t.Fatal("post entry:", e)
		}
		return entry
	}
	usage := func(quantity int64) []MaterialUsage {
		return []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(quantity)}}
	}

	// Two stamping batches with different costs become two WIP lots.
	// 20 pieces, 20 material at 10 plus 20 × 5 process = 300 over 20 good = 15.
	first := post(stamping.ID, "2026-09-10", 20, 20, usage(20))
	// 10 processed, 8 good: 10 material at 10 plus 10 × 5 = 150 over 8 good = 18.75.
	second := post(stamping.ID, "2026-09-12", 10, 8, usage(10))
	balance, e := s.GetWIP(ctx, a, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !balance.Operations[0].OnHand.Equal(decimal.NewFromInt(28)) || !balance.Operations[0].Received.Equal(decimal.NewFromInt(28)) {
		t.Fatalf("stamping output did not reach the ledger: %+v", balance.Operations[0])
	}
	if !balance.Operations[0].Value.Equal(decimal.NewFromInt(450)) {
		t.Fatalf("wrong WIP value: %s", balance.Operations[0].Value)
	}
	if !balance.Reconciled {
		t.Fatalf("fresh ledger does not reconcile: %+v", balance.Operations)
	}

	// A manual transfer still cannot overdraw the balance or predate its stock.
	if _, e = s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: stamping.ID, Quantity: decimal.NewFromInt(29), MovementDate: "2026-09-13"}); e == nil {
		t.Fatal("transferred more than the WIP balance")
	}
	if _, e = s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: stamping.ID, Quantity: decimal.NewFromInt(5), MovementDate: "2026-09-01"}); e == nil {
		t.Fatal("transfer predates the WIP it draws from")
	}
	if _, e = s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: welding.ID, Quantity: decimal.NewFromInt(1), MovementDate: "2026-09-13"}); e == nil {
		t.Fatal("final operation output treated as WIP")
	}
	moved, e := s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: stamping.ID, Quantity: decimal.NewFromInt(25), MovementDate: "2026-09-13", Notes: "To welding line"})
	if e != nil {
		t.Fatal("transfer:", e)
	}
	// FIFO: 20 at 15 plus 5 at 18.75 leaves 3 pieces at 18.75 behind.
	if !moved.Operations[0].OnHand.Equal(decimal.NewFromInt(3)) || !moved.Operations[0].Value.Equal(decimal.RequireFromString("56.25")) {
		t.Fatalf("FIFO transfer left the wrong remainder: %+v", moved.Operations[0])
	}
	if !moved.Operations[1].Staged.Equal(decimal.NewFromInt(25)) || !moved.Operations[1].Value.Equal(decimal.RequireFromString("393.75")) {
		t.Fatalf("wrong staged balance: %+v", moved.Operations[1])
	}
	locked, e := s.GetEntry(ctx, a, first.ID)
	if e != nil || locked.CanCorrect {
		t.Fatal("entry whose output moved on is still correctable", e)
	}
	if _, e = s.UpdateEntry(ctx, a, first.ID, first.EntryInput); e == nil {
		t.Fatal("locked entry edited")
	}

	// Welding consumes staged WIP; the rest stays as a balance.
	post(welding.ID, "2026-09-14", 20, 19, nil)
	after, e := s.GetWIP(ctx, a, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !after.Operations[1].Consumed.Equal(decimal.NewFromInt(20)) || !after.Operations[1].Staged.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("consumption did not follow the ledger: %+v", after.Operations[1])
	}
	if !after.TotalQty.Equal(decimal.NewFromInt(8)) || !after.Reconciled {
		t.Fatalf("order WIP does not reconcile: %+v", after)
	}
	consumed := uuid.Nil
	for _, m := range after.Movements {
		if m.Type == MovementTransfer && !m.CanReverse && m.ReversesID == uuid.Nil {
			consumed = m.ID
		}
	}
	if consumed == uuid.Nil {
		t.Fatal("consumed transfer lot not found")
	}
	if _, e = s.ReverseMovement(ctx, a, consumed, "mistake"); e == nil {
		t.Fatal("consumed transfer reversed")
	}

	// An untouched transfer can be taken back and restores the source balance.
	open, e := s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: stamping.ID, Quantity: decimal.NewFromInt(3), MovementDate: "2026-09-15"})
	if e != nil {
		t.Fatal("second transfer:", e)
	}
	if !open.Operations[0].OnHand.IsZero() {
		t.Fatalf("stamping balance after full transfer: %+v", open.Operations[0])
	}
	latest := uuid.Nil
	for _, m := range open.Movements {
		if m.CanReverse && m.MovementDate == "2026-09-15" {
			latest = m.ID
		}
	}
	back, e := s.ReverseMovement(ctx, a, latest, "Returned to stamping")
	if e != nil {
		t.Fatal("reverse:", e)
	}
	if !back.Operations[0].OnHand.Equal(decimal.NewFromInt(3)) || !back.Operations[1].Staged.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("reversal did not restore balances: %+v", back.Operations)
	}
	if _, e = s.ReverseMovement(ctx, a, latest, "again"); e == nil {
		t.Fatal("movement reversed twice")
	}
	released, e := s.GetEntry(ctx, a, second.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !released.EffectsLocked {
		t.Fatal("entry unlocked while five pieces remain transferred")
	}
	for _, m := range back.Movements {
		if m.CanReverse && m.ReversesID == uuid.Nil {
			if _, e = s.ReverseMovement(ctx, a, m.ID, "Return remaining five pieces"); e != nil {
				t.Fatal(e)
			}
		}
	}
	released, e = s.GetEntry(ctx, a, second.ID)
	if e != nil || released.EffectsLocked || !released.CanCorrect {
		t.Fatalf("entry not correctable after all its WIP returned: %+v %v", released, e)
	}

	// The ledger is history: rows may never be edited or deleted.
	if _, e = admin.Exec(ctx, `DELETE FROM production_wip_movements WHERE tenant_id=$1`, a.TenantID); e == nil {
		t.Fatal("ledger rows can be deleted")
	}
	if _, e = admin.Exec(ctx, `UPDATE production_wip_movements SET quantity=1 WHERE tenant_id=$1`, a.TenantID); e == nil {
		t.Fatal("ledger rows can be updated")
	}
	other := Actor{uuid.New(), uuid.New()}
	items, e := s.ListWIP(ctx, other)
	if e != nil || len(items) != 0 {
		t.Fatal("cross-tenant WIP visible", e)
	}
	if e = s.CloseProductionPeriod(ctx, a, "2026-09", "Month reviewed"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: stamping.ID, Quantity: decimal.NewFromInt(1), MovementDate: "2026-09-16"}); e == nil {
		t.Fatal("closed period accepted a WIP movement")
	}

	// Actual cost: 300 material, 150 stamping and 160 welding were booked, and
	// 150 of that is still held as WIP, so 460 left the order as finished output.
	cost, e := s.GetOrderCost(ctx, a, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !cost.MaterialActual.Equal(decimal.NewFromInt(300)) || !cost.ProcessActual.Equal(decimal.NewFromInt(310)) || !cost.TotalActual.Equal(decimal.NewFromInt(610)) {
		t.Fatalf("wrong actual cost: %+v", cost)
	}
	if !cost.WIPValue.Equal(decimal.NewFromInt(150)) || !cost.FinishedCost.Equal(decimal.NewFromInt(460)) {
		t.Fatalf("WIP cost not separated from finished output: %+v", cost)
	}
	if !cost.GoodQty.Equal(decimal.NewFromInt(19)) || !cost.ActualUnitCost.Equal(decimal.RequireFromString("24.210526")) {
		t.Fatalf("wrong cost per finished piece: %s over %s", cost.ActualUnitCost, cost.GoodQty)
	}
	if len(cost.Operations) != 2 || !cost.Operations[0].ActualCost.Equal(decimal.NewFromInt(150)) || !cost.Operations[1].ActualCost.Equal(decimal.NewFromInt(160)) {
		t.Fatalf("wrong process cost per operation: %+v", cost.Operations)
	}
	if len(cost.Materials) != 1 || !cost.Materials[0].UsedQty.Equal(decimal.NewFromInt(30)) || !cost.Materials[0].StandardQty.Equal(decimal.NewFromInt(30)) {
		t.Fatalf("wrong material usage: %+v", cost.Materials)
	}
	if !cost.Materials[0].AveragePrice.Equal(decimal.NewFromInt(10)) || !cost.Materials[0].QtyVariance.IsZero() {
		t.Fatalf("material snapshot price drifted: %+v", cost.Materials[0])
	}
	// Master prices may change afterwards without touching recorded history.
	exec(`UPDATE raw_materials SET standard_unit_price=99 WHERE tenant_id=$1 AND id=$2`, a.TenantID, material)
	repeat, e := s.GetOrderCost(ctx, a, o.ID)
	if e != nil || !repeat.MaterialActual.Equal(decimal.NewFromInt(300)) {
		t.Fatalf("master price change rewrote history: %+v %v", repeat, e)
	}
	periods, e := s.ListPeriodCosts(ctx, a)
	if e != nil || len(periods) != 1 {
		t.Fatalf("period costs: %+v %v", periods, e)
	}
	if periods[0].Period != "2026-09" || !periods[0].Closed || periods[0].Entries != 3 || !periods[0].TotalActual.Equal(decimal.NewFromInt(610)) {
		t.Fatalf("wrong period summary: %+v", periods[0])
	}
	if _, e = s.ListOrderCosts(ctx, Actor{uuid.New(), uuid.New()}); e != nil {
		t.Fatal(e)
	}
}
