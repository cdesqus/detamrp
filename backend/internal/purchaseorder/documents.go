package purchaseorder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
)

type approvedDocumentLine struct {
	ID                   uuid.UUID
	DeliveryNoteLineID   uuid.UUID
	TotalKanban          decimal.Decimal
	QtyPerKanbanSnapshot decimal.Decimal
}

const (
	maxDeliveryNoteSequence int64 = 99_999
	maxKanbanSequence       int64 = 999_999
)

const insertMissingKanbanLotsSQL = `WITH missing AS (
 SELECT dnl.id AS delivery_note_line_id,pol.id AS purchase_order_line_id,
  generated.lot_number::integer AS lot_number,pol.qty_per_kanban_snapshot AS quantity,
  row_number() OVER (ORDER BY pol.sort_position,pol.id,generated.lot_number)-1 AS sequence_offset
 FROM purchase_order_lines pol
 JOIN delivery_note_lines dnl ON dnl.tenant_id=pol.tenant_id
  AND dnl.purchase_order_id=pol.purchase_order_id AND dnl.purchase_order_line_id=pol.id
 CROSS JOIN LATERAL generate_series(1,pol.total_kanban::bigint) AS generated(lot_number)
 LEFT JOIN kanban_lots existing ON existing.tenant_id=pol.tenant_id
  AND existing.delivery_note_line_id=dnl.id AND existing.purchase_order_line_id=pol.id
  AND existing.lot_number=generated.lot_number
 WHERE pol.tenant_id=$1 AND pol.purchase_order_id=$2
  AND generated.lot_number <= pol.total_kanban AND existing.id IS NULL
)
INSERT INTO kanban_lots(tenant_id,delivery_note_line_id,purchase_order_line_id,kanban_id,lot_number,quantity,created_by_user_id,updated_by_user_id)
SELECT $1,delivery_note_line_id,purchase_order_line_id,
 'KB-'||$3||'-'||lpad(($4+sequence_offset)::text,6,'0'),lot_number,quantity,$5,$5
FROM missing ORDER BY sequence_offset`

