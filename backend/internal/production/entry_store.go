package production

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
	"strings"
)

const entrySelect = `SELECT e.id,e.entry_number,o.id,e.operation_id,e.production_date::text,e.shift_code,e.operator_user_id,e.qty_processed,e.qty_good,e.qty_rejected,e.notes,e.version,
 o.order_number,o.part_number,o.part_name,o.unit_code,COALESCE(pl.name,''),op.operation_name,op.operation_code,op.sequence_no,u.display_name,
 CASE WHEN e.voided_at IS NULL THEN 'POSTED' ELSE 'VOIDED' END,e.void_reason,e.currency,e.material_cost,e.process_cost,e.process_rate_snapshot,e.updated_at,creator.display_name,
 EXISTS(SELECT 1 FROM production_periods p WHERE p.tenant_id=e.tenant_id AND p.period=to_char(e.production_date,'YYYY-MM') AND p.closed_at IS NOT NULL),e.effects_locked,e.material_usage,
 o.status
 FROM production_entries e JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id
 JOIN production_orders o ON o.tenant_id=op.tenant_id AND o.id=op.production_order_id
 JOIN users u ON u.tenant_id=e.tenant_id AND u.id=e.operator_user_id
 JOIN users creator ON creator.tenant_id=e.tenant_id AND creator.id=e.created_by_user_id
 LEFT JOIN plants pl ON pl.tenant_id=o.tenant_id AND pl.id=o.plant_id`

