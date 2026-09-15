package production

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"order-stock/backend/internal/database"
)

// entryScope restricts an aggregate to the posted entries of one order.
const entryScope = ` FROM production_entries e JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id
 WHERE op.tenant_id=o.tenant_id AND op.production_order_id=o.id AND e.voided_at IS NULL`

// Cost still held in work in progress: every lot that has quantity left, valued
// at the cost frozen when it was produced or moved.
const wipValueScope = `COALESCE((SELECT sum((` + lotRemaining + `)*m.unit_cost) FROM production_wip_movements m
 WHERE m.tenant_id=o.tenant_id AND m.production_order_id=o.id AND m.reverses_id IS NULL AND m.movement_type IN ('RECEIPT','TRANSFER')),0)`

const costSelect = `SELECT o.id,o.order_number,p.plan_number,o.part_number,o.part_name,o.unit_code,COALESCE(pl.name,''),o.status,o.currency,
 o.period_start::text,o.due_date::text,o.planned_qty,o.material_estimate,o.process_estimate,o.updated_at,
 COALESCE((SELECT sum(e.material_cost)` + entryScope + `),0),
 COALESCE((SELECT sum(e.process_cost)` + entryScope + `),0),
 COALESCE((SELECT count(*)` + entryScope + `),0),
 COALESCE((SELECT sum(e.qty_good)` + entryScope + ` AND op.sequence_no=(SELECT max(last.sequence_no) FROM production_operations last WHERE last.tenant_id=op.tenant_id AND last.production_order_id=op.production_order_id)),0),
 COALESCE((SELECT sum(e.qty_rejected)` + entryScope + `),0),
 ` + wipValueScope + `
 FROM production_orders o JOIN production_plans p ON p.tenant_id=o.tenant_id AND p.id=o.plan_id
 LEFT JOIN plants pl ON pl.tenant_id=o.tenant_id AND pl.id=o.plant_id`

func scanOrderCost(row interface{ Scan(...any) error }) (c OrderCost, err error) {
	c.Operations = []OperationCost{}
	c.Materials = []MaterialCost{}
	err = row.Scan(&c.OrderID, &c.OrderNumber, &c.PlanNumber, &c.PartNumber, &c.PartName, &c.UnitCode, &c.PlantName,
		&c.Status, &c.Currency, &c.PeriodStart, &c.DueDate, &c.PlannedQty, &c.MaterialEstimate, &c.ProcessEstimate,
		&c.UpdatedAt, &c.MaterialActual, &c.ProcessActual, &c.EntryCount, &c.GoodQty, &c.RejectQty, &c.WIPValue)
	if err != nil {
		return
	}
	c.summarise()
	return
}

func (s *Store) ListOrderCosts(ctx context.Context, a Actor) (items []OrderCost, err error) {
	items = []OrderCost{}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rates, e := loadRates(ctx, tx, a)
		if e != nil {
			return e
		}
		rows, e := tx.Query(ctx, costSelect+` WHERE o.tenant_id=$1 ORDER BY o.order_number`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			c, e := scanOrderCost(rows)
			if e != nil {
				return e
			}
			c.convertTo(rates)
			items = append(items, c)
		}
		return rows.Err()
	})
	return
}

func (s *Store) GetOrderCost(ctx context.Context, a Actor, id uuid.UUID) (c OrderCost, err error) {
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		var e error
		if c, e = scanOrderCost(tx.QueryRow(ctx, costSelect+` WHERE o.tenant_id=$1 AND o.id=$2`, a.TenantID, id)); e != nil {
			return e
		}
		if c.Operations, e = loadOperationCosts(ctx, tx, a, c); e != nil {
			return e
		}
		if c.Materials, e = loadMaterialCosts(ctx, tx, a, c); e != nil {
			return e
		}
		rates, e := loadRates(ctx, tx, a)
		if e != nil {
			return e
		}
		c.convertTo(rates)
		return nil
	})
	return
}