func ensureApprovedDocuments(ctx context.Context, tx database.TenantTx, actor Actor, purchaseOrderID uuid.UUID) error {
	var orderStatus Status
	if err := tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE tenant_id=$1 AND id=$2`, actor.TenantID, purchaseOrderID).Scan(&orderStatus); err != nil {
		return err
	}
	if orderStatus != StatusApproved {
		return fmt.Errorf("purchase order %s is not approved", purchaseOrderID)
	}

	lines, err := loadApprovedDocumentLines(ctx, tx, actor.TenantID, purchaseOrderID)
	if err != nil {
		return err
	}
	if err := validateApprovedDocumentLines(purchaseOrderID, lines); err != nil {
		return err
	}

	deliveryNoteID, issuedAt, err := ensureDeliveryNote(ctx, tx, actor, purchaseOrderID)
	if err != nil {
		return err
	}
	for index := range lines {
		lines[index].DeliveryNoteLineID, err = ensureDeliveryNoteLine(ctx, tx, actor, deliveryNoteID, purchaseOrderID, lines[index].ID)
		if err != nil {
			return err
		}
	}

	if err := validateExistingKanbanLots(ctx, tx, actor.TenantID, purchaseOrderID, issuedAt); err != nil {
		return err
	}
	missingLots, err := countMissingKanbanLots(ctx, tx, actor.TenantID, purchaseOrderID)
	if err != nil {
		return err
	}
	if missingLots == 0 {
		return nil
	}
	yearMonth := issuedAt.UTC().Format("200601")
	firstNumber, err := reserveNumberBlock(ctx, tx, "kanban_number_sequences", actor.TenantID, yearMonth, missingLots)
	if err != nil {
		return err
	}
	inserted, err := tx.Exec(ctx, insertMissingKanbanLotsSQL, actor.TenantID, purchaseOrderID, yearMonth, firstNumber, actor.UserID)
	if err != nil {
		return err
	}
	if inserted.RowsAffected() != missingLots {
		return fmt.Errorf("purchase order %s generated %d of %d missing Kanban lots", purchaseOrderID, inserted.RowsAffected(), missingLots)
	}
	return nil
}

func validateApprovalDocumentCapacity(ctx context.Context, tx database.TenantTx, actor Actor, purchaseOrderID uuid.UUID) error {
	lines, err := loadApprovedDocumentLines(ctx, tx, actor.TenantID, purchaseOrderID)
	if err != nil {
		return err
	}
	if err := validateApprovedDocumentLines(purchaseOrderID, lines); err != nil {
		return ValidationError{Fields: FieldErrors{"lines": "A purchase order cannot exceed 999999 Kanban labels"}}
	}
	var issuedAt time.Time
	if err := tx.QueryRow(ctx, `SELECT now()`).Scan(&issuedAt); err != nil {
		return err
	}
	yearMonth := issuedAt.UTC().Format("200601")
	if err := checkNumberBlockCapacity(ctx, tx, "delivery_note_number_sequences", actor.TenantID, yearMonth, 1); err != nil {
		return err
	}
	var totalKanbans int64
	for _, line := range lines {
		totalKanbans += line.TotalKanban.IntPart()
	}
	return checkNumberBlockCapacity(ctx, tx, "kanban_number_sequences", actor.TenantID, yearMonth, totalKanbans)
}

func validateApprovedDocumentLines(purchaseOrderID uuid.UUID, lines []approvedDocumentLine) error {
	if len(lines) == 0 {
		return fmt.Errorf("purchase order %s has no lines", purchaseOrderID)
	}
	var totalKanbans int64
	for _, line := range lines {
		if !line.TotalKanban.Equal(line.TotalKanban.Truncate(0)) || !line.TotalKanban.IsPositive() {
			return fmt.Errorf("PO line %s has invalid Kanban count", line.ID)
		}
		lineKanbans := line.TotalKanban.IntPart()
		if lineKanbans > maxKanbanSequence-totalKanbans {
			return fmt.Errorf("purchase order %s exceeds monthly identifier capacity of %d Kanbans", purchaseOrderID, maxKanbanSequence)
		}
		totalKanbans += lineKanbans
	}
	return nil
}

func loadApprovedDocumentLines(ctx context.Context, tx database.TenantTx, tenantID, purchaseOrderID uuid.UUID) ([]approvedDocumentLine, error) {
	rows, err := tx.Query(ctx, `SELECT id,total_kanban,qty_per_kanban_snapshot
 FROM purchase_order_lines WHERE tenant_id=$1 AND purchase_order_id=$2 ORDER BY sort_position`, tenantID, purchaseOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lines []approvedDocumentLine
	for rows.Next() {
		var line approvedDocumentLine
		if err := rows.Scan(&line.ID, &line.TotalKanban, &line.QtyPerKanbanSnapshot); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, rows.Err()
}

func ensureDeliveryNote(ctx context.Context, tx database.TenantTx, actor Actor, purchaseOrderID uuid.UUID) (uuid.UUID, time.Time, error) {
	deliveryNoteID, issuedAt, found, err := loadDeliveryNote(ctx, tx, actor.TenantID, purchaseOrderID)
	if err != nil || found {
		return deliveryNoteID, issuedAt, err
	}
	if err := tx.QueryRow(ctx, `SELECT now()`).Scan(&issuedAt); err != nil {
		return uuid.Nil, time.Time{}, err
	}
	yearMonth := issuedAt.UTC().Format("200601")
	sequence, err := reserveNumberBlock(ctx, tx, "delivery_note_number_sequences", actor.TenantID, yearMonth, 1)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	deliveryNoteNumber := fmt.Sprintf("DN-%s-%05d", yearMonth, sequence)
	err = tx.QueryRow(ctx, `INSERT INTO delivery_notes(tenant_id,purchase_order_id,delivery_note_number,status,issued_at,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,'ISSUED',$4,$5,$5)
 ON CONFLICT(tenant_id,purchase_order_id) DO NOTHING
 RETURNING id`, actor.TenantID, purchaseOrderID, deliveryNoteNumber, issuedAt, actor.UserID).Scan(&deliveryNoteID)
	if errors.Is(err, pgx.ErrNoRows) {
		deliveryNoteID, issuedAt, found, err = loadDeliveryNote(ctx, tx, actor.TenantID, purchaseOrderID)
		if err != nil {
			return uuid.Nil, time.Time{}, err
		}
		if !found {
			return uuid.Nil, time.Time{}, fmt.Errorf("delivery note for purchase order %s was not created", purchaseOrderID)
		}
		return deliveryNoteID, issuedAt, nil
	}
	return deliveryNoteID, issuedAt, err
}

func loadDeliveryNote(ctx context.Context, tx database.TenantTx, tenantID, purchaseOrderID uuid.UUID) (uuid.UUID, time.Time, bool, error) {
	var deliveryNoteID, storedPurchaseOrderID uuid.UUID
	var deliveryNoteNumber string
	var issuedAt time.Time
	err := tx.QueryRow(ctx, `SELECT id,purchase_order_id,delivery_note_number,issued_at
 FROM delivery_notes WHERE tenant_id=$1 AND purchase_order_id=$2`, tenantID, purchaseOrderID).
		Scan(&deliveryNoteID, &storedPurchaseOrderID, &deliveryNoteNumber, &issuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, time.Time{}, false, nil
	}
	if err != nil {
		return uuid.Nil, time.Time{}, false, err
	}
	if storedPurchaseOrderID != purchaseOrderID || deliveryNoteID == uuid.Nil ||
		!strings.HasPrefix(deliveryNoteNumber, "DN-"+issuedAt.UTC().Format("200601")+"-") {
		return uuid.Nil, time.Time{}, false, fmt.Errorf("existing delivery note for purchase order %s is inconsistent", purchaseOrderID)
	}
	return deliveryNoteID, issuedAt, true, nil
}

func ensureDeliveryNoteLine(ctx context.Context, tx database.TenantTx, actor Actor, deliveryNoteID, purchaseOrderID, purchaseOrderLineID uuid.UUID) (uuid.UUID, error) {
	var deliveryNoteLineID uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO delivery_note_lines(tenant_id,delivery_note_id,purchase_order_id,purchase_order_line_id,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$5)
 ON CONFLICT(tenant_id,delivery_note_id,purchase_order_line_id) DO NOTHING
 RETURNING id`, actor.TenantID, deliveryNoteID, purchaseOrderID, purchaseOrderLineID, actor.UserID).Scan(&deliveryNoteLineID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	if err == nil {
		return deliveryNoteLineID, nil
	}

	var storedDeliveryNoteID, storedPurchaseOrderID, storedPurchaseOrderLineID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id,delivery_note_id,purchase_order_id,purchase_order_line_id
 FROM delivery_note_lines WHERE tenant_id=$1 AND delivery_note_id=$2 AND purchase_order_line_id=$3`, actor.TenantID, deliveryNoteID, purchaseOrderLineID).
		Scan(&deliveryNoteLineID, &storedDeliveryNoteID, &storedPurchaseOrderID, &storedPurchaseOrderLineID)
	if err != nil {
		return uuid.Nil, err
	}
	if storedDeliveryNoteID != deliveryNoteID || storedPurchaseOrderID != purchaseOrderID || storedPurchaseOrderLineID != purchaseOrderLineID {
		return uuid.Nil, fmt.Errorf("existing delivery note line for PO line %s is inconsistent", purchaseOrderLineID)
	}
	return deliveryNoteLineID, nil
}

func countMissingKanbanLots(ctx context.Context, tx database.TenantTx, tenantID, purchaseOrderID uuid.UUID) (int64, error) {
	var missing int64
	err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(pol.total_kanban::bigint),0)-COUNT(kl.id)
 FROM purchase_order_lines pol
 JOIN delivery_note_lines dnl ON dnl.tenant_id=pol.tenant_id AND dnl.purchase_order_id=pol.purchase_order_id AND dnl.purchase_order_line_id=pol.id
 LEFT JOIN kanban_lots kl ON kl.tenant_id=dnl.tenant_id AND kl.delivery_note_line_id=dnl.id AND kl.purchase_order_line_id=pol.id
 WHERE pol.tenant_id=$1 AND pol.purchase_order_id=$2`, tenantID, purchaseOrderID).Scan(&missing)
	return missing, err
}

