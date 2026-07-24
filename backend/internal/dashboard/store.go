package dashboard

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"order-stock/backend/internal/database"
)

var ErrSupplierNotFound = errors.New("supplier not found")

type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db: db} }

func (s *Store) Snapshot(ctx context.Context, actor Actor, filter Filter) (snapshot Snapshot, err error) {
	snapshot = Snapshot{
		Filter: filter, Trend: []TrendPoint{}, POStatus: []StatusPoint{},
		OutstandingBySupplier: []SupplierPoint{}, Activities: []Activity{},
	}
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		if filter.SupplierID != uuid.Nil {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers WHERE tenant_id=$1 AND id=$2)`, actor.TenantID, filter.SupplierID).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return ErrSupplierNotFound
			}
		}
		args := []any{actor.TenantID, filter.From, filter.To, filter.SupplierID}
		supplierWhere := `($4::uuid='00000000-0000-0000-0000-000000000000'::uuid OR p.supplier_id=$4)`

		metricSQL := fmt.Sprintf(`
SELECT
 count(*) FILTER(WHERE p.status='PENDING_APPROVAL'),
 count(*) FILTER(WHERE p.status IN('APPROVED','PARTIALLY_RECEIVED'))
FROM purchase_orders p
WHERE p.tenant_id=$1 AND p.order_date BETWEEN $2 AND $3 AND %s`, supplierWhere)
		if err := tx.QueryRow(ctx, metricSQL, args...).Scan(&snapshot.Metrics.PendingApproval, &snapshot.Metrics.OpenPO); err != nil {
			return err
		}
		receivedSQL := fmt.Sprintf(`
SELECT count(DISTINCT rkl.kanban_lot_id)
FROM receivings r
JOIN receiving_kanban_lots rkl ON rkl.tenant_id=r.tenant_id AND rkl.receiving_id=r.id
JOIN purchase_orders p ON p.tenant_id=r.tenant_id AND p.id=r.purchase_order_id
WHERE r.tenant_id=$1 AND r.status='COMPLETED' AND r.receiving_date BETWEEN $2 AND $3 AND %s`, supplierWhere)
		if err := tx.QueryRow(ctx, receivedSQL, args...).Scan(&snapshot.Metrics.ReceivedKanban); err != nil {
			return err
		}
		lotSQL := fmt.Sprintf(`
SELECT
 count(*) FILTER(WHERE kl.status='ISSUED'),
 count(*) FILTER(WHERE kl.status='IN_STOCK')
FROM kanban_lots kl
JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id
JOIN purchase_orders p ON p.tenant_id=pol.tenant_id AND p.id=pol.purchase_order_id
WHERE kl.tenant_id=$1 AND %s
  AND (kl.status='IN_STOCK' OR p.order_date BETWEEN $2 AND $3)`, supplierWhere)
		if err := tx.QueryRow(ctx, lotSQL, args...).Scan(&snapshot.Metrics.OutstandingKanban, &snapshot.Metrics.CurrentStock); err != nil {
			return err
		}
		if err := s.loadTrend(ctx, tx, args, supplierWhere, &snapshot); err != nil {
			return err
		}
		if err := s.loadStatus(ctx, tx, args, supplierWhere, &snapshot); err != nil {
			return err
		}
		if err := s.loadOutstanding(ctx, tx, args, supplierWhere, &snapshot); err != nil {
			return err
		}
		return s.loadActivities(ctx, tx, args, supplierWhere, &snapshot)
	})
	return snapshot, err
}

func (s *Store) loadTrend(ctx context.Context, tx database.TenantTx, args []any, supplierWhere string, snapshot *Snapshot) error {
	query := fmt.Sprintf(`
WITH ordered AS (
 SELECT p.order_date AS day, sum(pol.total_kanban)::bigint AS count
 FROM purchase_orders p JOIN purchase_order_lines pol ON pol.tenant_id=p.tenant_id AND pol.purchase_order_id=p.id
 WHERE p.tenant_id=$1 AND p.order_date BETWEEN $2 AND $3 AND p.status NOT IN('REJECTED','CANCELLED') AND %s
 GROUP BY p.order_date
), received AS (
 SELECT r.receiving_date AS day, count(DISTINCT rkl.kanban_lot_id)::bigint AS count
 FROM receivings r
 JOIN receiving_kanban_lots rkl ON rkl.tenant_id=r.tenant_id AND rkl.receiving_id=r.id
 JOIN purchase_orders p ON p.tenant_id=r.tenant_id AND p.id=r.purchase_order_id
 WHERE r.tenant_id=$1 AND r.receiving_date BETWEEN $2 AND $3 AND %s
 GROUP BY r.receiving_date
)
SELECT coalesce(o.day,rc.day),coalesce(o.count,0),coalesce(rc.count,0)
FROM ordered o FULL JOIN received rc ON rc.day=o.day ORDER BY 1`, supplierWhere, supplierWhere)
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	points := map[string]TrendPoint{}
	for rows.Next() {
		var day time.Time
		var point TrendPoint
		if err := rows.Scan(&day, &point.Ordered, &point.Received); err != nil {
			return err
		}
		point.Date = day.Format(time.DateOnly)
		points[point.Date] = point
	}
	if err := rows.Err(); err != nil {
		return err
	}
	snapshot.Trend = completeTrend(snapshot.Filter.From, snapshot.Filter.To, points)
	return nil
}

func completeTrend(from, to time.Time, values map[string]TrendPoint) []TrendPoint {
	points := make([]TrendPoint, 0, int(to.Sub(from).Hours()/24)+1)
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		key := day.Format(time.DateOnly)
		point := values[key]
		point.Date = key
		points = append(points, point)
	}
	return points
}

func (s *Store) loadStatus(ctx context.Context, tx database.TenantTx, args []any, supplierWhere string, snapshot *Snapshot) error {
	rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT p.status,count(*) FROM purchase_orders p WHERE p.tenant_id=$1 AND p.order_date BETWEEN $2 AND $3 AND %s GROUP BY p.status ORDER BY p.status`, supplierWhere), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var point StatusPoint
		if err := rows.Scan(&point.Status, &point.Count); err != nil {
			return err
		}
		snapshot.POStatus = append(snapshot.POStatus, point)
	}
	return rows.Err()
}

