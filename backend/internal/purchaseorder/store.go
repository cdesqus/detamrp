package purchaseorder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"order-stock/backend/internal/database"
)

type SQLStore struct{ db *database.Pool }

var _ Repository = (*SQLStore)(nil)

func NewSQLStore(db *database.Pool) *SQLStore { return &SQLStore{db: db} }

const orderSelect = `SELECT p.id,p.tenant_id,p.po_number,p.supplier_id,p.order_date,p.expected_delivery_date,
 p.currency,p.notes,p.status,p.version,p.total_amount,COALESCE(p.sage_purchase_order_number,''),
 COALESCE(p.submitted_approver_user_id,'00000000-0000-0000-0000-000000000000'),p.submitted_approver_display_name,p.submitted_approver_email,
 p.created_by_user_id,cu.display_name,cu.email,p.created_at,p.updated_by_user_id,uu.display_name,uu.email,p.updated_at
 FROM purchase_orders p
 JOIN users cu ON cu.tenant_id=p.tenant_id AND cu.id=p.created_by_user_id
 JOIN users uu ON uu.tenant_id=p.tenant_id AND uu.id=p.updated_by_user_id`

const orderLineSelect = `SELECT l.id,l.tenant_id,l.purchase_order_id,l.raw_material_id,l.raw_material_code_snapshot,
 l.raw_material_name_snapshot,l.base_unit_id,l.base_unit_code_snapshot,l.qty_per_kanban_snapshot,l.total_kanban,
 l.ordered_base_qty,l.unit_price_snapshot,l.line_total,l.sort_position,l.created_by_user_id,cu.display_name,cu.email,
 l.created_at,l.updated_by_user_id,uu.display_name,uu.email,l.updated_at
 FROM purchase_order_lines l
 JOIN users cu ON cu.tenant_id=l.tenant_id AND cu.id=l.created_by_user_id
 JOIN users uu ON uu.tenant_id=l.tenant_id AND uu.id=l.updated_by_user_id`

const approvalSelect = `SELECT a.id,a.tenant_id,a.purchase_order_id,a.version,a.approver_user_id,a.approver_display_name,
 a.approver_email,a.status,a.decision_reason,a.decided_at,COALESCE(a.decided_by_user_id,'00000000-0000-0000-0000-000000000000'),
 a.created_by_user_id,cu.display_name,cu.email,a.created_at,a.updated_by_user_id,uu.display_name,uu.email,a.updated_at
 FROM purchase_order_approvals a
 JOIN users cu ON cu.tenant_id=a.tenant_id AND cu.id=a.created_by_user_id
 JOIN users uu ON uu.tenant_id=a.tenant_id AND uu.id=a.updated_by_user_id`

func scanOrder(row pgx.Row) (Order, error) {
	var order Order
	err := row.Scan(
		&order.ID, &order.TenantID, &order.PONumber, &order.SupplierID, &order.OrderDate, &order.ExpectedDeliveryDate,
		&order.Currency, &order.Notes, &order.Status, &order.Version, &order.TotalAmount, &order.SagePurchaseOrderNumber,
		&order.SubmittedApproverUserID, &order.SubmittedApproverDisplayName, &order.SubmittedApproverEmail,
		&order.CreatedBy.UserID, &order.CreatedBy.DisplayName, &order.CreatedBy.Email, &order.CreatedAt,
		&order.UpdatedBy.UserID, &order.UpdatedBy.DisplayName, &order.UpdatedBy.Email, &order.UpdatedAt,
	)
	order.CreatedBy.TenantID = order.TenantID
	order.UpdatedBy.TenantID = order.TenantID
	return order, err
}

