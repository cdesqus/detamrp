package production

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
	"os"
	"testing"
)

// Requires a migrated disposable database; immutable audit fixtures are retained.
func TestDailyProductionSQLWorkflow(t *testing.T) {
	url := os.Getenv("PLANNING_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PLANNING_TEST_DATABASE_URL must point to a disposable database with migrations through 031")
	}
	ctx := context.Background()
	admin, e := pgx.Connect(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(ctx)
	a := Actor{uuid.New(), uuid.New()}
	plant, unit, fg, material, supplier, bom := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`INSERT INTO tenants(id,code,name) VALUES($1,$2,'Daily production test')`, a.TenantID, a.TenantID.String())
	exec(`INSERT INTO users(id,tenant_id,username,display_name,email,password_hash) VALUES($1,$2,'operator','Operator','operator@test.invalid','test')`, a.UserID, a.TenantID)
	exec(`SELECT set_config('app.tenant_id',$1,false)`, a.TenantID.String())
	exec(`SELECT set_config('app.user_id',$1,false)`, a.UserID.String())
	exec(`INSERT INTO units(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES($1,$2,'PCS','Pieces',$3,$3)`, unit, a.TenantID, a.UserID)
	exec(`INSERT INTO plants(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES($1,$2,'MAIN','Main plant',$3,$3)`, plant, a.TenantID, a.UserID)
	exec(`INSERT INTO suppliers(id,tenant_id,code,sage_supplier_code,name,email,currency,created_by_user_id,updated_by_user_id) VALUES($1,$2,'SUP','SUP','Supplier','s@test.invalid','IDR',$3,$3)`, supplier, a.TenantID, a.UserID)
	exec(`INSERT INTO raw_materials(id,tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,currency,standard_unit_price,created_by_user_id,updated_by_user_id) VALUES($1,$2,'RM','RM','Steel',$3,$4,1,'IDR',5,$5,$5)`, material, a.TenantID, supplier, unit, a.UserID)
	exec(`INSERT INTO finished_goods(id,tenant_id,item_code,name,base_unit_id,sales_price,currency,created_by_user_id,updated_by_user_id) VALUES($1,$2,'FG','Panel',$3,100,'IDR',$4,$4)`, fg, a.TenantID, unit, a.UserID)
	exec(`INSERT INTO boms(id,tenant_id,output_kind,finished_good_id,revision,status,created_by_user_id,updated_by_user_id) VALUES($1,$2,'FG',$3,1,'ACTIVE',$4,$4)`, bom, a.TenantID, fg, a.UserID)
	exec(`INSERT INTO bom_components(tenant_id,bom_id,raw_material_id,usage_qty,sort_position) VALUES($1,$2,$3,2,0)`, a.TenantID, bom, material)
	db, e := database.Open(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := NewStore(applicationDB{db})
	if _, e = s.CreateRouting(ctx, a, RoutingInput{PartID: fg, Kind: "FG", Name: "Stamp and weld", Currency: "IDR", Steps: []RoutingStep{{Code: "STAMPING", Name: "Stamping", Rate: decimal.NewFromInt(10)}, {Code: "WELDING", Name: "Welding", Rate: decimal.NewFromInt(20)}}}); e != nil {
		t.Fatal(e)
	}
	p := validPlan()
	p.PlantID = plant
	p.Lines = []PlanLine{{FinishedGoodID: fg, PlannedQty: decimal.NewFromInt(100)}}
	p, e = s.Create(ctx, a, p)
	if e != nil {
		t.Fatal(e)
	}
	p, e = s.Approve(ctx, a, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	o, e := s.CreateOrder(ctx, a, OrderInput{PlanLineID: p.Lines[0].ID, PlannedQty: decimal.NewFromInt(100), DueDate: "2026-09-30"})
	if e != nil {
		t.Fatal(e)
	}
	i := validEntryInput()
	i.OrderID = o.ID
	i.OperationID = o.Operations[1].ID
	i.OperatorID = a.UserID
	if _, e = s.CreateEntry(ctx, a, i); e == nil {
		t.Fatal("Welding started without upstream good output")
	}
	i.OperationID = o.Operations[0].ID
	i.Materials = []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(120)}}
	i.Processed = decimal.NewFromInt(60)
	i.Good = decimal.NewFromInt(55)
	i.Rejected = decimal.NewFromInt(5)
	first, e := s.CreateEntry(ctx, a, i)
	if e != nil {
		t.Fatal("create:", e)
	}
	if first.EntryNumber != "DP-0000001" || !first.MaterialCost.Equal(decimal.NewFromInt(600)) || !first.ProcessCost.Equal(decimal.NewFromInt(600)) {
		t.Fatalf("wrong snapshots: %+v", first)
	}
	// Edits retain the original transaction price, even when master prices change.
	exec(`UPDATE raw_materials SET standard_unit_price=9 WHERE tenant_id=$1 AND id=$2`, a.TenantID, material)
	edit := first.EntryInput
	edit.Processed = decimal.NewFromInt(40)
	edit.Good = decimal.NewFromInt(36)
	edit.Rejected = decimal.NewFromInt(4)
	edit.Materials = []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(80)}}
	first, e = s.UpdateEntry(ctx, a, first.ID, edit)
	if e != nil {
		t.Fatal("edit:", e)
	}
	if !first.MaterialCost.Equal(decimal.NewFromInt(400)) || first.Version != 2 {
		t.Fatal("edited snapshot changed price or version", first)
	}
	if _, e = s.UpdateEntry(ctx, a, first.ID, edit); e == nil {
		t.Fatal("stale edit accepted")
	}
	other := Actor{uuid.New(), uuid.New()}
	if _, e = s.GetEntry(ctx, other, first.ID); !errors.Is(e, ErrNotFound) {
		t.Fatal("cross tenant read", e)
	}
	secondInput := validEntryInput()
	secondInput.OrderID = o.ID
	secondInput.OperationID = o.Operations[1].ID
	secondInput.OperatorID = a.UserID
	secondInput.Processed = decimal.NewFromInt(20)
	secondInput.Good = decimal.NewFromInt(18)
	secondInput.Rejected = decimal.NewFromInt(2)
	if _, e = s.CreateEntry(ctx, a, secondInput); e == nil {
		t.Fatal("welding started without a WIP transfer")
	}
	transfer, e := s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: o.Operations[0].ID, Quantity: decimal.NewFromInt(20), MovementDate: "2026-09-14"})
	if e != nil {
		t.Fatal("transfer:", e)
	}
	if !transfer.Operations[0].OnHand.Equal(decimal.NewFromInt(16)) || !transfer.Operations[1].Staged.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("wrong balances after transfer: %+v", transfer.Operations)
	}
	second, e := s.CreateEntry(ctx, a, secondInput)
	if e != nil {
		t.Fatal("welding:", e)
	}
	if _, e = s.VoidEntry(ctx, a, first.ID, first.Version, "Wrong source"); e == nil {
		t.Fatal("consumed source voided")
	}
	edit.Version = first.Version
	if _, e = s.UpdateEntry(ctx, a, first.ID, edit); e == nil {
		t.Fatal("consumed source edited")
	}
	current, e := s.GetOrder(ctx, a, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if current.Status != OrderPartial || !current.WIPQty.Equal(decimal.NewFromInt(16)) {
		t.Fatalf("wrong progress: %+v", current)
	}
	if _, e = s.CreateEntry(ctx, a, secondInput); e == nil {
		t.Fatal("PARTIAL order accepted without resume")
	}
	if _, e = s.OrderAction(ctx, a, o.ID, "cancel", "stop"); e == nil {
		t.Fatal("order cancelled with remaining WIP")
	}
	second, e = s.VoidEntry(ctx, a, second.ID, second.Version, "Duplicate entry")
	if e != nil {
		t.Fatal("void:", e)
	}
	if second.Status != "VOIDED" || second.CanCorrect {
		t.Fatal("voided entry still editable")
	}
	if _, e = s.VoidEntry(ctx, a, second.ID, second.Version, "Again"); e == nil {
		t.Fatal("double void accepted")
	}
	current, e = s.GetOrder(ctx, a, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !current.ActualGood.IsZero() || !current.ProcessCost.Equal(decimal.NewFromInt(400)) || !current.WIPQty.Equal(decimal.NewFromInt(36)) {
		t.Fatalf("void failed to reverse aggregates: %+v", current)
	}
	plan, e := s.Get(ctx, a, p.ID)
	if e != nil || !plan.Orders[0].GoodQty.IsZero() {
		t.Fatal("voided final output remains in planning progress", e)
	}
	first, e = s.GetEntry(ctx, a, first.ID)
	if e != nil || first.CanCorrect {
		t.Fatal("stamping output stays locked while its WIP sits at welding", e)
	}
	movements, e := s.GetWIP(ctx, a, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	open := uuid.Nil
	for _, m := range movements.Movements {
		if m.CanReverse {
			open = m.ID
		}
	}
	if open == uuid.Nil {
		t.Fatal("transfer could not be taken back after the welding entry was voided")
	}
	if _, e = s.ReverseMovement(ctx, a, open, "Returned to stamping"); e != nil {
		t.Fatal("reverse transfer:", e)
	}
	first, e = s.GetEntry(ctx, a, first.ID)
	if e != nil || !first.CanCorrect {
		t.Fatal("source still locked after its WIP came back", e)
	}
	if e = s.CloseProductionPeriod(ctx, a, "2026-09", "Month reviewed"); e != nil {
		t.Fatal("close period:", e)
	}
	if _, e = s.VoidEntry(ctx, a, first.ID, first.Version, "correct"); e == nil {
		t.Fatal("closed-period void accepted")
	}
	if _, e = s.UpdateEntry(ctx, a, first.ID, first.EntryInput); e == nil {
		t.Fatal("closed-period edit accepted")
	}
	if _, e = s.CreateEntry(ctx, a, i); e == nil {
		t.Fatal("closed-period create accepted")
	}
	locked, e := s.GetEntry(ctx, a, first.ID)
	if e != nil || !locked.PeriodClosed || locked.CanCorrect {
		t.Fatal("closed-period UI flags incorrect", e)
	}
	list, e := s.ListEntries(ctx, a)
	if e != nil || len(list) != 2 {
		t.Fatal("list lost history", e)
	}
	options, e := s.EntryOptions(ctx, a)
	if e != nil || len(options.ClosedPeriods) != 1 || len(options.Operators) != 1 {
		t.Fatal("entry options", e)
	}
	historyFound := false
	for _, h := range second.History {
		if h.Action == "VOIDED" {
			historyFound = true
		}
	}
	if !historyFound {
		t.Fatal("void missing from audit")
	}
	if _, e = admin.Exec(ctx, `DELETE FROM production_entries WHERE tenant_id=$1 AND id=$2`, a.TenantID, second.ID); e == nil {
		t.Fatal("hard delete not guarded")
	}
}