func loadOperationCosts(ctx context.Context, tx database.TenantTx, a Actor, c OrderCost) ([]OperationCost, error) {
	rows, e := tx.Query(ctx, `SELECT op.id,op.operation_code,op.operation_name,op.sequence_no,
 op.sequence_no=(SELECT max(last.sequence_no) FROM production_operations last WHERE last.tenant_id=op.tenant_id AND last.production_order_id=op.production_order_id),
 op.rate_snapshot,
 COALESCE(sum(e.qty_processed) FILTER(WHERE e.voided_at IS NULL),0),
 COALESCE(sum(e.qty_good) FILTER(WHERE e.voided_at IS NULL),0),
 COALESCE(sum(e.qty_rejected) FILTER(WHERE e.voided_at IS NULL),0),
 COALESCE(sum(e.process_cost) FILTER(WHERE e.voided_at IS NULL),0),
 count(e.id) FILTER(WHERE e.voided_at IS NULL)
 FROM production_operations op LEFT JOIN production_entries e ON e.tenant_id=op.tenant_id AND e.operation_id=op.id
 WHERE op.tenant_id=$1 AND op.production_order_id=$2 GROUP BY op.id,op.operation_code,op.operation_name,op.sequence_no,op.rate_snapshot ORDER BY op.sequence_no`, a.TenantID, c.OrderID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	operations := []OperationCost{}
	for rows.Next() {
		var operation OperationCost
		if e = rows.Scan(&operation.OperationID, &operation.Code, &operation.Name, &operation.Sequence, &operation.Final,
			&operation.RateSnapshot, &operation.ProcessedQty, &operation.GoodQty, &operation.RejectQty,
			&operation.ActualCost, &operation.Entries); e != nil {
			return nil, e
		}
		operation.EstimateCost = operation.RateSnapshot.Mul(c.PlannedQty).Round(6)
		operation.CostPerPiece = perUnit(operation.ActualCost, operation.GoodQty)
		operations = append(operations, operation)
	}
	return operations, rows.Err()
}

// loadMaterialCosts sums the usage snapshots of every posted entry, so master
// price changes never rewrite what an order already cost.
func loadMaterialCosts(ctx context.Context, tx database.TenantTx, a Actor, c OrderCost) ([]MaterialCost, error) {
	started := decimal.Zero
	for _, operation := range c.Operations {
		if operation.Sequence == 1 {
			started = operation.ProcessedQty
		}
	}
	rows, e := tx.Query(ctx, `SELECT (usage->>'materialId')::uuid,max(usage->>'partNumber'),max(usage->>'partName'),max(usage->>'unitCode'),
 sum((usage->>'quantity')::numeric),sum((usage->>'cost')::numeric)
 FROM production_entries e JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id,
 LATERAL jsonb_array_elements(e.material_usage) usage
 WHERE e.tenant_id=$1 AND op.production_order_id=$2 AND e.voided_at IS NULL
 GROUP BY 1 ORDER BY 2`, a.TenantID, c.OrderID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	materials := []MaterialCost{}
	for rows.Next() {
		var material MaterialCost
		if e = rows.Scan(&material.MaterialID, &material.PartNumber, &material.PartName, &material.UnitCode,
			&material.UsedQty, &material.ActualCost); e != nil {
			return nil, e
		}
		material.AveragePrice = perUnit(material.ActualCost, material.UsedQty)
		materials = append(materials, material)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	return withStandardUsage(ctx, tx, a, c, started, materials)
}

// withStandardUsage adds the BOM expectation for the quantity actually started,
// which is what the usage variance is measured against.
func withStandardUsage(ctx context.Context, tx database.TenantTx, a Actor, c OrderCost, started decimal.Decimal, materials []MaterialCost) ([]MaterialCost, error) {
	rows, e := tx.Query(ctx, `SELECT (component->>'id')::uuid,(component->>'usageQty')::numeric
 FROM production_orders o, LATERAL jsonb_array_elements(COALESCE(o.bom_snapshot,'[]'::jsonb)) component
 WHERE o.tenant_id=$1 AND o.id=$2`, a.TenantID, c.OrderID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	expected := map[uuid.UUID]decimal.Decimal{}
	for rows.Next() {
		var id uuid.UUID
		var usage decimal.Decimal
		if e = rows.Scan(&id, &usage); e != nil {
			return nil, e
		}
		expected[id] = usage.Mul(started).Round(6)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	for index := range materials {
		materials[index].StandardQty = expected[materials[index].MaterialID]
		materials[index].QtyVariance = materials[index].UsedQty.Sub(materials[index].StandardQty)
	}
	return materials, nil
}

// ListPeriodCosts totals every production month and reports whether it is still
// open for postings.
func (s *Store) ListPeriodCosts(ctx context.Context, a Actor) (items []PeriodCost, err error) {
	items = []PeriodCost{}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rates, e := loadRates(ctx, tx, a)
		if e != nil {
			return e
		}
		rows, e := tx.Query(ctx, `SELECT months.period,
 COALESCE(p.closed_at IS NOT NULL,false),COALESCE(u.display_name,''),
 months.orders,months.entries,months.processed,months.good,months.rejected,months.material,months.process,months.currency
 FROM (SELECT to_char(e.production_date,'YYYY-MM') AS period,count(DISTINCT op.production_order_id) AS orders,count(*) AS entries,
  sum(e.qty_processed) AS processed,sum(e.qty_good) AS good,sum(e.qty_rejected) AS rejected,
  sum(e.material_cost) AS material,sum(e.process_cost) AS process,e.currency AS currency
  FROM production_entries e JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id
  WHERE e.tenant_id=$1 AND e.voided_at IS NULL GROUP BY 1,e.currency) months
 LEFT JOIN production_periods p ON p.tenant_id=$1 AND p.period=months.period
 LEFT JOIN users u ON u.tenant_id=$1 AND u.id=p.closed_by_user_id
 ORDER BY months.period DESC,months.currency`, a.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		order := []string{}
		byPeriod := map[string]*PeriodCost{}
		for rows.Next() {
			var row PeriodCost
			var currency string
			if e = rows.Scan(&row.Period, &row.Closed, &row.ClosedBy, &row.Orders, &row.Entries,
				&row.ProcessedQty, &row.GoodQty, &row.RejectQty, &row.MaterialActual, &row.ProcessActual, &currency); e != nil {
				return e
			}
			period, seen := byPeriod[row.Period]
			if !seen {
				period = &PeriodCost{Period: row.Period, Closed: row.Closed, ClosedBy: row.ClosedBy, Currency: ReportingCurrency}
				byPeriod[row.Period] = period
				order = append(order, row.Period)
			}
			period.Orders += row.Orders
			period.Entries += row.Entries
			period.ProcessedQty = period.ProcessedQty.Add(row.ProcessedQty)
			period.GoodQty = period.GoodQty.Add(row.GoodQty)
			period.RejectQty = period.RejectQty.Add(row.RejectQty)
			period.MaterialActual = period.MaterialActual.Add(rates.mustConvert(row.MaterialActual, currency))
			period.ProcessActual = period.ProcessActual.Add(rates.mustConvert(row.ProcessActual, currency))
		}
		if e = rows.Err(); e != nil {
			return e
		}
		for _, month := range order {
			period := byPeriod[month]
			period.TotalActual = period.MaterialActual.Add(period.ProcessActual)
			items = append(items, *period)
		}
		return nil
	})
	return
}
