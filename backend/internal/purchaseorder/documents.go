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

type missingKanbanLot struct {
	deliveryNoteLineID  uuid.UUID
	purchaseOrderLineID uuid.UUID
	lotNumber           int64
	quantity            decimal.Decimal
}

const (
	maxDeliveryNoteSequence int64 = 99_999
	maxKanbanSequence       int64 = 999_999
)

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

	missingLots, err := findMissingKanbanLots(ctx, tx, actor.TenantID, issuedAt, lines)
	if err != nil {
		return err
	}
	if len(missingLots) == 0 {
		return nil
	}
	firstNumber, err := reserveNumberBlock(ctx, tx, "kanban_number_sequences", actor.TenantID, issuedAt.UTC().Format("200601"), int64(len(missingLots)))
	if err != nil {
		return err
	}
	for index, lot := range missingLots {
		kanbanID := fmt.Sprintf("KB-%s-%06d", issuedAt.UTC().Format("200601"), firstNumber+int64(index))
		var insertedID uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO kanban_lots(tenant_id,delivery_note_line_id,purchase_order_line_id,kanban_id,lot_number,quantity,created_by_user_id,updated_by_user_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$7)
 ON CONFLICT(tenant_id,delivery_note_line_id,purchase_order_line_id,lot_number) DO NOTHING
 RETURNING id`, actor.TenantID, lot.deliveryNoteLineID, lot.purchaseOrderLineID, kanbanID, lot.lotNumber, lot.quantity, actor.UserID).Scan(&insertedID)
		if errors.Is(err, pgx.ErrNoRows) {
			if err := validateExistingKanbanLot(ctx, tx, actor.TenantID, issuedAt, lot); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
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

func findMissingKanbanLots(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID, issuedAt time.Time, lines []approvedDocumentLine) ([]missingKanbanLot, error) {
	var missing []missingKanbanLot
	for _, line := range lines {
		existing, err := loadExistingKanbanLots(ctx, tx, tenantID, issuedAt, line)
		if err != nil {
			return nil, err
		}
		for lotNumber := int64(1); lotNumber <= line.TotalKanban.IntPart(); lotNumber++ {
			if _, found := existing[lotNumber]; found {
				continue
			}
			missing = append(missing, missingKanbanLot{
				deliveryNoteLineID:  line.DeliveryNoteLineID,
				purchaseOrderLineID: line.ID,
				lotNumber:           lotNumber,
				quantity:            line.QtyPerKanbanSnapshot,
			})
		}
	}
	return missing, nil
}

func loadExistingKanbanLots(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID, issuedAt time.Time, line approvedDocumentLine) (map[int64]struct{}, error) {
	rows, err := tx.Query(ctx, `SELECT delivery_note_line_id,purchase_order_line_id,kanban_id,lot_number,quantity
 FROM kanban_lots WHERE tenant_id=$1 AND delivery_note_line_id=$2 ORDER BY lot_number`, tenantID, line.DeliveryNoteLineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	existing := make(map[int64]struct{})
	for rows.Next() {
		var deliveryNoteLineID, purchaseOrderLineID uuid.UUID
		var kanbanID string
		var lotNumber int64
		var quantity decimal.Decimal
		if err := rows.Scan(&deliveryNoteLineID, &purchaseOrderLineID, &kanbanID, &lotNumber, &quantity); err != nil {
			return nil, err
		}
		if deliveryNoteLineID != line.DeliveryNoteLineID || purchaseOrderLineID != line.ID ||
			lotNumber < 1 || lotNumber > line.TotalKanban.IntPart() || !quantity.Equal(line.QtyPerKanbanSnapshot) ||
			!strings.HasPrefix(kanbanID, "KB-"+issuedAt.UTC().Format("200601")+"-") {
			return nil, fmt.Errorf("existing Kanban lot %q for PO line %s is inconsistent", kanbanID, line.ID)
		}
		existing[lotNumber] = struct{}{}
	}
	return existing, rows.Err()
}

func validateExistingKanbanLot(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID, issuedAt time.Time, expected missingKanbanLot) error {
	var deliveryNoteLineID, purchaseOrderLineID uuid.UUID
	var kanbanID string
	var lotNumber int64
	var quantity decimal.Decimal
	err := tx.QueryRow(ctx, `SELECT delivery_note_line_id,purchase_order_line_id,kanban_id,lot_number,quantity
 FROM kanban_lots
 WHERE tenant_id=$1 AND delivery_note_line_id=$2 AND purchase_order_line_id=$3 AND lot_number=$4`,
		tenantID, expected.deliveryNoteLineID, expected.purchaseOrderLineID, expected.lotNumber).
		Scan(&deliveryNoteLineID, &purchaseOrderLineID, &kanbanID, &lotNumber, &quantity)
	if err != nil {
		return err
	}
	if deliveryNoteLineID != expected.deliveryNoteLineID || purchaseOrderLineID != expected.purchaseOrderLineID ||
		lotNumber != expected.lotNumber || !quantity.Equal(expected.quantity) ||
		!strings.HasPrefix(kanbanID, "KB-"+issuedAt.UTC().Format("200601")+"-") {
		return fmt.Errorf("existing Kanban lot %q for PO line %s is inconsistent", kanbanID, expected.purchaseOrderLineID)
	}
	return nil
}

func reserveNumberBlock(ctx context.Context, tx database.TenantTx, table string, tenantID uuid.UUID, yearMonth string, count int64) (int64, error) {
	if count <= 0 {
		return 0, fmt.Errorf("number block size must be positive")
	}
	var insertSQL, selectSQL, updateSQL string
	var maxSequence int64
	switch table {
	case "delivery_note_number_sequences":
		maxSequence = maxDeliveryNoteSequence
		insertSQL = `INSERT INTO delivery_note_number_sequences(tenant_id,year_month,next_value) VALUES($1,$2,1) ON CONFLICT(tenant_id,year_month) DO NOTHING`
		selectSQL = `SELECT next_value FROM delivery_note_number_sequences WHERE tenant_id=$1 AND year_month=$2 FOR UPDATE`
		updateSQL = `UPDATE delivery_note_number_sequences SET next_value=$3 WHERE tenant_id=$1 AND year_month=$2`
	case "kanban_number_sequences":
		maxSequence = maxKanbanSequence
		insertSQL = `INSERT INTO kanban_number_sequences(tenant_id,year_month,next_value) VALUES($1,$2,1) ON CONFLICT(tenant_id,year_month) DO NOTHING`
		selectSQL = `SELECT next_value FROM kanban_number_sequences WHERE tenant_id=$1 AND year_month=$2 FOR UPDATE`
		updateSQL = `UPDATE kanban_number_sequences SET next_value=$3 WHERE tenant_id=$1 AND year_month=$2`
	default:
		return 0, fmt.Errorf("unsupported number sequence %q", table)
	}
	if _, err := tx.Exec(ctx, insertSQL, tenantID, yearMonth); err != nil {
		return 0, err
	}
	var first int64
	if err := tx.QueryRow(ctx, selectSQL, tenantID, yearMonth).Scan(&first); err != nil {
		return 0, err
	}
	if first < 1 || count > maxSequence-first+1 {
		return 0, fmt.Errorf("%s sequence %s has insufficient capacity for block of %d", table, yearMonth, count)
	}
	if _, err := tx.Exec(ctx, updateSQL, tenantID, yearMonth, first+count); err != nil {
		return 0, err
	}
	return first, nil
}
