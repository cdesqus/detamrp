package production

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"order-stock/backend/internal/database"
)

// Remaining quantity of a lot: its own quantity, less what later movements drew
// from it, less its own reversal. Reversals of children carry the same lot, so
// they give their quantity back automatically.
const lotRemaining = `m.quantity
 - COALESCE((SELECT sum(c.quantity) FROM production_wip_movements c WHERE c.tenant_id=m.tenant_id AND c.lot_movement_id=m.id),0)
 + COALESCE((SELECT sum(r.quantity) FROM production_wip_movements r WHERE r.tenant_id=m.tenant_id AND r.reverses_id=m.id),0)`

const movementSelect = `SELECT m.id,m.production_order_id,o.order_number,m.movement_type,
 COALESCE(m.source_operation_id,'` + nilUUID + `'),COALESCE(src.operation_code,''),
 COALESCE(m.destination_operation_id,'` + nilUUID + `'),COALESCE(dst.operation_code,''),
 m.quantity,m.unit_cost,m.total_cost,m.currency,
 COALESCE(m.entry_id,'` + nilUUID + `'),COALESCE(e.entry_number,''),
 COALESCE(m.lot_movement_id,'` + nilUUID + `'),COALESCE(m.reverses_id,'` + nilUUID + `'),
 EXISTS(SELECT 1 FROM production_wip_movements rv WHERE rv.tenant_id=m.tenant_id AND rv.reverses_id=m.id),
 COALESCE((SELECT sum(c.quantity) FROM production_wip_movements c WHERE c.tenant_id=m.tenant_id AND c.lot_movement_id=m.id),0),
 m.movement_date::text,m.notes,u.display_name,m.created_at
 FROM production_wip_movements m
 JOIN production_orders o ON o.tenant_id=m.tenant_id AND o.id=m.production_order_id
 JOIN users u ON u.tenant_id=m.tenant_id AND u.id=m.created_by_user_id
 LEFT JOIN production_operations src ON src.tenant_id=m.tenant_id AND src.id=m.source_operation_id
 LEFT JOIN production_operations dst ON dst.tenant_id=m.tenant_id AND dst.id=m.destination_operation_id
 LEFT JOIN production_entries e ON e.tenant_id=m.tenant_id AND e.id=m.entry_id`

func scanMovement(rows interface{ Scan(...any) error }) (m WIPMovement, err error) {
	var openChildren decimal.Decimal
	err = rows.Scan(&m.ID, &m.OrderID, &m.OrderNumber, &m.Type, &m.SourceOperationID, &m.SourceCode,
		&m.DestinationOperationID, &m.DestinationCode, &m.Quantity, &m.UnitCost, &m.TotalCost, &m.Currency,
		&m.EntryID, &m.EntryNumber, &m.LotID, &m.ReversesID, &m.Reversed, &openChildren,
		&m.MovementDate, &m.Notes, &m.CreatedBy, &m.CreatedAt)
	if err != nil {
		return
	}
	// Only an untouched transfer can be taken back; consumption and receipts are
	// reversed by correcting their production entry.
	m.CanReverse = m.Type == MovementTransfer && m.ReversesID == uuid.Nil && !m.Reversed && openChildren.IsZero()
	return
}

