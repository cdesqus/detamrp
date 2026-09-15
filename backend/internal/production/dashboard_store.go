package production

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"order-stock/backend/internal/database"
)

// entryRange restricts an aggregate to posted entries inside the filter window.
const entryRange = ` FROM production_entries e
 JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id
 JOIN production_orders o ON o.tenant_id=op.tenant_id AND o.id=op.production_order_id
 WHERE e.tenant_id=$1 AND e.voided_at IS NULL AND e.production_date BETWEEN $2::date AND $3::date`

const finalOperation = ` AND op.sequence_no=(SELECT max(last.sequence_no) FROM production_operations last WHERE last.tenant_id=op.tenant_id AND last.production_order_id=op.production_order_id)`

func (s *Store) Dashboard(ctx context.Context, a Actor, filter DashboardFilter) (v ProductionDashboard, err error) {
	v = ProductionDashboard{Filter: filter, Parts: []PartPerformance{}, Processes: []ProcessWIP{},
		Quality: []OperationQuality{}, Costs: []PartCost{}, GeneratedAt: time.Now()}
	err = s.transaction(ctx, a, func(tx database.TenantTx) error {
		rates, e := loadRates(ctx, tx, a)
		if e != nil {
			return e
		}
		if e := loadDashboardTotals(ctx, tx, a, filter, rates, &v); e != nil {
			return e
		}
		if e := loadPartPerformance(ctx, tx, a, filter, &v); e != nil {
			return e
		}
		if e := loadProcessWIP(ctx, tx, a, rates, &v); e != nil {
			return e
		}
		if e := loadOperationQuality(ctx, tx, a, filter, &v); e != nil {
			return e
		}
		return loadPartCosts(ctx, tx, a, filter, rates, &v)
	})
	return
}

func loadDashboardTotals(ctx context.Context, tx database.TenantTx, a Actor, filter DashboardFilter, rates Rates, v *ProductionDashboard) error {
	t := &v.Totals
	t.Currency = ReportingCurrency
	if e := tx.QueryRow(ctx, `SELECT COALESCE(sum(e.qty_processed),0),COALESCE(sum(e.qty_rejected),0),count(*)`+
		entryRange, a.TenantID, filter.From, filter.To).
		Scan(&t.ProcessedQty, &t.RejectQty, &t.Entries); e != nil {
		return e
	}
	// Cost is summed per currency and converted, never added across denominations.
	costs, e := tx.Query(ctx, `SELECT e.currency,COALESCE(sum(e.material_cost),0),COALESCE(sum(e.process_cost),0)`+
		entryRange+` GROUP BY e.currency`, a.TenantID, filter.From, filter.To)
	if e != nil {
		return e
	}
	defer costs.Close()
	for costs.Next() {
		var currency string
		var material, process decimal.Decimal
		if e = costs.Scan(&currency, &material, &process); e != nil {
			return e
		}
		t.MaterialActual = t.MaterialActual.Add(rates.mustConvert(material, currency))
		t.ProcessActual = t.ProcessActual.Add(rates.mustConvert(process, currency))
	}
	if e = costs.Err(); e != nil {
		return e
	}
	if e := tx.QueryRow(ctx, `SELECT COALESCE(sum(e.qty_good),0)`+entryRange+finalOperation, a.TenantID, filter.From, filter.To).Scan(&t.GoodQty); e != nil {
		return e
	}
	// Orders whose scheduling window overlaps the filter carry the plan target.
	if e := tx.QueryRow(ctx, `SELECT COALESCE(sum(planned_qty),0),count(*),
 count(*) FILTER(WHERE status IN ('RELEASED','IN_PROGRESS','PARTIAL')),count(*) FILTER(WHERE status='COMPLETED')
 FROM production_orders WHERE tenant_id=$1 AND status<>'CANCELLED' AND period_start<=$3::date AND due_date>=$2::date`,
		a.TenantID, filter.From, filter.To).Scan(&t.PlannedQty, &t.ActiveOrders, &t.OpenOrders, &t.CompletedOrder); e != nil {
		return e
	}
	t.summarise()
	return nil
}