func scanEntry(row pgx.Row) (v Entry, err error) {
	var raw []byte
	var orderStatus OrderStatus
	err = row.Scan(&v.ID, &v.EntryNumber, &v.OrderID, &v.OperationID, &v.ProductionDate, &v.Shift, &v.OperatorID, &v.Processed, &v.Good, &v.Rejected, &v.Notes, &v.Version, &v.OrderNumber, &v.PartNumber, &v.PartName, &v.UnitCode, &v.PlantName, &v.OperationName, &v.OperationCode, &v.Sequence, &v.OperatorName, &v.Status, &v.VoidReason, &v.Currency, &v.MaterialCost, &v.ProcessCost, &v.ProcessRate, &v.UpdatedAt, &v.CreatedBy, &v.PeriodClosed, &v.EffectsLocked, &raw, &orderStatus)
	if err != nil {
		return
	}
	v.Materials = []MaterialUsage{}
	v.History = []PlanHistory{}
	err = json.Unmarshal(raw, &v.Materials)
	v.CanCorrect = v.Status == "POSTED" && !v.PeriodClosed && !v.EffectsLocked && orderStatus != OrderCancelled
	return
}
func loadEntry(ctx context.Context, tx database.TenantTx, a Actor, id uuid.UUID) (v Entry, err error) {
	v, err = scanEntry(tx.QueryRow(ctx, entrySelect+` WHERE e.tenant_id=$1 AND e.id=$2`, a.TenantID, id))
	if err != nil {
		return
	}
	v.History, err = loadHistory(ctx, tx, a, "production_entries", id)
	return
}
func (s *Store) ListEntries(ctx context.Context, a Actor) (items []Entry, err error) {
	items = []Entry{}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, entrySelect+` WHERE e.tenant_id=$1 ORDER BY e.production_date DESC,e.created_at DESC,e.id`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			v, e := scanEntry(rows)
			if e != nil {
				return e
			}
			items = append(items, v)
		}
		return rows.Err()
	})
	return
}
func (s *Store) GetEntry(ctx context.Context, a Actor, id uuid.UUID) (v Entry, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error { var e error; v, e = loadEntry(ctx, tx, a, id); return e })
	return
}
func (s *Store) EntryOptions(ctx context.Context, a Actor) (v EntryOptions, err error) {
	v.Operators = []OperatorOption{}
	v.ClosedPeriods = []string{}
	v.Orders, err = s.ListOrders(ctx, a)
	if err != nil {
		return
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT id,COALESCE(NULLIF(display_name,''),username) FROM users WHERE tenant_id=$1 AND NOT locked ORDER BY display_name`, a.TenantID)
		if e != nil {
			return e
		}
		for rows.Next() {
			var u OperatorOption
			if e = rows.Scan(&u.ID, &u.Name); e != nil {
				rows.Close()
				return e
			}
			v.Operators = append(v.Operators, u)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		rows, e = tx.Query(ctx, `SELECT period FROM production_periods WHERE tenant_id=$1 AND closed_at IS NOT NULL ORDER BY period`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var p string
			if e = rows.Scan(&p); e != nil {
				return e
			}
			v.ClosedPeriods = append(v.ClosedPeriods, p)
		}
		return rows.Err()
	})
	return
}

func (s *Store) CreateEntry(ctx context.Context, a Actor, i EntryInput) (v Entry, err error) {
	if e := i.Validate(false); e != nil {
		return v, e
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		o, e := lockOrder(ctx, tx, a, i.OrderID)
		if e != nil {
			return e
		}
		if o.Status != OrderReleased && o.Status != OrderInProgress {
			return invalid("Only RELEASED or IN_PROGRESS orders accept new entries; resume a PARTIAL order first")
		}
		if e = lockEntryPeriod(ctx, tx, a, i.ProductionDate); e != nil {
			return e
		}
		index, e := validateEntryAgainstOrder(ctx, tx, a, o, i, nil)
		if e != nil {
			return e
		}
		materials, materialCost, rate, e := entryCosts(ctx, tx, a, o, index, i, nil)
		if e != nil {
			return e
		}
		rawUsage, e := json.Marshal(materials)
		if e != nil {
			return e
		}
		processCost := i.Processed.Mul(rate).Round(6)
		if e = validateEntryCosts(materialCost, processCost); e != nil {
			return e
		}
		var number int64
		if e = tx.QueryRow(ctx, `INSERT INTO production_plan_counters(tenant_id,period,last_number) VALUES($1,'ENTRY',1) ON CONFLICT(tenant_id,period) DO UPDATE SET last_number=production_plan_counters.last_number+1 RETURNING last_number`, a.TenantID).Scan(&number); e != nil {
			return e
		}
		id := uuid.New()
		_, e = tx.Exec(ctx, `INSERT INTO production_entries(id,tenant_id,entry_number,operation_id,production_date,shift_code,qty_processed,qty_good,qty_rejected,operator_user_id,notes,created_by_user_id,updated_by_user_id,material_usage,material_cost,process_cost,process_rate_snapshot,currency) VALUES($1,$2,$3,$4,$5::date,$6,$7,$8,$9,$10,$11,$12,$12,$13,$14,$15,$16,$17)`, id, a.TenantID, fmt.Sprintf("DP-%07d", number), i.OperationID, i.ProductionDate, strings.TrimSpace(i.Shift), i.Processed, i.Good, i.Rejected, i.OperatorID, strings.TrimSpace(i.Notes), a.UserID, string(rawUsage), materialCost, processCost, rate, o.Currency)
		if e != nil {
			return e
		}
		if e = postEntryMovements(ctx, tx, a, o, index, id, i.ProductionDate, i.Processed, i.Good, materialCost, processCost); e != nil {
			return e
		}
		if e = issueMaterialToInventory(ctx, tx, a, id, materials); e != nil {
			return e
		}
		if e = refreshEntryProgress(ctx, tx, a, o); e != nil {
			return e
		}
		v, e = loadEntry(ctx, tx, a, id)
		return e
	})
	return
}

func validateEntryAgainstOrder(ctx context.Context, tx database.TenantTx, a Actor, o Order, i EntryInput, old *Entry) (int, error) {
	if i.OrderID != o.ID {
		return 0, invalid("Order reference cannot be changed")
	}
	if i.ProductionDate < o.PeriodStart || i.ProductionDate > o.DueDate {
		return 0, invalid("Production date must be inside the order period")
	}
	var active uuid.UUID
	if e := tx.QueryRow(ctx, `SELECT id FROM users WHERE tenant_id=$1 AND id=$2 AND NOT locked FOR SHARE`, a.TenantID, i.OperatorID).Scan(&active); e != nil {
		if e == pgx.ErrNoRows {
			return 0, invalid("Select an active operator from your company")
		}
		return 0, e
	}
	index := -1
	for n, op := range o.Operations {
		if op.ID == i.OperationID {
			index = n
			break
		}
	}
	if index < 0 {
		return 0, invalid("Operation does not belong to the order routing")
	}
	if old != nil {
		if old.OperationID != i.OperationID {
			return 0, invalid("Operation cannot change during an edit; void and create a new entry")
		}
		o.Operations[index].ProcessedQty = o.Operations[index].ProcessedQty.Sub(old.Processed)
		if index > 0 {
			// Correcting an entry gives its consumed WIP back to the operation.
			o.Operations[index].StagedQty = o.Operations[index].StagedQty.Add(old.Processed)
		}
	}
	remaining := operationRemaining(o, index)
	if index > 0 {
		excluded := uuid.Nil
		if old != nil {
			excluded = old.ID
		}
		// Later operations may only work on WIP transferred to them by that date.
		rows, e := tx.Query(ctx, `SELECT day::text,COALESCE(sum(supplied),0),COALESCE(sum(processed),0) FROM (
 SELECT m.movement_date AS day,m.quantity AS supplied,0::numeric AS processed FROM production_wip_movements m WHERE m.tenant_id=$1 AND m.movement_type='TRANSFER' AND m.destination_operation_id=$2
 UNION ALL
 SELECT e.production_date,0::numeric,e.qty_processed FROM production_entries e WHERE e.tenant_id=$1 AND e.operation_id=$2 AND e.id<>$3 AND e.voided_at IS NULL
 ) timeline GROUP BY day ORDER BY day`, a.TenantID, i.OperationID, excluded)
		if e != nil {
			return 0, e
		}
		days := []OperationDay{}
		for rows.Next() {
			var day OperationDay
			if e = rows.Scan(&day.Date, &day.Supplied, &day.Processed); e != nil {
				rows.Close()
				return 0, e
			}
			days = append(days, day)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return 0, e
		}
		if !inputTimelineAvailable(days, i.ProductionDate, i.Processed) {
			return 0, invalid("Not enough WIP had been transferred to %s on this production date", o.Operations[index].Code)
		}
	}
	if i.Processed.GreaterThan(remaining) {
		if index > 0 {
			return 0, invalid("Processed quantity exceeds the WIP staged at %s (%s); transfer stock from %s first", o.Operations[index].Code, decimal.Max(decimal.Zero, remaining), o.Operations[index-1].Code)
		}
		return 0, invalid("Processed quantity exceeds the remaining order target (%s)", decimal.Max(decimal.Zero, remaining))
	}
	return index, nil
}
func (s *Store) UpdateEntry(ctx context.Context, a Actor, id uuid.UUID, i EntryInput) (v Entry, err error) {
	if e := i.Validate(true); e != nil {
		return v, e
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		old, e := loadEntry(ctx, tx, a, id)
		if e != nil {
			return e
		}
		o, e := lockOrder(ctx, tx, a, old.OrderID)
		if e != nil {
			return e
		}
		old, e = loadEntry(ctx, tx, a, id)
		if e != nil {
			return e
		}
		if !old.CanCorrect {
			return invalid("Entry cannot be edited: period closed, voided, or downstream activity exists")
		}
		if old.Version != i.Version {
			return invalid("This entry changed; reload before saving")
		}
		if e = lockEntryPeriod(ctx, tx, a, old.ProductionDate); e != nil {
			return e
		}
		if i.ProductionDate[:7] != old.ProductionDate[:7] {
			if e = lockEntryPeriod(ctx, tx, a, i.ProductionDate); e != nil {
				return e
			}
		}
		index, e := validateEntryAgainstOrder(ctx, tx, a, o, i, &old)
		if e != nil {
			return e
		}
		materials, materialCost, rate, e := entryCosts(ctx, tx, a, o, index, i, &old)
		if e != nil {
			return e
		}
		rawUsage, e := json.Marshal(materials)
		if e != nil {
			return e
		}
		processCost := i.Processed.Mul(rate).Round(6)
		if e = validateEntryCosts(materialCost, processCost); e != nil {
			return e
		}
		// The ledger rows of the previous version are annulled before the
		// corrected quantities are posted again.
		if e = reverseEntryMovements(ctx, tx, a, o.ID, id); e != nil {
			return e
		}
		if e = returnMaterialToInventory(ctx, tx, a, id); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE production_entries SET production_date=$3::date,shift_code=$4,qty_processed=$5,qty_good=$6,qty_rejected=$7,operator_user_id=$8,notes=$9,material_usage=$10,material_cost=$11,process_cost=$12,updated_by_user_id=$13,updated_at=now(),version=version+1 WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, i.ProductionDate, strings.TrimSpace(i.Shift), i.Processed, i.Good, i.Rejected, i.OperatorID, strings.TrimSpace(i.Notes), string(rawUsage), materialCost, processCost, a.UserID)
		if e != nil {
			return e
		}
		if e = postEntryMovements(ctx, tx, a, o, index, id, i.ProductionDate, i.Processed, i.Good, materialCost, processCost); e != nil {
			return e
		}
		if e = issueMaterialToInventory(ctx, tx, a, id, materials); e != nil {
			return e
		}
		if e = refreshEntryProgress(ctx, tx, a, o); e != nil {
			return e
		}
		v, e = loadEntry(ctx, tx, a, id)
		return e
	})
	return
}
func (s *Store) VoidEntry(ctx context.Context, a Actor, id uuid.UUID, version int, reason string) (v Entry, err error) {
	if strings.TrimSpace(reason) == "" || len(reason) > 2000 {
		return v, invalid("A void reason is required, up to 2000 characters")
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		old, e := loadEntry(ctx, tx, a, id)
		if e != nil {
			return e
		}
		o, e := lockOrder(ctx, tx, a, old.OrderID)
		if e != nil {
			return e
		}
		old, e = loadEntry(ctx, tx, a, id)
		if e != nil {
			return e
		}
		if !old.CanCorrect {
			return invalid("Entry cannot be voided: period closed, already voided, or downstream activity exists")
		}
		if old.Version != version {
			return invalid("This entry changed; reload before voiding")
		}
		if e = lockEntryPeriod(ctx, tx, a, old.ProductionDate); e != nil {
			return e
		}
		if e = reverseEntryMovements(ctx, tx, a, o.ID, id); e != nil {
			return e
		}
		if e = returnMaterialToInventory(ctx, tx, a, id); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE production_entries SET voided_at=now(),voided_by_user_id=$3,void_reason=$4,updated_by_user_id=$3,updated_at=now(),version=version+1 WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, a.UserID, strings.TrimSpace(reason)); e != nil {
			return e
		}
		if e = refreshEntryProgress(ctx, tx, a, o); e != nil {
			return e
		}
		v, e = loadEntry(ctx, tx, a, id)
		return e
	})
	return
}
func refreshEntryProgress(ctx context.Context, tx database.TenantTx, a Actor, previous Order) error {
	o, e := loadOrder(ctx, tx, a, previous.ID)
	if e != nil {
		return e
	}
	status := derivedOrderStatus(o)
	if _, e = tx.Exec(ctx, `UPDATE production_operations op SET completed_qty=COALESCE((SELECT sum(e.qty_good) FROM production_entries e WHERE e.tenant_id=op.tenant_id AND e.operation_id=op.id AND e.voided_at IS NULL),0) WHERE op.tenant_id=$1 AND op.production_order_id=$2`, a.TenantID, o.ID); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE production_orders SET status=$3,updated_at=now(),updated_by_user_id=$4 WHERE tenant_id=$1 AND id=$2`, a.TenantID, o.ID, status, a.UserID); e != nil {
		return e
	}
	return touchPlan(ctx, tx, a, o.PlanID)
}