func validateExistingKanbanLots(ctx context.Context, tx database.TenantTx, tenantID, purchaseOrderID uuid.UUID, issuedAt time.Time) error {
	var inconsistent int64
	err := tx.QueryRow(ctx, `SELECT COUNT(*)
 FROM delivery_note_lines dnl
 JOIN purchase_order_lines pol ON pol.tenant_id=dnl.tenant_id AND pol.purchase_order_id=dnl.purchase_order_id AND pol.id=dnl.purchase_order_line_id
 JOIN kanban_lots kl ON kl.tenant_id=dnl.tenant_id AND kl.delivery_note_line_id=dnl.id
 WHERE dnl.tenant_id=$1 AND dnl.purchase_order_id=$2
  AND (kl.purchase_order_line_id<>pol.id OR kl.lot_number<1 OR kl.lot_number>pol.total_kanban
   OR kl.quantity<>pol.qty_per_kanban_snapshot OR kl.kanban_id NOT LIKE $3||'%')`, tenantID, purchaseOrderID, "KB-"+issuedAt.UTC().Format("200601")+"-").Scan(&inconsistent)
	if err != nil {
		return err
	}
	if inconsistent > 0 {
		return fmt.Errorf("purchase order %s has %d inconsistent Kanban lots", purchaseOrderID, inconsistent)
	}
	return nil
}