func loadPartPerformance(ctx context.Context, tx database.TenantTx, a Actor, filter DashboardFilter, v *ProductionDashboard) error {
	rows, e := tx.Query(ctx, `SELECT o.part_number,max(o.part_name),max(o.unit_code),count(DISTINCT o.id),COALESCE(sum(o.planned_qty),0),
 COALESCE(sum((SELECT sum(e.qty_good) FROM production_entries e JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id
   WHERE op.production_order_id=o.id AND e.voided_at IS NULL AND e.production_date BETWEEN $2::date AND $3::date
   AND op.sequence_no=(SELECT max(last.sequence_no) FROM production_operations last WHERE last.tenant_id=op.tenant_id AND last.production_order_id=o.id))),0),
 COALESCE(sum((SELECT sum(e.qty_rejected) FROM production_entries e JOIN production_operations op ON op.tenant_id=e.tenant_id AND op.id=e.operation_id
   WHERE op.production_order_id=o.id AND e.voided_at IS NULL AND e.production_date BETWEEN $2::date AND $3::date)),0)
 FROM production_orders o
 WHERE o.tenant_id=$1 AND o.status<>'CANCELLED' AND o.period_start<=$3::date AND o.due_date>=$2::date
 GROUP BY o.part_number ORDER BY sum(o.planned_qty) DESC,o.part_number LIMIT 12`, a.TenantID, filter.From, filter.To)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var part PartPerformance
		if e = rows.Scan(&part.PartNumber, &part.PartName, &part.UnitCode, &part.Orders, &part.PlannedQty, &part.GoodQty, &part.RejectQty); e != nil {
			return e
		}
		part.Achievement = ratio(part.GoodQty, part.PlannedQty)
		v.Parts = append(v.Parts, part)
	}
	return rows.Err()
}

// loadProcessWIP reports the live ledger balance per operation code: a transfer
// leaves its source and lands on its destination in the same pass.
func loadProcessWIP(ctx context.Context, tx database.TenantTx, a Actor, rates Rates, v *ProductionDashboard) error {
	rows, e := tx.Query(ctx, `SELECT code,currency,COALESCE(sum(qty),0),COALESCE(sum(value),0) FROM (
 SELECT op.operation_code AS code,o.currency,
  CASE WHEN m.movement_type='CONSUMPTION' AND m.source_operation_id=op.id THEN -m.quantity
       WHEN m.movement_type='TRANSFER' AND m.source_operation_id=op.id THEN -m.quantity
       WHEN m.destination_operation_id=op.id THEN m.quantity ELSE 0 END AS qty,
  CASE WHEN m.reverses_id IS NULL AND m.destination_operation_id=op.id AND m.movement_type IN ('RECEIPT','TRANSFER')
       THEN (`+lotRemaining+`)*m.unit_cost ELSE 0 END AS value
 FROM production_wip_movements m
 JOIN production_operations op ON op.tenant_id=m.tenant_id AND (op.id=m.source_operation_id OR op.id=m.destination_operation_id)
 JOIN production_orders o ON o.tenant_id=m.tenant_id AND o.id=m.production_order_id
 WHERE m.tenant_id=$1) balances GROUP BY code,currency ORDER BY code,currency`, a.TenantID)
	if e != nil {
		return e
	}
	defer rows.Close()
	// One row per operation, with foreign balances converted into the total.
	order := []string{}
	byCode := map[string]*ProcessWIP{}
	for rows.Next() {
		var code, currency string
		var quantity, value decimal.Decimal
		if e = rows.Scan(&code, &currency, &quantity, &value); e != nil {
			return e
		}
		process, seen := byCode[code]
		if !seen {
			process = &ProcessWIP{Code: code}
			byCode[code] = process
			order = append(order, code)
		}
		process.Quantity = process.Quantity.Add(quantity)
		process.Value = process.Value.Add(rates.mustConvert(value, currency)).Round(6)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	for _, code := range order {
		process := byCode[code]
		v.Totals.WIPQty = v.Totals.WIPQty.Add(process.Quantity)
		v.Totals.WIPValue = v.Totals.WIPValue.Add(process.Value).Round(6)
		if !process.Quantity.IsZero() || !process.Value.IsZero() {
			v.Processes = append(v.Processes, *process)
		}
	}
	return nil
}

func loadOperationQuality(ctx context.Context, tx database.TenantTx, a Actor, filter DashboardFilter, v *ProductionDashboard) error {
	rows, e := tx.Query(ctx, `SELECT op.operation_code,COALESCE(sum(e.qty_processed),0),COALESCE(sum(e.qty_good),0),COALESCE(sum(e.qty_rejected),0)`+
		entryRange+` GROUP BY op.operation_code ORDER BY op.operation_code`, a.TenantID, filter.From, filter.To)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var quality OperationQuality
		if e = rows.Scan(&quality.Code, &quality.ProcessedQty, &quality.GoodQty, &quality.RejectQty); e != nil {
			return e
		}
		quality.RejectRate = ratio(quality.RejectQty, quality.ProcessedQty)
		v.Quality = append(v.Quality, quality)
	}
	return rows.Err()
}