func scanOrderLine(row pgx.Row) (OrderLine, error) {
	var line OrderLine
	err := row.Scan(
		&line.ID, &line.TenantID, &line.PurchaseOrderID, &line.RawMaterialID, &line.RawMaterialCode, &line.RawMaterialName,
		&line.BaseUnitID, &line.BaseUnitCode, &line.QtyPerKanbanSnapshot, &line.TotalKanban, &line.OrderedBaseQty,
		&line.UnitPriceSnapshot, &line.LineTotal, &line.SortPosition,
		&line.CreatedBy.UserID, &line.CreatedBy.DisplayName, &line.CreatedBy.Email, &line.CreatedAt,
		&line.UpdatedBy.UserID, &line.UpdatedBy.DisplayName, &line.UpdatedBy.Email, &line.UpdatedAt,
	)
	line.CreatedBy.TenantID = line.TenantID
	line.UpdatedBy.TenantID = line.TenantID
	return line, err
}

func scanApproval(row pgx.Row) (Approval, error) {
	var approval Approval
	err := row.Scan(
		&approval.ID, &approval.TenantID, &approval.PurchaseOrderID, &approval.Version, &approval.ApproverUserID,
		&approval.ApproverDisplayName, &approval.ApproverEmail, &approval.Status, &approval.DecisionReason, &approval.DecidedAt,
		&approval.DecidedByUserID, &approval.CreatedBy.UserID, &approval.CreatedBy.DisplayName, &approval.CreatedBy.Email,
		&approval.CreatedAt, &approval.UpdatedBy.UserID, &approval.UpdatedBy.DisplayName, &approval.UpdatedBy.Email, &approval.UpdatedAt,
	)
	approval.CreatedBy.TenantID = approval.TenantID
	approval.UpdatedBy.TenantID = approval.TenantID
	return approval, err
}

func (s *SQLStore) ListOrders(ctx context.Context, actor Actor, query ListQuery) (items []Order, total int, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM purchase_orders p WHERE p.tenant_id=$1
 AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR p.supplier_id=$2)
 AND ($3='' OR p.status=$3)
 AND ($4='' OR p.po_number ILIKE '%'||$4||'%' OR p.notes ILIKE '%'||$4||'%')`, actor.TenantID, query.SupplierID, query.Status, query.Search).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, orderSelect+` WHERE p.tenant_id=$1
 AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR p.supplier_id=$2)
 AND ($3='' OR p.status=$3)
 AND ($4='' OR p.po_number ILIKE '%'||$4||'%' OR p.notes ILIKE '%'||$4||'%')
 ORDER BY p.order_date DESC,p.po_number DESC LIMIT $5 OFFSET $6`, actor.TenantID, query.SupplierID, query.Status, query.Search, query.Limit, query.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			order, err := scanOrder(rows)
			if err != nil {
				return err
			}
			items = append(items, order)
		}
		return rows.Err()
	})
	return
}

func (s *SQLStore) GetOrder(ctx context.Context, actor Actor, id uuid.UUID) (order Order, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		order, err = getOrder(ctx, tx, actor.TenantID, id)
		return err
	})
	return
}

func getOrder(ctx context.Context, tx database.TenantTx, tenantID, id uuid.UUID) (Order, error) {
	order, err := scanOrder(tx.QueryRow(ctx, orderSelect+` WHERE p.tenant_id=$1 AND p.id=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, NotFoundError{Resource: "purchase order"}
	}
	if err != nil {
		return Order{}, err
	}
	rows, err := tx.Query(ctx, orderLineSelect+` WHERE l.tenant_id=$1 AND l.purchase_order_id=$2 ORDER BY l.sort_position`, tenantID, id)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()
	for rows.Next() {
		line, err := scanOrderLine(rows)
		if err != nil {
			return Order{}, err
		}
		order.Lines = append(order.Lines, line)
	}
	return order, rows.Err()
}

