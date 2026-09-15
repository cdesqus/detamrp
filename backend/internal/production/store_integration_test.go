package production

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
	"os"
	"strings"
	"testing"
)

// Run only against a disposable database with migrations 001–030 applied.
// Audit history is append-only, so test fixtures are retained in that database.
func TestPlanningSQLWorkflow(t *testing.T) {
	url := os.Getenv("PLANNING_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("PLANNING_TEST_DATABASE_URL must point to a migrated disposable database")
	}
	ctx := context.Background()
	admin, e := pgx.Connect(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(ctx)
	a := Actor{uuid.New(), uuid.New()}
	plant, unit, fg := uuid.New(), uuid.New(), uuid.New()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := admin.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`INSERT INTO tenants(id,code,name) VALUES($1,$2,'Planning test')`, a.TenantID, a.TenantID.String())
	exec(`INSERT INTO users(id,tenant_id,username,display_name,email,password_hash) VALUES($1,$2,'planner','Planner','planner@test.invalid','test')`, a.UserID, a.TenantID)
	exec(`SELECT set_config('app.tenant_id',$1,false)`, a.TenantID.String())
	exec(`SELECT set_config('app.user_id',$1,false)`, a.UserID.String())
	exec(`INSERT INTO units(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES($1,$2,'PCS','Pieces',$3,$3)`, unit, a.TenantID, a.UserID)
	exec(`INSERT INTO plants(id,tenant_id,code,name,created_by_user_id,updated_by_user_id) VALUES($1,$2,'MAIN','Main plant',$3,$3)`, plant, a.TenantID, a.UserID)
	exec(`INSERT INTO finished_goods(id,tenant_id,item_code,name,base_unit_id,sales_price,currency,created_by_user_id,updated_by_user_id) VALUES($1,$2,'FG-TEST','Test part',$3,10,'IDR',$4,$4)`, fg, a.TenantID, unit, a.UserID)
	// Orders snapshot an active BOM and routing, so both must exist before a plan
	// can be released into production.
	supplier, component, bom := uuid.New(), uuid.New(), uuid.New()
	exec(`INSERT INTO suppliers(id,tenant_id,code,sage_supplier_code,name,email,currency,created_by_user_id,updated_by_user_id) VALUES($1,$2,'SUP','SUP','Supplier','s@test.invalid','IDR',$3,$3)`, supplier, a.TenantID, a.UserID)
	exec(`INSERT INTO raw_materials(id,tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,currency,standard_unit_price,created_by_user_id,updated_by_user_id) VALUES($1,$2,'BLANK','BLANK','Blank',$3,$4,1,'IDR',5,$5,$5)`, component, a.TenantID, supplier, unit, a.UserID)
	exec(`INSERT INTO boms(id,tenant_id,output_kind,finished_good_id,revision,status,created_by_user_id,updated_by_user_id) VALUES($1,$2,'FG',$3,1,'ACTIVE',$4,$4)`, bom, a.TenantID, fg, a.UserID)
	exec(`INSERT INTO bom_components(tenant_id,bom_id,raw_material_id,usage_qty,sort_position) VALUES($1,$2,$3,2,0)`, a.TenantID, bom, component)
	db, e := database.Open(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s := NewStore(applicationDB{db})
	routing := routingWorkflow(t, ctx, s, a, "FG", fg)
	input := validPlan()
	input.PlantID = plant
	input.Lines[0].FinishedGoodID = fg
	input.Lines[0].UnitCode = "FORGED"
	options, e := s.Options(ctx, a)
	if e != nil || len(options.Parts) != 2 || len(options.Plants) != 1 {
		t.Fatalf("options: %+v %v", options, e)
	}
	p, e := s.Create(ctx, a, input)
	if e != nil {
		t.Fatal("create:", e)
	}
	if p.Status != PlanDraft || p.PlanNumber != "PP-202609-000001" || p.Lines[0].UnitCode != "PCS" || p.Lines[0].PartNumber != "FG-TEST" {
		t.Fatalf("unexpected draft: %+v", p)
	}
	input.Lines[0].PlannedQty = decimal.NewFromInt(25)
	p, e = s.Update(ctx, a, p.ID, input)
	if e != nil || !p.TotalPlannedQty.Equal(decimal.NewFromInt(25)) {
		t.Fatalf("update: %+v %v", p, e)
	}
	wrong := input
	wrong.Lines = []PlanLine{{FinishedGoodID: uuid.New(), PlannedQty: decimal.NewFromInt(99)}}
	if _, e = s.Update(ctx, a, p.ID, wrong); !errors.Is(e, ErrInvalidReference) {
		t.Fatalf("invalid reference: %v", e)
	}
	current, e := s.Get(ctx, a, p.ID)
	if e != nil || !current.TotalPlannedQty.Equal(decimal.NewFromInt(25)) {
		t.Fatalf("failed update was not rolled back: %+v %v", current, e)
	}
	other := Actor{uuid.New(), uuid.New()}
	if _, e = s.Get(ctx, other, p.ID); !errors.Is(e, ErrNotFound) {
		t.Fatalf("cross-tenant read: %v", e)
	}
	if _, e = s.CreateOrders(ctx, a, p.ID); !errors.Is(e, ErrConflict) {
		t.Fatalf("draft orders: %v", e)
	}
	exec(`UPDATE finished_goods SET active=false WHERE id=$1`, fg)
	if _, e = s.Approve(ctx, a, p.ID); !errors.Is(e, ErrInvalidReference) {
		t.Fatalf("inactive approval: %v", e)
	}
	exec(`UPDATE finished_goods SET active=true WHERE id=$1`, fg)
	p, e = s.Approve(ctx, a, p.ID)
	if e != nil || p.Status != PlanApproved {
		t.Fatalf("approve: %+v %v", p, e)
	}
	if _, e = s.Update(ctx, a, p.ID, input); !errors.Is(e, ErrConflict) {
		t.Fatalf("approved edit: %v", e)
	}
	p, e = s.CreateOrders(ctx, a, p.ID)
	if e != nil || len(p.Orders) != 1 {
		t.Fatalf("create orders: %+v %v", p, e)
	}
	order, e := s.GetOrder(ctx, a, p.Orders[0].ID)
	if e != nil || len(order.Operations) != 2 || len(order.Materials) != 1 || !order.ProcessEstimate.Equal(decimal.NewFromInt(650*25)) {
		t.Fatalf("orders from planning must carry routing and BOM snapshots: %+v %v", order, e)
	}
	if order.RoutingID != routing.ID || !order.MaterialEstimate.Equal(decimal.NewFromInt(2*5*25)) {
		t.Fatalf("unexpected order snapshot: %+v", order)
	}
	if _, e = s.UpdateRouting(ctx, a, routing.ID, RoutingInput{Name: "Changed", Currency: "IDR", Steps: routing.Steps}); !errors.Is(e, ErrConflict) {
		t.Fatalf("routing used by an order was edited: %v", e)
	}
	if _, e = s.RoutingAction(ctx, a, routing.ID, "delete"); !errors.Is(e, ErrConflict) {
		t.Fatalf("routing used by an order was deleted: %v", e)
	}
	if _, e = s.CreateOrders(ctx, a, p.ID); !errors.Is(e, ErrConflict) {
		t.Fatalf("duplicate orders: %v", e)
	}
	if e = s.Delete(ctx, a, p.ID); !errors.Is(e, ErrConflict) {
		t.Fatalf("linked delete: %v", e)
	}
	// Plan progress counts the good output of the final routing operation only.
	final := order.Operations[len(order.Operations)-1].ID
	exec(`INSERT INTO production_entries(tenant_id,operation_id,production_date,qty_processed,qty_good,qty_rejected,created_by_user_id,entry_number,operator_user_id,updated_by_user_id) VALUES($1,$2,'2026-09-14',10,9,1,$3,'DP-TEST-FIXTURE',$3,$3)`, a.TenantID, final, a.UserID)
	p, e = s.Close(ctx, a, p.ID)
	if e != nil || p.Status != PlanClosed || !p.Orders[0].GoodQty.Equal(decimal.NewFromInt(9)) {
		t.Fatalf("close/progress: %+v %v", p, e)
	}
	found := false
	for _, h := range p.History {
		if h.Action == "CLOSED" {
			found = true
		}
	}
	if !found {
		t.Fatal("close missing from audit history")
	}
	draft, e := s.Create(ctx, a, input)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasSuffix(draft.PlanNumber, "000002") {
		t.Fatal("number not sequential", draft.PlanNumber)
	}
	if e = s.Delete(ctx, a, draft.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Get(ctx, a, draft.ID); !errors.Is(e, ErrNotFound) {
		t.Fatal("deleted draft still found", e)
	}
	items, e := s.List(ctx, a)
	if e != nil || len(items) != 1 {
		t.Fatalf("list: %+v %v", items, e)
	}
	// Process material targets use the same workflow and keep FG references empty.
	material, processBOM := uuid.New(), uuid.New()
	exec(`INSERT INTO raw_materials(id,tenant_id,code,sage_item_code,name,supplier_id,base_unit_id,qty_per_kanban,currency,standard_unit_price,created_by_user_id,updated_by_user_id) VALUES($1,$2,'PROCESS','PROCESS','Process material',$3,$4,1,'IDR',10,$5,$5)`, material, a.TenantID, supplier, unit, a.UserID)
	exec(`INSERT INTO boms(id,tenant_id,output_kind,raw_material_id,revision,status,created_by_user_id,updated_by_user_id) VALUES($1,$2,'RAW_MATERIAL',$3,1,'ACTIVE',$4,$4)`, processBOM, a.TenantID, material, a.UserID)
	exec(`INSERT INTO bom_components(tenant_id,bom_id,raw_material_id,usage_qty,sort_position) VALUES($1,$2,$3,1,0)`, a.TenantID, processBOM, component)
	routingWorkflow(t, ctx, s, a, "RAW_MATERIAL", material)
	input.Lines = []PlanLine{{RawMaterialID: material, PlannedQty: decimal.NewFromInt(30)}}
	process, e := s.Create(ctx, a, input)
	if e != nil {
		t.Fatal(e)
	}
	if process.Lines[0].FinishedGoodID != uuid.Nil || process.Lines[0].PartNumber != "PROCESS" {
		t.Fatalf("wrong process output: %+v", process.Lines)
	}
	if _, e = s.Approve(ctx, a, process.ID); e != nil {
		t.Fatal(e)
	}
	process, e = s.CreateOrders(ctx, a, process.ID)
	if e != nil || len(process.Orders) != 1 {
		t.Fatalf("process order: %+v %v", process, e)
	}
}

type applicationDB struct{ database.TenantBeginner }

func (d applicationDB) BeginTenantTx(ctx context.Context) (database.TenantTx, error) {
	tx, e := d.TenantBeginner.BeginTenantTx(ctx)
	if e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, `SET LOCAL ROLE nextgen_app`); e != nil {
		_ = tx.Rollback(ctx)
		return nil, e
	}
	return tx, nil
}

// routingWorkflow exercises the routing revision lifecycle against the real
// store and leaves exactly one active routing for the part.
func routingWorkflow(t *testing.T, ctx context.Context, s *Store, a Actor, kind string, part uuid.UUID) Routing {
	t.Helper()
	stamping := RoutingStep{Code: "STAMPING", Name: "Stamping", Rate: decimal.NewFromInt(250)}
	welding := RoutingStep{Code: "WELDING", Name: "Welding", Rate: decimal.NewFromInt(400)}
	first, e := s.CreateRouting(ctx, a, RoutingInput{PartID: part, Kind: kind, Name: "First routing", Steps: []RoutingStep{stamping}})
	if e != nil || !first.Active || first.Revision != 1 {
		t.Fatalf("first routing: %+v %v", first, e)
	}
	if _, e = s.CreateRouting(ctx, a, RoutingInput{PartID: part, Kind: kind, Name: "No operations", Steps: nil}); e == nil {
		t.Fatal("routing without operations accepted")
	}
	if _, e = s.CreateRouting(ctx, a, RoutingInput{PartID: uuid.New(), Kind: kind, Name: "Unknown part", Steps: []RoutingStep{stamping}}); !errors.Is(e, ErrInvalidReference) {
		t.Fatalf("routing for an unknown part: %v", e)
	}
	second, e := s.CreateRouting(ctx, a, RoutingInput{PartID: part, Kind: kind, Name: "Second routing", Steps: []RoutingStep{stamping, welding}})
	if e != nil || second.Active || second.Revision != 2 {
		t.Fatalf("second routing: %+v %v", second, e)
	}
	if _, e = s.RoutingAction(ctx, a, second.ID, "deactivate"); !errors.Is(e, ErrConflict) {
		t.Fatalf("deactivating an inactive routing: %v", e)
	}
	if second, e = s.UpdateRouting(ctx, a, second.ID, RoutingInput{Name: "Stamping and welding", Currency: "IDR", Steps: []RoutingStep{stamping, welding}}); e != nil {
		t.Fatal("unused routing could not be edited:", e)
	}
	if second, e = s.RoutingAction(ctx, a, second.ID, "activate"); e != nil || !second.Active || !second.TotalRate.Equal(decimal.NewFromInt(650)) {
		t.Fatalf("activate: %+v %v", second, e)
	}
	previous, e := s.GetRouting(ctx, a, first.ID)
	if e != nil || previous.Active {
		t.Fatalf("activation left two active routings: %+v %v", previous, e)
	}
	if _, e = s.RoutingAction(ctx, a, first.ID, "delete"); e != nil {
		t.Fatal("unused routing could not be deleted:", e)
	}
	if _, e = s.GetRouting(ctx, a, first.ID); !errors.Is(e, ErrNotFound) {
		t.Fatalf("deleted routing still readable: %v", e)
	}
	return second
}