// loadPartCosts reports one row per part number, converting any foreign cost
// into the reporting currency instead of listing denominations side by side.
func loadPartCosts(ctx context.Context, tx database.TenantTx, a Actor, filter DashboardFilter, rates Rates, v *ProductionDashboard) error {
	rows, e := tx.Query(ctx, `SELECT o.part_number,max(o.part_name),count(DISTINCT o.id),
 COALESCE(sum(e.material_cost),0),COALESCE(sum(e.process_cost),0),e.currency,
 COALESCE(sum(e.qty_good) FILTER(WHERE op.sequence_no=(SELECT max(last.sequence_no) FROM production_operations last WHERE last.tenant_id=op.tenant_id AND last.production_order_id=o.id)),0)`+
		entryRange+` GROUP BY o.part_number,e.currency ORDER BY o.part_number,e.currency`, a.TenantID, filter.From, filter.To)
	if e != nil {
		return e
	}
	order := []string{}
	byPart := map[string]*PartCost{}
	for rows.Next() {
		var part, name, currency string
		var orders int
		var material, process, good decimal.Decimal
		if e = rows.Scan(&part, &name, &orders, &material, &process, &currency, &good); e != nil {
			rows.Close()
			return e
		}
		cost, seen := byPart[part]
		if !seen {
			cost = &PartCost{PartNumber: part, PartName: name, Currency: ReportingCurrency}
			byPart[part] = cost
			order = append(order, part)
		}
		cost.Orders += orders
		cost.GoodQty = cost.GoodQty.Add(good)
		cost.MaterialActual = cost.MaterialActual.Add(rates.mustConvert(material, currency))
		cost.ProcessActual = cost.ProcessActual.Add(rates.mustConvert(process, currency))
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, part := range order {
		cost := byPart[part]
		cost.TotalActual = cost.MaterialActual.Add(cost.ProcessActual)
		v.Costs = append(v.Costs, *cost)
	}
	// Unit costs are cumulative for the orders that posted in this window.
	for i := range v.Costs {
		c := &v.Costs[i]
		orders, err := tx.Query(ctx, costSelect+` WHERE o.tenant_id=$1 AND o.part_number=$4 AND EXISTS(SELECT 1 `+entryScope+` AND e.production_date BETWEEN $2::date AND $3::date)`, a.TenantID, filter.From, filter.To, c.PartNumber)
		if err != nil {
			return err
		}
		for orders.Next() {
			order, err := scanOrderCost(orders)
			if err != nil {
				orders.Close()
				return err
			}
			c.FinishedCost = c.FinishedCost.Add(rates.mustConvert(order.FinishedCost, order.Currency))
			c.LifetimeGoodQty = c.LifetimeGoodQty.Add(order.GoodQty)
		}
		err = orders.Err()
		orders.Close()
		if err != nil {
			return err
		}
		c.CostPerPiece = perUnit(c.FinishedCost, c.LifetimeGoodQty)
	}
	return nil
}

// DashboardValue keeps the quantity view usable when costs are not granted.
func (d ProductionDashboard) withoutCosts() ProductionDashboard {
	d.Totals.MaterialActual, d.Totals.ProcessActual, d.Totals.TotalActual, d.Totals.WIPValue = decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	d.Costs = []PartCost{}
	for index := range d.Processes {
		d.Processes[index].Value = decimal.Zero
	}
	return d
}