func (s *SQLStore) CreateOrder(ctx context.Context, actor Actor, input OrderInput) (order Order, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		currency, err := activeSupplierCurrency(ctx, tx, actor.TenantID, input.SupplierID)
		if err != nil {
			return err
		}
		poNumber, err := nextPONumber(ctx, tx, actor.TenantID, input.OrderDate)
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO purchase_orders(tenant_id,po_number,supplier_id,order_date,expected_delivery_date,currency,notes,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING id`, actor.TenantID, poNumber, input.SupplierID, input.OrderDate, input.ExpectedDeliveryDate, currency, input.Notes, actor.UserID).Scan(&order.ID); err != nil {
			return writeError(err)
		}
		if err := replaceDraftLines(ctx, tx, actor, order.ID, input.SupplierID, input.Lines); err != nil {
			return err
		}
		order, err = getOrder(ctx, tx, actor.TenantID, order.ID)
		return err
	})
	return
}

func (s *SQLStore) UpdateOrder(ctx context.Context, actor Actor, id uuid.UUID, input OrderInput) (order Order, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		if err := lockDraft(ctx, tx, actor.TenantID, id, "edited"); err != nil {
			return err
		}
		currency, err := activeSupplierCurrency(ctx, tx, actor.TenantID, input.SupplierID)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET supplier_id=$3,order_date=$4,expected_delivery_date=$5,currency=$6,notes=$7,
 updated_by_user_id=$8,updated_at=now() WHERE tenant_id=$1 AND id=$2`, actor.TenantID, id, input.SupplierID, input.OrderDate, input.ExpectedDeliveryDate, currency, input.Notes, actor.UserID); err != nil {
			return writeError(err)
		}
		if err = replaceDraftLines(ctx, tx, actor, id, input.SupplierID, input.Lines); err != nil {
			return err
		}
		order, err = getOrder(ctx, tx, actor.TenantID, id)
		return err
	})
	return
}

func (s *SQLStore) SubmitOrder(ctx context.Context, actor Actor, id uuid.UUID) (order Order, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		if err := lockDraft(ctx, tx, actor.TenantID, id, "submitted"); err != nil {
			return err
		}
		order, err = getOrder(ctx, tx, actor.TenantID, id)
		if err != nil {
			return err
		}
		if fields, validationErr := validStoredOrder(ctx, tx, actor.TenantID, order); validationErr != nil {
			return validationErr
		} else if len(fields) > 0 {
			return ValidationError{Fields: fields}
		}
		approverID, approverName, approverEmail, err := configuredApprover(ctx, tx, actor.TenantID)
		if err != nil {
			return err
		}
		version := order.Version + 1
		if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET status=$3,version=$4,submitted_approver_user_id=$5,
 submitted_approver_display_name=$6,submitted_approver_email=$7,updated_by_user_id=$8,updated_at=now()
 WHERE tenant_id=$1 AND id=$2`, actor.TenantID, id, StatusPendingApproval, version, approverID, approverName, approverEmail, actor.UserID); err != nil {
			return writeError(err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO purchase_order_approvals(tenant_id,purchase_order_id,version,approver_user_id,approver_display_name,approver_email,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, actor.TenantID, id, version, approverID, approverName, approverEmail, actor.UserID); err != nil {
			return writeError(err)
		}
		order, err = getOrder(ctx, tx, actor.TenantID, id)
		return err
	})
	return
}

func (s *SQLStore) CancelOrder(ctx context.Context, actor Actor, id uuid.UUID) (order Order, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		if err := lockDraft(ctx, tx, actor.TenantID, id, "cancelled"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE purchase_orders SET status=$3,updated_by_user_id=$4,updated_at=now()
 WHERE tenant_id=$1 AND id=$2`, actor.TenantID, id, StatusCancelled, actor.UserID); err != nil {
			return err
		}
		order, err = getOrder(ctx, tx, actor.TenantID, id)
		return err
	})
	return
}