func loadMovements(ctx context.Context, tx database.TenantTx, a Actor, orderID uuid.UUID) (items []WIPMovement, err error) {
	items = []WIPMovement{}
	rows, e := tx.Query(ctx, movementSelect+` WHERE m.tenant_id=$1 AND m.production_order_id=$2 ORDER BY m.movement_date DESC,m.created_at DESC,m.id`, a.TenantID, orderID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		m, e := scanMovement(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

const balanceSelect = `SELECT op.id,op.operation_code,op.operation_name,op.sequence_no,
 op.sequence_no=(SELECT max(last.sequence_no) FROM production_operations last WHERE last.tenant_id=op.tenant_id AND last.production_order_id=op.production_order_id),
 COALESCE((SELECT sum(e.qty_good) FROM production_entries e WHERE e.tenant_id=op.tenant_id AND e.operation_id=op.id AND e.voided_at IS NULL),0),
 COALESCE((SELECT sum(e.qty_processed) FROM production_entries e WHERE e.tenant_id=op.tenant_id AND e.operation_id=op.id AND e.voided_at IS NULL),0),
 COALESCE((SELECT sum(m.quantity) FROM production_wip_movements m WHERE m.tenant_id=op.tenant_id AND m.movement_type='RECEIPT' AND m.destination_operation_id=op.id),0),
 COALESCE((SELECT sum(m.quantity) FROM production_wip_movements m WHERE m.tenant_id=op.tenant_id AND m.movement_type='TRANSFER' AND m.source_operation_id=op.id),0),
 COALESCE((SELECT sum(m.quantity) FROM production_wip_movements m WHERE m.tenant_id=op.tenant_id AND m.movement_type='TRANSFER' AND m.destination_operation_id=op.id),0),
 COALESCE((SELECT sum(m.quantity) FROM production_wip_movements m WHERE m.tenant_id=op.tenant_id AND m.movement_type='CONSUMPTION' AND m.source_operation_id=op.id),0),
 COALESCE((SELECT sum((` + lotRemaining + `)*m.unit_cost) FROM production_wip_movements m WHERE m.tenant_id=op.tenant_id AND m.reverses_id IS NULL AND m.movement_type='RECEIPT' AND m.destination_operation_id=op.id),0),
 COALESCE((SELECT sum((` + lotRemaining + `)*m.unit_cost) FROM production_wip_movements m WHERE m.tenant_id=op.tenant_id AND m.reverses_id IS NULL AND m.movement_type='TRANSFER' AND m.destination_operation_id=op.id),0)
 FROM production_operations op WHERE op.tenant_id=$1 AND op.production_order_id=$2 ORDER BY op.sequence_no`

func loadWIPBalances(ctx context.Context, tx database.TenantTx, a Actor, orderID uuid.UUID) ([]WIPOperationBalance, error) {
	rows, e := tx.Query(ctx, balanceSelect, a.TenantID, orderID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	balances := []WIPOperationBalance{}
	for rows.Next() {
		var b WIPOperationBalance
		var onHandValue, stagedValue decimal.Decimal
		if e = rows.Scan(&b.OperationID, &b.Code, &b.Name, &b.Sequence, &b.Final, &b.GoodQty, &b.ProcessedQty,
			&b.Received, &b.TransferredOut, &b.TransferredIn, &b.Consumed, &onHandValue, &stagedValue); e != nil {
			return nil, e
		}
		b.reconcile()
		b.Value = onHandValue.Add(stagedValue).Round(6)
		balances = append(balances, b)
	}
	return balances, rows.Err()
}

func loadWIPOrder(ctx context.Context, tx database.TenantTx, a Actor, orderID uuid.UUID, withMovements bool) (v WIPOrderBalance, err error) {
	v.Movements = []WIPMovement{}
	if err = tx.QueryRow(ctx, `SELECT o.id,o.order_number,p.plan_number,o.part_number,o.part_name,o.unit_code,COALESCE(pl.name,''),o.status,o.currency,o.planned_qty,o.updated_at
 FROM production_orders o JOIN production_plans p ON p.tenant_id=o.tenant_id AND p.id=o.plan_id
 LEFT JOIN plants pl ON pl.tenant_id=o.tenant_id AND pl.id=o.plant_id WHERE o.tenant_id=$1 AND o.id=$2`, a.TenantID, orderID).
		Scan(&v.OrderID, &v.OrderNumber, &v.PlanNumber, &v.PartNumber, &v.PartName, &v.UnitCode, &v.PlantName, &v.Status, &v.Currency, &v.PlannedQty, &v.UpdatedAt); err != nil {
		return
	}
	if v.Operations, err = loadWIPBalances(ctx, tx, a, orderID); err != nil {
		return
	}
	rates, err := loadRates(ctx, tx, a)
	if err != nil {
		return
	}
	source := v.Currency
	v.Currency = ReportingCurrency
	for index := range v.Operations {
		v.Operations[index].Value = rates.mustConvert(v.Operations[index].Value, source)
	}
	v.Reconciled = true
	v.TotalQty, v.TotalValue = decimal.Zero, decimal.Zero
	for _, b := range v.Operations {
		v.TotalQty = v.TotalQty.Add(b.Balance)
		v.TotalValue = v.TotalValue.Add(b.Value)
		if !b.Reconciled {
			v.Reconciled = false
		}
	}
	if withMovements {
		v.Movements, err = loadMovements(ctx, tx, a, orderID)
	}
	return
}

func (s *Store) ListWIP(ctx context.Context, a Actor) (items []WIPOrderBalance, err error) {
	items = []WIPOrderBalance{}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT id FROM production_orders WHERE tenant_id=$1 AND status<>'CANCELLED' ORDER BY order_number`, a.TenantID)
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
			v, e := loadWIPOrder(ctx, tx, a, id, false)
			if e != nil {
				return e
			}
			items = append(items, v)
		}
		return nil
	})
	return
}

func (s *Store) GetWIP(ctx context.Context, a Actor, orderID uuid.UUID) (v WIPOrderBalance, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		var e error
		v, e = loadWIPOrder(ctx, tx, a, orderID, true)
		return e
	})
	return
}

// openLots lists the costed lots of one kind still holding quantity, oldest first.
func openLots(ctx context.Context, tx database.TenantTx, a Actor, orderID, operationID uuid.UUID, kind MovementType, date string) ([]Lot, error) {
	rows, e := tx.Query(ctx, `SELECT m.id,m.unit_cost,m.movement_date::text,`+lotRemaining+`
 FROM production_wip_movements m
 WHERE m.tenant_id=$1 AND m.production_order_id=$2 AND m.destination_operation_id=$3 AND m.movement_type=$4 AND m.reverses_id IS NULL AND m.movement_date<=$5::date
 ORDER BY m.movement_date,m.movement_sequence`, a.TenantID, orderID, operationID, string(kind), date)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	lots := []Lot{}
	for rows.Next() {
		var lot Lot
		if e = rows.Scan(&lot.ID, &lot.UnitCost, &lot.Date, &lot.Remaining); e != nil {
			return nil, e
		}
		lots = append(lots, lot)
	}
	return lots, rows.Err()
}

type movementRow struct {
	Type         MovementType
	Source       uuid.UUID
	Destination  uuid.UUID
	Quantity     decimal.Decimal
	UnitCost     decimal.Decimal
	Currency     string
	EntryID      uuid.UUID
	LotID        uuid.UUID
	ReversesID   uuid.UUID
	MovementDate string
	Notes        string
}

func insertMovement(ctx context.Context, tx database.TenantTx, a Actor, orderID uuid.UUID, row movementRow) (uuid.UUID, error) {
	id := uuid.New()
	_, e := tx.Exec(ctx, `INSERT INTO production_wip_movements(id,tenant_id,production_order_id,movement_type,source_operation_id,destination_operation_id,quantity,unit_cost,total_cost,currency,entry_id,lot_movement_id,reverses_id,movement_date,notes,created_by_user_id)
 VALUES($1,$2,$3,$4,NULLIF($5::uuid,'`+nilUUID+`'),NULLIF($6::uuid,'`+nilUUID+`'),$7,$8,$9,$10,NULLIF($11::uuid,'`+nilUUID+`'),NULLIF($12::uuid,'`+nilUUID+`'),NULLIF($13::uuid,'`+nilUUID+`'),$14::date,$15,$16)`,
		id, a.TenantID, orderID, string(row.Type), row.Source, row.Destination, row.Quantity, row.UnitCost,
		row.Quantity.Mul(row.UnitCost).Round(6), row.Currency, row.EntryID, row.LotID, row.ReversesID,
		row.MovementDate, trimNotes(row.Notes), a.UserID)
	return id, e
}

// postEntryMovements turns one posted entry into ledger movements: it consumes
// staged WIP for later operations and receipts the good output of every
// operation that still has a successor.
func postEntryMovements(ctx context.Context, tx database.TenantTx, a Actor, o Order, index int, entryID uuid.UUID, date string, processed, good, inputCost, processCost decimal.Decimal) error {
	operation := o.Operations[index]
	consumedCost := inputCost
	if index > 0 {
		lots, e := openLots(ctx, tx, a, o.ID, operation.ID, MovementTransfer, date)
		if e != nil {
			return e
		}
		allocations, e := allocateFIFO(lots, processed)
		if e != nil {
			return invalid("Not enough WIP has been transferred to %s; transfer stock from %s first", operation.Code, o.Operations[index-1].Code)
		}
		consumedCost = allocationCost(allocations)
		for _, allocation := range allocations {
			if _, e = insertMovement(ctx, tx, a, o.ID, movementRow{
				Type: MovementConsumption, Source: operation.ID, Quantity: allocation.Quantity,
				UnitCost: allocation.UnitCost, Currency: o.Currency, EntryID: entryID, LotID: allocation.LotID,
				MovementDate: date, Notes: "Consumed by " + operation.Code,
			}); e != nil {
				return e
			}
		}
	}
	if index == len(o.Operations)-1 || !good.IsPositive() {
		return nil
	}
	_, e := insertMovement(ctx, tx, a, o.ID, movementRow{
		Type: MovementReceipt, Destination: operation.ID, Quantity: good,
		UnitCost: unitCost(consumedCost.Add(processCost), good), Currency: o.Currency, EntryID: entryID,
		MovementDate: date, Notes: "Good output of " + operation.Code,
	})
	return e
}

// reverseEntryMovements annuls every ledger row of an entry. Output that has
// already moved downstream cannot be annulled, which is what locks the entry.
func reverseEntryMovements(ctx context.Context, tx database.TenantTx, a Actor, orderID, entryID uuid.UUID) error {
	rows, e := tx.Query(ctx, `SELECT m.id,m.movement_type,COALESCE(m.source_operation_id,'`+nilUUID+`'),COALESCE(m.destination_operation_id,'`+nilUUID+`'),m.quantity,m.unit_cost,m.currency,COALESCE(m.lot_movement_id,'`+nilUUID+`'),m.movement_date::text,
 COALESCE((SELECT sum(c.quantity) FROM production_wip_movements c WHERE c.tenant_id=m.tenant_id AND c.lot_movement_id=m.id),0)
 FROM production_wip_movements m WHERE m.tenant_id=$1 AND m.entry_id=$2 AND m.reverses_id IS NULL
 AND NOT EXISTS(SELECT 1 FROM production_wip_movements rv WHERE rv.tenant_id=m.tenant_id AND rv.reverses_id=m.id)`, a.TenantID, entryID)
	if e != nil {
		return e
	}
	pending := []movementRow{}
	ids := []uuid.UUID{}
	for rows.Next() {
		var row movementRow
		var id uuid.UUID
		var drawn decimal.Decimal
		if e = rows.Scan(&id, &row.Type, &row.Source, &row.Destination, &row.Quantity, &row.UnitCost, &row.Currency, &row.LotID, &row.MovementDate, &drawn); e != nil {
			rows.Close()
			return e
		}
		if !drawn.IsZero() {
			rows.Close()
			return invalid("This output has already been transferred; reverse the WIP transfer before correcting the entry")
		}
		row.Quantity = row.Quantity.Neg()
		row.ReversesID = id
		row.EntryID = entryID
		row.Notes = "Reversal"
		pending = append(pending, row)
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, row := range pending {
		if _, e = insertMovement(ctx, tx, a, orderID, row); e != nil {
			return e
		}
	}
	return nil
}

// refreshEntryLocks re-derives which entries may still be corrected: an entry is
// locked while any of the WIP it produced has moved on, and is released again
// once those movements are reversed.
func refreshEntryLocks(ctx context.Context, tx database.TenantTx, a Actor, orderID uuid.UUID) error {
	_, e := tx.Exec(ctx, `UPDATE production_entries e SET effects_locked=state.locked,updated_at=now(),updated_by_user_id=$2
 FROM (SELECT entries.id,EXISTS(
   SELECT 1 FROM production_wip_movements lot
   WHERE lot.tenant_id=entries.tenant_id AND lot.entry_id=entries.id AND lot.reverses_id IS NULL AND lot.movement_type='RECEIPT'
   AND COALESCE((SELECT sum(c.quantity) FROM production_wip_movements c WHERE c.tenant_id=lot.tenant_id AND c.lot_movement_id=lot.id),0)<>0) AS locked
  FROM production_entries entries
  JOIN production_operations op ON op.tenant_id=entries.tenant_id AND op.id=entries.operation_id
  WHERE entries.tenant_id=$1 AND op.production_order_id=$3 AND entries.voided_at IS NULL) state
 WHERE e.tenant_id=$1 AND e.id=state.id AND e.effects_locked IS DISTINCT FROM state.locked`, a.TenantID, a.UserID, orderID)
	return e
}

func (s *Store) CreateTransfer(ctx context.Context, a Actor, input TransferInput) (v WIPOrderBalance, err error) {
	if e := input.Validate(); e != nil {
		return v, e
	}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		o, e := lockOrder(ctx, tx, a, input.OrderID)
		if e != nil {
			return e
		}
		if o.Status == OrderCancelled || o.Status == OrderCompleted {
			return invalid("WIP cannot be moved on a %s order", o.Status)
		}
		if input.MovementDate < o.PeriodStart || input.MovementDate > o.DueDate {
			return invalid("Movement date must be inside the order period")
		}
		if e = lockEntryPeriod(ctx, tx, a, input.MovementDate); e != nil {
			return e
		}
		index := -1
		for n, operation := range o.Operations {
			if operation.ID == input.SourceOperationID {
				index = n
				break
			}
		}
		if index < 0 {
			return invalid("Operation does not belong to the order routing")
		}
		if index == len(o.Operations)-1 {
			return invalid("%s is the final operation; its good output is finished production, not WIP", o.Operations[index].Code)
		}
		destination := o.Operations[index+1]
		lots, e := openLots(ctx, tx, a, o.ID, input.SourceOperationID, MovementReceipt, input.MovementDate)
		if e != nil {
			return e
		}
		allocations, e := allocateFIFO(lots, input.Quantity)
		if e != nil {
			return e
		}
		for _, allocation := range allocations {
			for _, lot := range lots {
				if lot.ID == allocation.LotID && input.MovementDate < lot.Date {
					return invalid("Movement date cannot be earlier than the WIP it draws from (%s)", lot.Date)
				}
			}
			if _, e = insertMovement(ctx, tx, a, o.ID, movementRow{
				Type: MovementTransfer, Source: input.SourceOperationID, Destination: destination.ID,
				Quantity: allocation.Quantity, UnitCost: allocation.UnitCost, Currency: o.Currency,
				LotID: allocation.LotID, MovementDate: input.MovementDate, Notes: input.Notes,
			}); e != nil {
				return e
			}
		}
		if e = refreshEntryLocks(ctx, tx, a, o.ID); e != nil {
			return e
		}
		v, e = loadWIPOrder(ctx, tx, a, o.ID, true)
		return e
	})
	return
}

func (s *Store) ReverseMovement(ctx context.Context, a Actor, id uuid.UUID, notes string) (v WIPOrderBalance, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		var orderID uuid.UUID
		if e := tx.QueryRow(ctx, `SELECT production_order_id FROM production_wip_movements WHERE tenant_id=$1 AND id=$2`, a.TenantID, id).Scan(&orderID); e != nil {
			return e
		}
		o, e := lockOrder(ctx, tx, a, orderID)
		if e != nil {
			return e
		}
		movements, e := loadMovements(ctx, tx, a, orderID)
		if e != nil {
			return e
		}
		var target *WIPMovement
		for i := range movements {
			if movements[i].ID == id {
				target = &movements[i]
			}
		}
		if target == nil {
			return ErrNotFound
		}
		if !target.CanReverse {
			return invalid("Only an untouched transfer can be reversed; correct the production entry instead")
		}
		if e = lockEntryPeriod(ctx, tx, a, target.MovementDate); e != nil {
			return e
		}
		if _, e = insertMovement(ctx, tx, a, orderID, movementRow{
			Type: target.Type, Source: target.SourceOperationID, Destination: target.DestinationOperationID,
			Quantity: target.Quantity.Neg(), UnitCost: target.UnitCost, Currency: o.Currency,
			LotID: target.LotID, ReversesID: target.ID, MovementDate: target.MovementDate,
			Notes: trimNotes(notes),
		}); e != nil {
			return e
		}
		if e = refreshEntryLocks(ctx, tx, a, orderID); e != nil {
			return e
		}
		v, e = loadWIPOrder(ctx, tx, a, orderID, true)
		return e
	})
	return
}
