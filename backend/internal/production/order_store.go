package production

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
	"strings"
)

type RoutingStep struct {
	Code string          `json:"code"`
	Name string          `json:"name"`
	Rate decimal.Decimal `json:"rate"`
}

func (s *Store) OrderOptions(ctx context.Context, a Actor) (out OrderOptions, err error) {
	masters, e := s.Options(ctx, a)
	if e != nil {
		return out, e
	}
	out = OrderOptions{Lines: []OrderLineOption{}, Parts: masters.Parts, Plants: masters.Plants}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT l.id,p.id,p.plan_number,l.part_number,l.part_name,l.unit_code,pl.name,p.period_start::text,p.period_end::text,l.planned_qty-COALESCE((SELECT sum(o.planned_qty) FROM production_orders o WHERE o.tenant_id=l.tenant_id AND o.plan_line_id=l.id AND o.status<>'CANCELLED'),0) FROM production_plan_lines l JOIN production_plans p ON p.tenant_id=l.tenant_id AND p.id=l.plan_id JOIN plants pl ON pl.tenant_id=p.tenant_id AND pl.id=p.plant_id WHERE p.tenant_id=$1 AND p.status='APPROVED' ORDER BY p.plan_number,l.sort_position`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var l OrderLineOption
			if e = rows.Scan(&l.ID, &l.PlanID, &l.PlanNumber, &l.PartNumber, &l.PartName, &l.UnitCode, &l.PlantName, &l.PeriodStart, &l.PeriodEnd, &l.RemainingQty); e != nil {
				return e
			}
			if l.RemainingQty.IsPositive() {
				out.Lines = append(out.Lines, l)
			}
		}
		return rows.Err()
	})
	return
}
func (s *Store) ListOrders(ctx context.Context, a Actor) (items []Order, err error) {
	items = []Order{}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT id FROM production_orders WHERE tenant_id=$1 ORDER BY updated_at DESC,id`, a.TenantID)
		if e != nil {
			return e
		}
		ids := []uuid.UUID{}
		for rows.Next() {
			var id uuid.UUID
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			o, e := loadOrder(ctx, tx, a, id)
			if e != nil {
				return e
			}
			items = append(items, o)
		}
		return nil
	})
	return
}
func (s *Store) GetOrder(ctx context.Context, a Actor, id uuid.UUID) (o Order, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error { var e error; o, e = loadOrder(ctx, tx, a, id); return e })
	return
}
func loadOrder(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) (o Order, err error) {
	var raw []byte
	o.Operations = []Operation{}
	o.Materials = []MaterialSnapshot{}
	o.History = []PlanHistory{}
	err = tx.QueryRow(ctx, `SELECT o.id,o.order_number,o.plan_id,COALESCE(o.plan_line_id,'00000000-0000-0000-0000-000000000000'),p.plan_number,COALESCE(o.routing_id,'00000000-0000-0000-0000-000000000000'),COALESCE(o.routing_revision,0),o.part_number,o.part_name,o.unit_code,COALESCE(pl.name,''),o.planned_qty,COALESCE(o.period_start,p.period_start)::text,COALESCE(o.due_date,p.period_end)::text,o.status,o.notes,COALESCE(o.bom_revision,0),o.currency,o.material_estimate,o.process_estimate,o.updated_at,u.display_name,COALESCE(o.bom_snapshot,'[]'::jsonb) FROM production_orders o JOIN production_plans p ON p.tenant_id=o.tenant_id AND p.id=o.plan_id LEFT JOIN plants pl ON pl.tenant_id=o.tenant_id AND pl.id=o.plant_id JOIN users u ON u.tenant_id=o.tenant_id AND u.id=o.created_by_user_id WHERE o.tenant_id=$1 AND o.id=$2`, a.TenantID, id).Scan(&o.ID, &o.OrderNumber, &o.PlanID, &o.PlanLineID, &o.PlanNumber, &o.RoutingID, &o.RoutingRevision, &o.PartNumber, &o.PartName, &o.UnitCode, &o.PlantName, &o.PlannedQty, &o.PeriodStart, &o.DueDate, &o.Status, &o.Notes, &o.BOMRevision, &o.Currency, &o.MaterialEstimate, &o.ProcessEstimate, &o.UpdatedAt, &o.CreatedBy, &raw)
	if err != nil {
		return
	}
	if err = json.Unmarshal(raw, &o.Materials); err != nil {
		return
	}
	rows, e := tx.Query(ctx, `SELECT op.id,op.operation_code,op.operation_name,op.sequence_no,op.rate_snapshot,op.planned_qty,COALESCE(sum(e.qty_processed) FILTER(WHERE e.voided_at IS NULL),0),COALESCE(sum(e.qty_good) FILTER(WHERE e.voided_at IS NULL),0),COALESCE(sum(e.qty_rejected) FILTER(WHERE e.voided_at IS NULL),0) FROM production_operations op LEFT JOIN production_entries e ON e.tenant_id=op.tenant_id AND e.operation_id=op.id WHERE op.tenant_id=$1 AND op.production_order_id=$2 GROUP BY op.id ORDER BY op.sequence_no`, a.TenantID, id)
	if e != nil {
		return o, e
	}
	for rows.Next() {
		var op Operation
		if e = rows.Scan(&op.ID, &op.Code, &op.Name, &op.Sequence, &op.Rate, &op.PlannedQty, &op.ProcessedQty, &op.GoodQty, &op.RejectQty); e != nil {
			rows.Close()
			return o, e
		}
		o.Operations = append(o.Operations, op)
		o.RejectQty = o.RejectQty.Add(op.RejectQty)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return o, e
	}
	if len(o.Operations) > 0 {
		o.ActualGood = o.Operations[len(o.Operations)-1].GoodQty
	}
	// Work in progress is whatever the ledger still holds: output waiting at an
	// operation plus stock transferred into one but not processed yet.
	balances, e := loadWIPBalances(ctx, tx, a, id)
	if e != nil {
		return o, e
	}
	byOperation := map[uuid.UUID]WIPOperationBalance{}
	for _, balance := range balances {
		byOperation[balance.OperationID] = balance
	}
	for index := range o.Operations {
		balance := byOperation[o.Operations[index].ID]
		o.Operations[index].OnHandQty = balance.OnHand
		o.Operations[index].StagedQty = balance.Staged
		o.Operations[index].WIPQty = balance.Balance
		o.WIPQty = o.WIPQty.Add(balance.Balance)
	}
	e = tx.QueryRow(ctx, `SELECT count(*),COALESCE(sum(e.material_cost) FILTER(WHERE e.voided_at IS NULL),0),COALESCE(sum(e.process_cost) FILTER(WHERE e.voided_at IS NULL),0) FROM production_entries e JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id WHERE op.tenant_id=$1 AND op.production_order_id=$2`, a.TenantID, id).Scan(&o.EntryCount, &o.MaterialCost, &o.ProcessCost)
	if e != nil {
		return o, e
	}
	o.History, e = loadHistory(ctx, tx, a, "production_orders", id)
	return o, e
}
func loadHistory(ctx context.Context, tx database.TenantTx, a Actor, table string, id uuid.UUID) (items []PlanHistory, err error) {
	items = []PlanHistory{}
	rows, e := tx.Query(ctx, `SELECT action,actor_name,occurred_at FROM activity_logs WHERE tenant_id=$1 AND target_type=$2 AND target_id=$3 ORDER BY occurred_at DESC,id DESC`, a.TenantID, table, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var h PlanHistory
		if e = rows.Scan(&h.Action, &h.Actor, &h.OccurredAt); e != nil {
			return nil, e
		}
		items = append(items, h)
	}
	return items, rows.Err()
}
func (s *Store) CreateOrder(ctx context.Context, a Actor, input OrderInput) (o Order, err error) {
	if e := input.Validate(true); e != nil {
		return o, e
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error { var e error; o, e = createOrder(ctx, tx, a, input); return e })
	return
}
func createOrder(ctx context.Context, tx database.TenantTx, a Actor, input OrderInput) (o Order, err error) {
	var planID uuid.UUID
	if e := tx.QueryRow(ctx, `SELECT plan_id FROM production_plan_lines WHERE tenant_id=$1 AND id=$2`, a.TenantID, input.PlanLineID).Scan(&planID); e != nil {
		return o, e
	}
	status, _, e := lockPlan(ctx, tx, a, planID)
	if e != nil {
		return o, e
	}
	if status != PlanApproved {
		return o, invalid("Only approved planning can create Production Orders")
	}
	var fg, rm, plant uuid.UUID
	var target, allocated decimal.Decimal
	var start, end, part, partName, unit string
	e = tx.QueryRow(ctx, `SELECT COALESCE(l.finished_good_id,'00000000-0000-0000-0000-000000000000'),COALESCE(l.raw_material_id,'00000000-0000-0000-0000-000000000000'),p.plant_id,l.planned_qty,p.period_start::text,p.period_end::text,l.part_number,l.part_name,l.unit_code FROM production_plan_lines l JOIN production_plans p ON p.tenant_id=l.tenant_id AND p.id=l.plan_id WHERE l.tenant_id=$1 AND l.id=$2`, a.TenantID, input.PlanLineID).Scan(&fg, &rm, &plant, &target, &start, &end, &part, &partName, &unit)
	if e != nil {
		return o, e
	}
	if input.DueDate < start || input.DueDate > end {
		return o, invalid("Due date must be within the planning period")
	}
	if e = tx.QueryRow(ctx, `SELECT COALESCE(sum(planned_qty),0) FROM production_orders WHERE tenant_id=$1 AND plan_line_id=$2 AND status<>'CANCELLED'`, a.TenantID, input.PlanLineID).Scan(&allocated); e != nil {
		return o, e
	}
	if allocated.Add(input.PlannedQty).GreaterThan(target) {
		return o, invalid("Quantity exceeds the unallocated planning target (%s)", target.Sub(allocated))
	}
	// Revalidate masters without trusting caller-supplied units or prices.
	p := validPlanForOrder(fg, rm, plant, start, end, input.PlannedQty)
	if e = resolvePlan(ctx, tx, a, &p); e != nil {
		return o, e
	}
	part = p.Lines[0].PartNumber
	partName = p.Lines[0].PartName
	unit = p.Lines[0].UnitCode
	var bomID, routingID uuid.UUID
	var revision, routingRevision int
	var stepsRaw []byte
	var currency string
	if e = tx.QueryRow(ctx, `SELECT id,revision FROM boms WHERE tenant_id=$1 AND status='ACTIVE' AND (finished_good_id=$2 OR raw_material_id=$3) FOR SHARE`, a.TenantID, fg, rm).Scan(&bomID, &revision); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return o, invalid("An active BOM is required for this part")
		}
		return o, e
	}
	if e = tx.QueryRow(ctx, `SELECT id,revision,currency,steps FROM production_routings WHERE tenant_id=$1 AND active AND (finished_good_id=$2 OR raw_material_id=$3) FOR SHARE`, a.TenantID, fg, rm).Scan(&routingID, &routingRevision, &currency, &stepsRaw); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return o, invalid("An active routing is required for this part")
		}
		return o, e
	}
	// Everything is reported in Rupiah, so an order may only be released in a
	// currency the tenant can convert.
	rates, e := loadRates(ctx, tx, a)
	if e != nil {
		return o, e
	}
	if _, ok := rates.rate(currency); !ok {
		return o, invalid("Set a conversion rate for %s before releasing production in that currency", currency)
	}
	var steps []RoutingStep
	if e = json.Unmarshal(stepsRaw, &steps); e != nil {
		return o, e
	}
	if len(steps) == 0 {
		return o, invalid("Routing requires at least one operation")
	}
	materials := []MaterialSnapshot{}
	materialUnit, processUnit := decimal.Zero, decimal.Zero
	rows, e := tx.Query(ctx, `SELECT r.id,r.code,r.name,u.code,c.usage_qty,r.standard_unit_price,trim(r.currency),r.active FROM bom_components c JOIN raw_materials r ON r.tenant_id=c.tenant_id AND r.id=c.raw_material_id JOIN units u ON u.tenant_id=r.tenant_id AND u.id=r.base_unit_id WHERE c.tenant_id=$1 AND c.bom_id=$2 ORDER BY c.sort_position FOR SHARE OF r,u`, a.TenantID, bomID)
	if e != nil {
		return o, e
	}
	for rows.Next() {
		var m MaterialSnapshot
		var active bool
		if e = rows.Scan(&m.ID, &m.PartNumber, &m.PartName, &m.UnitCode, &m.UsageQty, &m.UnitPrice, &m.Currency, &active); e != nil {
			rows.Close()
			return o, e
		}
		if !active || m.Currency != currency {
			rows.Close()
			return o, invalid("BOM materials must be active and use routing currency %s", currency)
		}
		materials = append(materials, m)
		materialUnit = materialUnit.Add(m.UsageQty.Mul(m.UnitPrice))
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return o, e
	}
	if len(materials) == 0 {
		return o, invalid("Active BOM has no materials")
	}
	for _, step := range steps {
		processUnit = processUnit.Add(step.Rate)
	}
	raw, e := json.Marshal(materials)
	if e != nil {
		return o, e
	}
	id := uuid.New()
	var sequence int64
	if e = tx.QueryRow(ctx, `INSERT INTO production_plan_counters(tenant_id,period,last_number) VALUES($1,'ORDER',1) ON CONFLICT(tenant_id,period) DO UPDATE SET last_number=production_plan_counters.last_number+1 RETURNING last_number`, a.TenantID).Scan(&sequence); e != nil {
		return o, e
	}
	number := fmt.Sprintf("PRO-%07d", sequence)
	_, e = tx.Exec(ctx, `INSERT INTO production_orders(id,tenant_id,order_number,plan_id,plan_line_id,finished_good_id,raw_material_id,planned_qty,bom_revision,bom_snapshot,notes,created_by_user_id,updated_by_user_id,routing_id,routing_revision,part_number,part_name,unit_code,plant_id,period_start,due_date,currency,material_estimate,process_estimate) VALUES($1,$2,$3,$4,$5,NULLIF($6::uuid,'00000000-0000-0000-0000-000000000000'),NULLIF($7::uuid,'00000000-0000-0000-0000-000000000000'),$8,$9,$10,$11,$12,$12,$13,$14,$15,$16,$17,$18,$19::date,$20::date,$21,$22,$23)`, id, a.TenantID, number, planID, input.PlanLineID, fg, rm, input.PlannedQty, revision, raw, input.Notes, a.UserID, routingID, routingRevision, part, partName, unit, plant, start, input.DueDate, currency, materialUnit.Mul(input.PlannedQty), processUnit.Mul(input.PlannedQty))
	if e != nil {
		return o, e
	}
	for i, step := range steps {
		if _, e = tx.Exec(ctx, `INSERT INTO production_operations(tenant_id,production_order_id,operation_code,operation_name,sequence_no,planned_qty,rate_snapshot) VALUES($1,$2,$3,$4,$5,$6,$7)`, a.TenantID, id, step.Code, step.Name, i+1, input.PlannedQty, step.Rate); e != nil {
			return o, e
		}
	}
	if e = touchPlan(ctx, tx, a, planID); e != nil {
		return o, e
	}
	return loadOrder(ctx, tx, a, id)
}
func validPlanForOrder(fg, rm, plant uuid.UUID, start, end string, qty decimal.Decimal) Plan {
	p := Plan{PlantID: plant, Lines: []PlanLine{{FinishedGoodID: fg, RawMaterialID: rm, PlannedQty: qty}}}
	p.PeriodStart, _ = parseDate(start)
	p.PeriodEnd, _ = parseDate(end)
	return p
}
func touchPlan(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) error {
	_, e := tx.Exec(ctx, `UPDATE production_plans SET updated_at=now(),updated_by_user_id=$3 WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, a.UserID)
	return e
}

// Plan first, then order is the lock order shared by allocations and mutations.
func lockOrder(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) (Order, error) {
	var p uuid.UUID
	if e := tx.QueryRow(ctx, `SELECT plan_id FROM production_orders WHERE tenant_id=$1 AND id=$2`, a.TenantID, id).Scan(&p); e != nil {
		return Order{}, e
	}
	if _, _, e := lockPlan(ctx, tx, a, p); e != nil {
		return Order{}, e
	}
	var locked uuid.UUID
	if e := tx.QueryRow(ctx, `SELECT id FROM production_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, id).Scan(&locked); e != nil {
		return Order{}, e
	}
	return loadOrder(ctx, tx, a, id)
}
func (s *Store) UpdateOrder(ctx context.Context, a Actor, id uuid.UUID, input OrderInput) (o Order, err error) {
	if e := input.Validate(false); e != nil {
		return o, e
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		old, e := lockOrder(ctx, tx, a, id)
		if e != nil {
			return e
		}
		if !canEditOrder(old.Status, old.EntryCount) {
			return invalid("Only orders without production entries can be edited")
		}
		var target, other decimal.Decimal
		var end string
		if e = tx.QueryRow(ctx, `SELECT l.planned_qty,p.period_end::text FROM production_plan_lines l JOIN production_plans p ON p.tenant_id=l.tenant_id AND p.id=l.plan_id WHERE l.tenant_id=$1 AND l.id=$2`, a.TenantID, old.PlanLineID).Scan(&target, &end); e != nil {
			return e
		}
		if input.DueDate < old.PeriodStart || input.DueDate > end {
			return invalid("Due date must be within the planning period")
		}
		if e = tx.QueryRow(ctx, `SELECT COALESCE(sum(planned_qty),0) FROM production_orders WHERE tenant_id=$1 AND plan_line_id=$2 AND id<>$3 AND status<>'CANCELLED'`, a.TenantID, old.PlanLineID, id).Scan(&other); e != nil {
			return e
		}
		if other.Add(input.PlannedQty).GreaterThan(target) {
			return invalid("Quantity exceeds unallocated planning target")
		}
		material, process := decimal.Zero, decimal.Zero
		for _, m := range old.Materials {
			material = material.Add(m.UsageQty.Mul(m.UnitPrice))
		}
		for _, op := range old.Operations {
			process = process.Add(op.Rate)
		}
		if _, e = tx.Exec(ctx, `UPDATE production_orders SET planned_qty=$3,due_date=$4::date,notes=$5,material_estimate=$6,process_estimate=$7,updated_at=now(),updated_by_user_id=$8 WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, input.PlannedQty, input.DueDate, input.Notes, material.Mul(input.PlannedQty), process.Mul(input.PlannedQty), a.UserID); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE production_operations SET planned_qty=$3 WHERE tenant_id=$1 AND production_order_id=$2`, a.TenantID, id, input.PlannedQty); e != nil {
			return e
		}
		if e = touchPlan(ctx, tx, a, old.PlanID); e != nil {
			return e
		}
		o, e = loadOrder(ctx, tx, a, id)
		return e
	})
	return
}
func (s *Store) OrderAction(ctx context.Context, a Actor, id uuid.UUID, action, reason string) (o Order, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		old, e := lockOrder(ctx, tx, a, id)
		if e != nil {
			return e
		}
		switch action {
		case "delete":
			if !canEditOrder(old.Status, old.EntryCount) {
				return invalid("Only unused orders can be deleted")
			}
			_, e = tx.Exec(ctx, `DELETE FROM production_orders WHERE tenant_id=$1 AND id=$2`, a.TenantID, id)
			if e != nil {
				return e
			}
			return touchPlan(ctx, tx, a, old.PlanID)
		case "start":
			if old.Status != OrderReleased && old.Status != OrderPartial {
				return invalid("Only released or partial orders can start or resume")
			}
			old.Status = OrderInProgress
		case "cancel":
			if old.Status == OrderCompleted || old.Status == OrderCancelled {
				return invalid("This order cannot be cancelled")
			}
			if strings.TrimSpace(reason) == "" {
				return invalid("Cancellation reason is required")
			}
			if old.WIPQty.IsPositive() {
				return invalid("Reconcile WIP before cancelling this order")
			}
			old.Status = OrderCancelled
		default:
			return invalid("Unknown order action")
		}
		if _, e = tx.Exec(ctx, `UPDATE production_orders SET status=$3,cancel_reason=$4,updated_at=now(),updated_by_user_id=$5 WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, old.Status, reason, a.UserID); e != nil {
			return e
		}
		if e = touchPlan(ctx, tx, a, old.PlanID); e != nil {
			return e
		}
		o, e = loadOrder(ctx, tx, a, id)
		return e
	})
	return
}