func (s *SQLStore) ListApprovals(ctx context.Context, actor Actor, query ListQuery) (items []Approval, total int, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM purchase_order_approvals a JOIN purchase_orders p ON p.tenant_id=a.tenant_id AND p.id=a.purchase_order_id
 WHERE a.tenant_id=$1 AND a.approver_user_id=$2 AND a.status='PENDING'
 AND ($3='' OR p.po_number ILIKE '%'||$3||'%' OR p.notes ILIKE '%'||$3||'%')`, actor.TenantID, actor.UserID, query.Search).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, approvalSelect+` JOIN purchase_orders p ON p.tenant_id=a.tenant_id AND p.id=a.purchase_order_id
 WHERE a.tenant_id=$1 AND a.approver_user_id=$2 AND a.status='PENDING'
 AND ($3='' OR p.po_number ILIKE '%'||$3||'%' OR p.notes ILIKE '%'||$3||'%')
 ORDER BY a.created_at DESC LIMIT $4 OFFSET $5`, actor.TenantID, actor.UserID, query.Search, query.Limit, query.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			approval, err := scanApproval(rows)
			if err != nil {
				return err
			}
			items = append(items, approval)
		}
		return rows.Err()
	})
	return
}

func (s *SQLStore) Approve(ctx context.Context, actor Actor, id uuid.UUID, input DecisionInput) (Approval, error) {
	return s.decide(ctx, actor, id, input, ApprovalApproved, StatusApproved)
}

func (s *SQLStore) Reject(ctx context.Context, actor Actor, id uuid.UUID, input DecisionInput) (Approval, error) {
	return s.decide(ctx, actor, id, input, ApprovalRejected, StatusRejected)
}

func (s *SQLStore) decide(ctx context.Context, actor Actor, id uuid.UUID, input DecisionInput, approvalStatus ApprovalStatus, orderStatus Status) (approval Approval, err error) {
	if fields := input.NormalizeAndValidate(approvalStatus == ApprovalRejected); len(fields) > 0 {
		return Approval{}, ValidationError{Fields: fields}
	}
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		var purchaseOrderID, approverID uuid.UUID
		var status ApprovalStatus
		var version int
		err := tx.QueryRow(ctx, `SELECT purchase_order_id,approver_user_id,status,version FROM purchase_order_approvals
 WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, actor.TenantID, id).Scan(&purchaseOrderID, &approverID, &status, &version)
		if errors.Is(err, pgx.ErrNoRows) {
			return NotFoundError{Resource: "purchase order approval"}
		}
		if err != nil {
			return err
		}
		if approverID != actor.UserID {
			return ConflictError{Fields: FieldErrors{"approval": "This approval is assigned to another user"}}
		}
		if status != ApprovalPending {
			return ConflictError{Fields: FieldErrors{"status": "This approval has already been decided"}}
		}
		var currentStatus Status
		var currentVersion int
		if err = tx.QueryRow(ctx, `SELECT status,version FROM purchase_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, actor.TenantID, purchaseOrderID).Scan(&currentStatus, &currentVersion); err != nil {
			return err
		}
		if currentStatus != StatusPendingApproval || currentVersion != version {
			return ConflictError{Fields: FieldErrors{"status": "Purchase order is no longer awaiting this approval"}}
		}
		if _, err = tx.Exec(ctx, `UPDATE purchase_order_approvals SET status=$3,decision_reason=$4,decided_at=now(),decided_by_user_id=$5,
 updated_by_user_id=$5,updated_at=now() WHERE tenant_id=$1 AND id=$2`, actor.TenantID, id, approvalStatus, input.Reason, actor.UserID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET status=$3,updated_by_user_id=$4,updated_at=now() WHERE tenant_id=$1 AND id=$2`, actor.TenantID, purchaseOrderID, orderStatus, actor.UserID); err != nil {
			return err
		}
		approval, err = getApproval(ctx, tx, actor.TenantID, id)
		return err
	})
	return
}

func tenantContext(actor Actor) database.TenantContext {
	return database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}
}

func lockDraft(ctx context.Context, tx database.TenantTx, tenantID, id uuid.UUID, action string) error {
	var status Status
	err := tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenantID, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return NotFoundError{Resource: "purchase order"}
	}
	if err != nil {
		return err
	}
	if status != StatusDraft {
		return draftConflict(action)
	}
	return nil
}

func activeSupplierCurrency(ctx context.Context, tx database.TenantTx, tenantID, supplierID uuid.UUID) (string, error) {
	var currency string
	err := tx.QueryRow(ctx, `SELECT currency FROM suppliers WHERE tenant_id=$1 AND id=$2 AND active FOR SHARE`, tenantID, supplierID).Scan(&currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ConflictError{Fields: FieldErrors{"supplierId": "Select an active supplier"}}
	}
	return currency, err
}

