package report

import (
	"context"
	"strings"

	"order-stock/backend/internal/database"
)

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db: db} }

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
