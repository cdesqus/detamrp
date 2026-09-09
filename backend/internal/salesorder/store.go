package salesorder

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/bom"
	"order-stock/backend/internal/database"
	"time"
)

type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}
type Line struct {
	ID                uuid.UUID       `json:"id"`
	FinishedGoodID    uuid.UUID       `json:"finishedGoodId"`
	ItemCode          string          `json:"itemCode"`
	Name              string          `json:"name"`
	Unit              string          `json:"unit"`
	Quantity          decimal.Decimal `json:"quantity"`
	DeliveredQuantity decimal.Decimal `json:"deliveredQuantity"`
	RemainingQuantity decimal.Decimal `json:"remainingQuantity"`
	SalesPrice        decimal.Decimal `json:"salesPrice"`
	Currency          string          `json:"currency"`
	PriceVersion      int             `json:"priceVersion"`
	Calculation       json.RawMessage `json:"calculation,omitempty"`
}
type Order struct {
	ID           uuid.UUID  `json:"id"`
	Number       string     `json:"number"`
	CustomerID   uuid.UUID  `json:"customerId"`
	CustomerName string     `json:"customerName"`
	Status       string     `json:"status"`
	OrderDate    time.Time  `json:"orderDate"`
	DeliveryDate *time.Time `json:"deliveryDate,omitempty"`
	Lines        []Line     `json:"lines"`
}
type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db} }
func tenant(a Actor) database.TenantContext {
	return database.TenantContext{TenantID: a.TenantID, UserID: a.UserID}
}
func (s *Store) Create(ctx context.Context, a Actor, input Input) (order Order, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "sales-order-number-"+a.TenantID.String()+"-"+time.Now().Format("200601")); e != nil {
			return e
		}
		var count int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM sales_orders WHERE tenant_id=$1 AND date_trunc('month',created_at)=date_trunc('month',now())`, a.TenantID).Scan(&count); e != nil {
			return e
		}
		number := fmt.Sprintf("SLO-%s-%04d", time.Now().Format("200601"), count+1)
		if e := tx.QueryRow(ctx, `INSERT INTO sales_orders(tenant_id,sales_order_number,customer_id,order_date,delivery_date,customer_po_reference,notes,created_by_user_id,updated_by_user_id) VALUES($1,$2,$3,COALESCE(NULLIF($4,'')::date,current_date),NULLIF($5,'')::date,$6,$7,$8,$8) RETURNING id`, a.TenantID, number, input.CustomerID, input.OrderDate, input.DeliveryDate, input.CustomerPOReference, input.Notes, a.UserID).Scan(&order.ID); e != nil {
			return e
		}
		for pos, line := range input.Lines {
			var code, name, unit, currency string
			var price decimal.Decimal
			var version int
			e := tx.QueryRow(ctx, `SELECT f.item_code,f.name,u.code,f.sales_price,f.currency,f.price_version FROM finished_goods f JOIN units u ON u.tenant_id=f.tenant_id AND u.id=f.base_unit_id WHERE f.tenant_id=$1 AND f.id=$2 AND f.active`, a.TenantID, line.FinishedGoodID).Scan(&code, &name, &unit, &price, &currency, &version)
			if e != nil {
				return fmt.Errorf("Finished Good is inactive or missing")
			}
			if _, e = tx.Exec(ctx, `INSERT INTO sales_order_lines(tenant_id,sales_order_id,finished_good_id,item_code_snapshot,item_name_snapshot,base_unit_snapshot,quantity,sales_price_snapshot,currency_snapshot,price_version,sort_position) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, a.TenantID, order.ID, line.FinishedGoodID, code, name, unit, line.Quantity, price, currency, version, pos); e != nil {
				return e
			}
		}
		return s.load(ctx, tx, a.TenantID, order.ID, &order)
	})
	return
}
func (s *Store) Get(ctx context.Context, a Actor, id uuid.UUID) (order Order, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error { return s.load(ctx, tx, a.TenantID, id, &order) })
	return
}
func (s *Store) List(ctx context.Context, a Actor) (orders []Order, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		rows, e := tx.Query(ctx, `SELECT id FROM sales_orders WHERE tenant_id=$1 ORDER BY created_at DESC`, a.TenantID)
		if e != nil {
			return e
		}
		ids := []uuid.UUID{}
		for rows.Next() {
			var id uuid.UUID
			if e = rows.Scan(&id); e != nil {
				return e
			}
			ids = append(ids, id)
		}
		if e = rows.Err(); e != nil {
			rows.Close()
			return e
		}
		rows.Close()
		for _, id := range ids {
			var order Order
			if e = s.load(ctx, tx, a.TenantID, id, &order); e != nil {
				return e
			}
			orders = append(orders, order)
		}
		return nil
	})
	return
}
func (s *Store) CreateDelivery(ctx context.Context, a Actor, orderID uuid.UUID, input DeliveryInput) (delivery Delivery, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		var status string
		if e := tx.QueryRow(ctx, `SELECT status FROM sales_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, orderID).Scan(&status); e != nil {
			return e
		}
		if status != StatusSubmitted {
			return fmt.Errorf("only submitted sales orders can be delivered")
		}
		for _, line := range input.Lines {
			var ordered, delivered decimal.Decimal
			e := tx.QueryRow(ctx, `SELECT l.quantity,COALESCE(sum(d.quantity),0) FROM sales_order_lines l LEFT JOIN customer_delivery_lines d ON d.tenant_id=l.tenant_id AND d.sales_order_line_id=l.id WHERE l.tenant_id=$1 AND l.sales_order_id=$2 AND l.id=$3 GROUP BY l.quantity`, a.TenantID, orderID, line.SalesOrderLineID).Scan(&ordered, &delivered)
			if e != nil {
				return fmt.Errorf("sales order line is invalid")
			}
			if line.Quantity.GreaterThan(ordered.Sub(delivered)) {
				return fmt.Errorf("delivery quantity exceeds remaining order quantity")
			}
		}
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, `customer-delivery-`+a.TenantID.String()+`-`+time.Now().Format("200601")); e != nil {
			return e
		}
		var count int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM customer_deliveries WHERE tenant_id=$1 AND date_trunc('month',created_at)=date_trunc('month',now())`, a.TenantID).Scan(&count); e != nil {
			return e
		}
		delivery.Number = fmt.Sprintf("CD-%s-%04d", time.Now().Format("200601"), count+1)
		delivery.SalesOrderID = orderID
		if e := tx.QueryRow(ctx, `INSERT INTO customer_deliveries(tenant_id,delivery_number,sales_order_id,delivery_date,notes,created_by_user_id) VALUES($1,$2,$3,COALESCE(NULLIF($4,'')::date,current_date),$5,$6) RETURNING id,delivery_date::text`, a.TenantID, delivery.Number, orderID, input.DeliveryDate, input.Notes, a.UserID).Scan(&delivery.ID, &delivery.DeliveryDate); e != nil {
			return e
		}
		for _, line := range input.Lines {
			if _, e := tx.Exec(ctx, `INSERT INTO customer_delivery_lines(tenant_id,customer_delivery_id,sales_order_line_id,quantity) VALUES($1,$2,$3,$4)`, a.TenantID, delivery.ID, line.SalesOrderLineID, line.Quantity); e != nil {
				return e
			}
		}
		delivery.Lines = input.Lines
		return nil
	})
	return
}
func (s *Store) Submit(ctx context.Context, a Actor, id uuid.UUID) (order Order, err error) {
	err = database.WithTenant(ctx, s.db, tenant(a), func(tx database.TenantTx) error {
		var status string
		if e := tx.QueryRow(ctx, `SELECT status FROM sales_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, a.TenantID, id).Scan(&status); e != nil {
			return e
		}
		if status != StatusDraft {
			return fmt.Errorf("only draft sales orders can be submitted")
		}
		rows, e := tx.Query(ctx, `SELECT id,finished_good_id,quantity FROM sales_order_lines WHERE tenant_id=$1 AND sales_order_id=$2 ORDER BY sort_position`, a.TenantID, id)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var lineID, fgID uuid.UUID
			var qty decimal.Decimal
			if e = rows.Scan(&lineID, &fgID, &qty); e != nil {
				return e
			}
			root, e := loadFinishedGoodNode(ctx, tx, a.TenantID, fgID)
			if e != nil {
				return e
			}
			snapshot, e := bom.BuildSnapshot(root)
			if e != nil {
				return e
			}
			payload, e := json.Marshal(snapshot)
			if e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE sales_order_lines SET calculation_snapshot=$3 WHERE tenant_id=$1 AND id=$2`, a.TenantID, lineID, payload); e != nil {
				return e
			}
		}
		if e = rows.Err(); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `UPDATE sales_orders SET status='SUBMITTED',updated_by_user_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, a.TenantID, id, a.UserID); e != nil {
			return e
		}
		return s.load(ctx, tx, a.TenantID, id, &order)
	})
	return
}
func (s *Store) load(ctx context.Context, tx database.TenantTx, tenantID, id uuid.UUID, out *Order) error {
	if e := tx.QueryRow(ctx, `SELECT s.id,s.sales_order_number,s.customer_id,c.name,s.status,s.order_date,s.delivery_date FROM sales_orders s JOIN customers c ON c.tenant_id=s.tenant_id AND c.id=s.customer_id WHERE s.tenant_id=$1 AND s.id=$2`, tenantID, id).Scan(&out.ID, &out.Number, &out.CustomerID, &out.CustomerName, &out.Status, &out.OrderDate, &out.DeliveryDate); e != nil {
		return e
	}
	rows, e := tx.Query(ctx, `SELECT l.id,l.finished_good_id,l.item_code_snapshot,l.item_name_snapshot,l.base_unit_snapshot,l.quantity,COALESCE((SELECT sum(d.quantity) FROM customer_delivery_lines d WHERE d.tenant_id=l.tenant_id AND d.sales_order_line_id=l.id),0),l.quantity-COALESCE((SELECT sum(d.quantity) FROM customer_delivery_lines d WHERE d.tenant_id=l.tenant_id AND d.sales_order_line_id=l.id),0),l.sales_price_snapshot,l.currency_snapshot,l.price_version,COALESCE(l.calculation_snapshot,'null') FROM sales_order_lines l WHERE l.tenant_id=$1 AND l.sales_order_id=$2 ORDER BY l.sort_position`, tenantID, id)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var line Line
		if e = rows.Scan(&line.ID, &line.FinishedGoodID, &line.ItemCode, &line.Name, &line.Unit, &line.Quantity, &line.DeliveredQuantity, &line.RemainingQuantity, &line.SalesPrice, &line.Currency, &line.PriceVersion, &line.Calculation); e != nil {
			return e
		}
		out.Lines = append(out.Lines, line)
	}
	return rows.Err()
}
func loadFinishedGoodNode(ctx context.Context, tx database.TenantTx, tenantID, fgID uuid.UUID) (bom.Node, error) {
	var node bom.Node
	node.Kind = "FG"
	node.Usage = decimal.NewFromInt(1)
	e := tx.QueryRow(ctx, `SELECT f.id,f.item_code,f.name,u.code FROM finished_goods f JOIN units u ON u.tenant_id=f.tenant_id AND u.id=f.base_unit_id WHERE f.tenant_id=$1 AND f.id=$2 AND f.active`, tenantID, fgID).Scan(&node.ItemID, &node.ItemCode, &node.Name, &node.Unit)
	if e != nil {
		return node, fmt.Errorf("Finished Good has no active BOM")
	}
	children, e := loadActiveComponents(ctx, tx, tenantID, "FG", fgID, map[uuid.UUID]bool{})
	if e != nil {
		return node, e
	}
	if len(children) == 0 {
		return node, fmt.Errorf("Finished Good %s has no active BOM", node.ItemCode)
	}
	node.Children = children
	return node, nil
}
func loadActiveComponents(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID, kind string, outputID uuid.UUID, stack map[uuid.UUID]bool) ([]bom.Node, error) {
	if stack[outputID] {
		return nil, fmt.Errorf("circular BOM")
	}
	stack[outputID] = true
	defer delete(stack, outputID)
	var bomID uuid.UUID
	var e error
	if kind == "FG" {
		e = tx.QueryRow(ctx, `SELECT id FROM boms WHERE tenant_id=$1 AND output_kind='FG' AND finished_good_id=$2 AND status='ACTIVE'`, tenantID, outputID).Scan(&bomID)
	} else {
		e = tx.QueryRow(ctx, `SELECT id FROM boms WHERE tenant_id=$1 AND output_kind='RAW_MATERIAL' AND raw_material_id=$2 AND status='ACTIVE'`, tenantID, outputID).Scan(&bomID)
	}
	if e == pgx.ErrNoRows {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	rows, e := tx.Query(ctx, `SELECT r.id,r.code,r.name,u.code,r.standard_unit_price,r.currency,r.qty_per_kanban,c.usage_qty FROM bom_components c JOIN raw_materials r ON r.tenant_id=c.tenant_id AND r.id=c.raw_material_id JOIN units u ON u.tenant_id=r.tenant_id AND u.id=r.base_unit_id WHERE c.tenant_id=$1 AND c.bom_id=$2 ORDER BY c.sort_position`, tenantID, bomID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var result []bom.Node
	for rows.Next() {
		var node bom.Node
		node.Kind = "RAW_MATERIAL"
		if e = rows.Scan(&node.ItemID, &node.ItemCode, &node.Name, &node.Unit, &node.UnitPrice, &node.Currency, &node.QtyPerKanban, &node.Usage); e != nil {
			return nil, e
		}
		children, e := loadActiveComponents(ctx, tx, tenantID, "RAW_MATERIAL", uuid.MustParse(node.ItemID), stack)
		if e != nil {
			return nil, e
		}
		node.Children = children
		result = append(result, node)
	}
	return result, rows.Err()
}