func nextPONumber(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID, orderDate time.Time) (string, error) {
	yearMonth := orderDate.Format("200601")
	var sequence int
	err := tx.QueryRow(ctx, `INSERT INTO purchase_order_number_sequences(tenant_id,year_month,next_value) VALUES($1,$2,2)
 ON CONFLICT(tenant_id,year_month) DO UPDATE SET next_value=purchase_order_number_sequences.next_value+1
 RETURNING next_value-1`, tenantID, yearMonth).Scan(&sequence)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("PO-%s-%05d", yearMonth, sequence), nil
}

func replaceDraftLines(ctx context.Context, tx database.TenantTx, actor Actor, orderID, supplierID uuid.UUID, inputs []LineInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM purchase_order_lines WHERE tenant_id=$1 AND purchase_order_id=$2`, actor.TenantID, orderID); err != nil {
		return err
	}
	order := Order{}
	for index, input := range inputs {
		line, err := snapshotLine(ctx, tx, actor.TenantID, orderID, supplierID, input, index+1)
		if err != nil {
			return err
		}
		order.Lines = append(order.Lines, line)
	}
	order.RecalculateTotals()
	for _, line := range order.Lines {
		if _, err := tx.Exec(ctx, `INSERT INTO purchase_order_lines(tenant_id,purchase_order_id,raw_material_id,raw_material_code_snapshot,
 raw_material_name_snapshot,base_unit_id,base_unit_code_snapshot,qty_per_kanban_snapshot,total_kanban,ordered_base_qty,
 unit_price_snapshot,line_total,sort_position,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14)`, actor.TenantID, orderID, line.RawMaterialID,
			line.RawMaterialCode, line.RawMaterialName, line.BaseUnitID, line.BaseUnitCode, line.QtyPerKanbanSnapshot,
			line.TotalKanban, line.OrderedBaseQty, line.UnitPriceSnapshot, line.LineTotal, line.SortPosition, actor.UserID); err != nil {
			return writeError(err)
		}
	}
	_, err := tx.Exec(ctx, `UPDATE purchase_orders SET total_amount=$3,updated_by_user_id=$4,updated_at=now()
 WHERE tenant_id=$1 AND id=$2`, actor.TenantID, orderID, order.TotalAmount, actor.UserID)
	return err
}

func snapshotLine(ctx context.Context, tx database.TenantTx, tenantID, orderID, supplierID uuid.UUID, input LineInput, position int) (OrderLine, error) {
	line := OrderLine{TenantID: tenantID, PurchaseOrderID: orderID, RawMaterialID: input.RawMaterialID, TotalKanban: input.TotalKanban, SortPosition: position}
	err := tx.QueryRow(ctx, `SELECT r.code,r.name,r.base_unit_id,m.code,r.qty_per_kanban,r.standard_unit_price
 FROM raw_materials r JOIN measurements m ON m.tenant_id=r.tenant_id AND m.id=r.base_unit_id AND m.active
	 WHERE r.tenant_id=$1 AND r.id=$2 AND r.supplier_id=$3 AND r.active FOR SHARE OF r,m`, tenantID, input.RawMaterialID, supplierID).
		Scan(&line.RawMaterialCode, &line.RawMaterialName, &line.BaseUnitID, &line.BaseUnitCode, &line.QtyPerKanbanSnapshot, &line.UnitPriceSnapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrderLine{}, ConflictError{Fields: FieldErrors{fmt.Sprintf("lines[%d].rawMaterialId", position-1): "Select an active Raw Material for this supplier"}}
	}
	if err != nil {
		return OrderLine{}, err
	}
	line.Recalculate()
	return line, nil
}

func validStoredOrder(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID, order Order) (FieldErrors, error) {
	input := OrderInput{SupplierID: order.SupplierID, OrderDate: order.OrderDate, ExpectedDeliveryDate: order.ExpectedDeliveryDate, Currency: order.Currency}
	for _, line := range order.Lines {
		input.Lines = append(input.Lines, LineInput{RawMaterialID: line.RawMaterialID, TotalKanban: line.TotalKanban})
	}
	if fields := input.NormalizeAndValidate(true); len(fields) > 0 {
		return fields, nil
	}
	if _, err := activeSupplierCurrency(ctx, tx, tenantID, order.SupplierID); err != nil {
		var conflict ConflictError
		if errors.As(err, &conflict) {
			return FieldErrors{"supplierId": "Select an active supplier before submission"}, nil
		}
		return nil, err
	}
	recalculated := Order{Lines: append([]OrderLine(nil), order.Lines...)}
	recalculated.RecalculateTotals()
	if !recalculated.TotalAmount.Equal(order.TotalAmount) {
		return FieldErrors{"totalAmount": "Purchase order totals must be recalculated before submission"}, nil
	}
	for index, line := range order.Lines {
		var active bool
		err := tx.QueryRow(ctx, `SELECT r.active AND r.supplier_id=$3 FROM raw_materials r WHERE r.tenant_id=$1 AND r.id=$2 FOR SHARE`, tenantID, line.RawMaterialID, order.SupplierID).Scan(&active)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return FieldErrors{fmt.Sprintf("lines[%d].rawMaterialId", index): "Raw Material is no longer active for this supplier"}, nil
			}
			return nil, err
		}
		if !active {
			return FieldErrors{fmt.Sprintf("lines[%d].rawMaterialId", index): "Raw Material is no longer active for this supplier"}, nil
		}
		if !line.QtyPerKanbanSnapshot.IsPositive() || line.UnitPriceSnapshot.IsNegative() || !line.OrderedBaseQty.IsPositive() || line.LineTotal.IsNegative() {
			return FieldErrors{fmt.Sprintf("lines[%d]", index): "Raw Material snapshots are invalid"}, nil
		}
		if !recalculated.Lines[index].OrderedBaseQty.Equal(line.OrderedBaseQty) || !recalculated.Lines[index].LineTotal.Equal(line.LineTotal) {
			return FieldErrors{fmt.Sprintf("lines[%d]", index): "Raw Material totals must be recalculated before submission"}, nil
		}
	}
	return nil, nil
}

func configuredApprover(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID) (uuid.UUID, string, string, error) {
	var id uuid.UUID
	var displayName, email string
	err := tx.QueryRow(ctx, `SELECT u.id,u.display_name,u.email FROM tenant_settings ts
 JOIN users u ON u.tenant_id=ts.tenant_id AND u.id=ts.default_approver_user_id AND NOT u.locked
 WHERE ts.tenant_id=$1 AND EXISTS(
  SELECT 1 FROM user_roles ur JOIN roles r ON r.tenant_id=ur.tenant_id AND r.id=ur.role_id AND r.active
  JOIN role_permissions rp ON rp.tenant_id=r.tenant_id AND rp.role_id=r.id AND rp.permission_code='po.approve'
  WHERE ur.tenant_id=ts.tenant_id AND ur.user_id=u.id)`, tenantID).Scan(&id, &displayName, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", "", ConflictError{Fields: FieldErrors{"approver": "Configure an active PO Approver before submission"}}
	}
	return id, displayName, email, err
}

func getApproval(ctx context.Context, tx database.TenantTx, tenantID, id uuid.UUID) (Approval, error) {
	approval, err := scanApproval(tx.QueryRow(ctx, approvalSelect+` WHERE a.tenant_id=$1 AND a.id=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Approval{}, NotFoundError{Resource: "purchase order approval"}
	}
	return approval, err
}

func writeError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		switch databaseError.Code {
		case "23505":
			return ConflictError{Fields: FieldErrors{"form": "Purchase order conflicts with existing data"}}
		case "23503", "23514":
			return ConflictError{Fields: FieldErrors{"form": "Purchase order references invalid data"}}
		}
	}
	return err
}