func (s *Store) loadOutstanding(ctx context.Context, tx database.TenantTx, args []any, supplierWhere string, snapshot *Snapshot) error {
	query := fmt.Sprintf(`
SELECT s.name,count(*)::bigint
FROM kanban_lots kl
JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id
JOIN purchase_orders p ON p.tenant_id=pol.tenant_id AND p.id=pol.purchase_order_id
JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id
WHERE kl.tenant_id=$1 AND kl.status='ISSUED' AND p.order_date BETWEEN $2 AND $3 AND %s
GROUP BY s.id,s.name ORDER BY count(*) DESC,s.name LIMIT 10`, supplierWhere)
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var point SupplierPoint
		if err := rows.Scan(&point.Supplier, &point.Kanban); err != nil {
			return err
		}
		snapshot.OutstandingBySupplier = append(snapshot.OutstandingBySupplier, point)
	}
	return rows.Err()
}

func (s *Store) loadActivities(ctx context.Context, tx database.TenantTx, args []any, supplierWhere string, snapshot *Snapshot) error {
	query := fmt.Sprintf(`
SELECT id,type,label,occurred_at FROM (
 SELECT p.id::text AS id,'PO' AS type,p.po_number||' · '||s.name||' · '||replace(initcap(p.status),'_',' ') AS label,p.updated_at AS occurred_at
 FROM purchase_orders p JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id
 WHERE p.tenant_id=$1 AND p.order_date BETWEEN $2 AND $3 AND %s
 UNION ALL
 SELECT r.id::text,'RECEIVING',r.receiving_number||' · '||p.po_number||' · Received '||r.received_now_kanban||' Kanban',r.completed_at
 FROM receivings r JOIN purchase_orders p ON p.tenant_id=r.tenant_id AND p.id=r.purchase_order_id
 WHERE r.tenant_id=$1 AND r.receiving_date BETWEEN $2 AND $3 AND %s
 UNION ALL
 SELECT od.id::text,'OUTGOING',od.document_number||' · Outgoing completed',od.completed_at
 FROM outgoing_documents od
 WHERE od.tenant_id=$1 AND od.transaction_date BETWEEN $2 AND $3
   AND ($4::uuid='00000000-0000-0000-0000-000000000000'::uuid OR EXISTS(
    SELECT 1 FROM outgoing_kanban_lots okl
    JOIN kanban_lots kl ON kl.tenant_id=okl.tenant_id AND kl.id=okl.kanban_lot_id
    JOIN purchase_order_lines pol ON pol.tenant_id=kl.tenant_id AND pol.id=kl.purchase_order_line_id
    JOIN purchase_orders p ON p.tenant_id=pol.tenant_id AND p.id=pol.purchase_order_id
    WHERE okl.tenant_id=od.tenant_id AND okl.outgoing_id=od.id AND p.supplier_id=$4))
) activity ORDER BY occurred_at DESC LIMIT 10`, supplierWhere, supplierWhere)
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item Activity
		if err := rows.Scan(&item.ID, &item.Type, &item.Label, &item.OccurredAt); err != nil {
			return err
		}
		snapshot.Activities = append(snapshot.Activities, item)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
}
