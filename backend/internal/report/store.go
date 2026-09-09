package report

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/shopspring/decimal"
	"order-stock/backend/internal/bom"
	"order-stock/backend/internal/database"
)

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db: db} }

func (s *Store) ListMaterialRequirements(ctx context.Context, actor Actor) (items []MaterialRequirementRow, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT l.quantity,l.calculation_snapshot FROM sales_order_lines l JOIN sales_orders s ON s.tenant_id=l.tenant_id AND s.id=l.sales_order_id WHERE l.tenant_id=$1 AND s.status='SUBMITTED'`, actor.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		grouped := map[string]*MaterialRequirementRow{}
		for rows.Next() {
			var quantity decimal.Decimal
			var raw []byte
			if e = rows.Scan(&quantity, &raw); e != nil {
				return e
			}
			var snapshot bom.Snapshot
			if e = json.Unmarshal(raw, &snapshot); e != nil {
				return e
			}
			result, e := bom.Calculate(snapshot, quantity)
			if e != nil {
				return e
			}
			for _, material := range result.Materials {
				row := grouped[material.ItemID]
				if row == nil {
					row = &MaterialRequirementRow{ItemCode: material.ItemCode, ItemName: material.Name, Unit: material.Unit, QtyPerKanban: material.QtyPerKanban}
					grouped[material.ItemID] = row
				}
				row.Required = row.Required.Add(material.Quantity)
			}
		}
		if e = rows.Err(); e != nil {
			return e
		}
		for _, row := range grouped {
			if row.QtyPerKanban.IsPositive() {
				whole, remainder := row.Required.QuoRem(row.QtyPerKanban, 0)
				row.PurchaseKanban = whole
				if !remainder.IsZero() {
					row.PurchaseKanban = row.PurchaseKanban.Add(decimal.NewFromInt(1))
				}
			}
			items = append(items, *row)
		}
		return nil
	})
	return
}

func (s *Store) ListReceiving(ctx context.Context, actor Actor, filter Filter) (Result, error) {
	result := Result{Items: []Row{}}
	err := database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		rows, err := tx.Query(ctx, `
SELECT r.receiving_number,r.receiving_date,dn.delivery_note_number,p.po_number,s.name,
 pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,pol.base_unit_code_snapshot,
 COUNT(rkl.kanban_lot_id),COALESCE(SUM(rkl.quantity),0),
 COALESCE((SELECT SUM(k2.quantity) FROM kanban_lots k2 WHERE k2.tenant_id=r.tenant_id AND k2.purchase_order_line_id=pol.id AND k2.status='ISSUED'),0),
 COALESCE(r.sage_receipt_number,''),u.display_name
FROM receivings r
JOIN delivery_notes dn ON dn.tenant_id=r.tenant_id AND dn.id=r.delivery_note_id
JOIN purchase_orders p ON p.tenant_id=r.tenant_id AND p.id=r.purchase_order_id
JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id
JOIN users u ON u.tenant_id=r.tenant_id AND u.id=r.completed_by_user_id
JOIN receiving_kanban_lots rkl ON rkl.tenant_id=r.tenant_id AND rkl.receiving_id=r.id
JOIN kanban_lots kl ON kl.tenant_id=rkl.tenant_id AND kl.id=rkl.kanban_lot_id
JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id
WHERE r.tenant_id=$1
 AND ($2::date IS NULL OR r.receiving_date >= $2)
 AND ($3::date IS NULL OR r.receiving_date <= $3)
 AND ($4::uuid IS NULL OR s.id=$4)
 AND ($5='' OR r.receiving_number ILIKE '%'||$5||'%' OR dn.delivery_note_number ILIKE '%'||$5||'%' OR p.po_number ILIKE '%'||$5||'%')
GROUP BY r.id,dn.delivery_note_number,p.po_number,s.name,pol.id,u.display_name
ORDER BY r.receiving_date DESC,r.receiving_number,pol.raw_material_code_snapshot`,
			actor.TenantID, filter.FromDate, filter.ToDate, filter.SupplierID, strings.TrimSpace(filter.Search))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var row Row
			if err := rows.Scan(&row.ReceivingNumber, &row.ReceivingDate, &row.DeliveryNoteNumber, &row.PONumber, &row.SupplierName,
				&row.RawMaterialCode, &row.RawMaterialName, &row.BaseUnitCode, &row.KanbanReceived, &row.ReceivedQuantity,
				&row.OutstandingQuantity, &row.SageNumber, &row.CreatedBy); err != nil {
				return err
			}
			result.Items = append(result.Items, row)
		}
		return rows.Err()
	})
	if err != nil {
		return Result{}, err
	}
	result.Totals = summarize(result.Items)
	return result, nil
}
func (s *Store) ListSalesOrders(ctx context.Context, actor Actor) (items []SalesOrderRow, err error) {
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT s.sales_order_number,c.name,s.order_date,s.status,l.item_code_snapshot,l.item_name_snapshot,l.quantity,COALESCE((SELECT sum(d.quantity) FROM customer_delivery_lines d WHERE d.tenant_id=l.tenant_id AND d.sales_order_line_id=l.id),0),l.quantity-COALESCE((SELECT sum(d.quantity) FROM customer_delivery_lines d WHERE d.tenant_id=l.tenant_id AND d.sales_order_line_id=l.id),0),l.base_unit_snapshot FROM sales_orders s JOIN customers c ON c.tenant_id=s.tenant_id AND c.id=s.customer_id JOIN sales_order_lines l ON l.tenant_id=s.tenant_id AND l.sales_order_id=s.id WHERE s.tenant_id=$1 ORDER BY s.order_date DESC,s.sales_order_number,l.sort_position`, actor.TenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var item SalesOrderRow
			if e = rows.Scan(&item.Number, &item.Customer, &item.OrderDate, &item.Status, &item.ItemCode, &item.ItemName, &item.Ordered, &item.Delivered, &item.Remaining, &item.Unit); e != nil {
				return e
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return
}