func reserveNumberBlock(ctx context.Context, tx database.TenantTx, table string, tenantID uuid.UUID, yearMonth string, count int64) (int64, error) {
	if count <= 0 {
		return 0, fmt.Errorf("number block size must be positive")
	}
	first, updateSQL, err := lockNumberSequence(ctx, tx, table, tenantID, yearMonth, count)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, updateSQL, tenantID, yearMonth, first+count); err != nil {
		return 0, err
	}
	return first, nil
}

func checkNumberBlockCapacity(ctx context.Context, tx database.TenantTx, table string, tenantID uuid.UUID, yearMonth string, count int64) error {
	if count <= 0 {
		return fmt.Errorf("number block size must be positive")
	}
	_, _, err := lockNumberSequence(ctx, tx, table, tenantID, yearMonth, count)
	return err
}

func lockNumberSequence(ctx context.Context, tx database.TenantTx, table string, tenantID uuid.UUID, yearMonth string, count int64) (int64, string, error) {
	var insertSQL, selectSQL, updateSQL, capacityMessage string
	var maxSequence int64
	switch table {
	case "delivery_note_number_sequences":
		maxSequence = maxDeliveryNoteSequence
		capacityMessage = "Monthly delivery note capacity is exhausted"
		insertSQL = `INSERT INTO delivery_note_number_sequences(tenant_id,year_month,next_value) VALUES($1,$2,1) ON CONFLICT(tenant_id,year_month) DO NOTHING`
		selectSQL = `SELECT next_value FROM delivery_note_number_sequences WHERE tenant_id=$1 AND year_month=$2 FOR UPDATE`
		updateSQL = `UPDATE delivery_note_number_sequences SET next_value=$3 WHERE tenant_id=$1 AND year_month=$2`
	case "kanban_number_sequences":
		maxSequence = maxKanbanSequence
		capacityMessage = "Monthly Kanban label capacity is exhausted"
		insertSQL = `INSERT INTO kanban_number_sequences(tenant_id,year_month,next_value) VALUES($1,$2,1) ON CONFLICT(tenant_id,year_month) DO NOTHING`
		selectSQL = `SELECT next_value FROM kanban_number_sequences WHERE tenant_id=$1 AND year_month=$2 FOR UPDATE`
		updateSQL = `UPDATE kanban_number_sequences SET next_value=$3 WHERE tenant_id=$1 AND year_month=$2`
	default:
		return 0, "", fmt.Errorf("unsupported number sequence %q", table)
	}
	if _, err := tx.Exec(ctx, insertSQL, tenantID, yearMonth); err != nil {
		return 0, "", err
	}
	var first int64
	if err := tx.QueryRow(ctx, selectSQL, tenantID, yearMonth).Scan(&first); err != nil {
		return 0, "", err
	}
	if first < 1 || count > maxSequence-first+1 {
		return 0, "", CapacityError{Field: "documents", Message: capacityMessage}
	}
	return first, updateSQL, nil
}
