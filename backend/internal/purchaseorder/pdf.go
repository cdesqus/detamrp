package purchaseorder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
)

type DeliveryNoteDocument struct {
	DeliveryNoteID       uuid.UUID
	DeliveryNoteNumber   string
	PurchaseOrderID      uuid.UUID
	PONumber             string
	SupplierName         string
	ExpectedDeliveryDate time.Time
	IssuedAt             time.Time
	Lines                []DeliveryNoteLine
}

type DeliveryNoteLine struct {
	RawMaterialCode string
	RawMaterialName string
	BaseUnitCode    string
	QtyPerKanban    decimal.Decimal
	TotalKanban     decimal.Decimal
	TotalQuantity   decimal.Decimal
}

type KanbanLabelDocument struct {
	DeliveryNoteID     uuid.UUID
	DeliveryNoteNumber string
	PurchaseOrderID    uuid.UUID
	PONumber           string
	SupplierName       string
	Labels             []KanbanLabel
}

type KanbanLabel struct {
	KanbanID        string
	RawMaterialCode string
	RawMaterialName string
	Quantity        decimal.Decimal
	BaseUnitCode    string
	LotNumber       int
}

func newA4PDF(title string) *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(false)
	pdf.SetTitle(title, false)
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 14)
	return pdf
}

func outputPDF(pdf *fpdf.Fpdf) ([]byte, error) {
	var result bytes.Buffer
	if err := pdf.Output(&result); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func RenderPOPDF(order Order, includePrices bool) ([]byte, error) {
	pdf := newA4PDF(order.PONumber)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 9, "PURCHASE ORDER", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	writeKeyValue(pdf, "PO Number", order.PONumber)
	writeKeyValue(pdf, "Status", string(order.Status))
	writeKeyValue(pdf, "Supplier", order.SupplierName)
	writeKeyValue(pdf, "Order Date", formatPDFDate(order.OrderDate))
	writeKeyValue(pdf, "Expected Delivery", formatPDFDate(order.ExpectedDeliveryDate))
	writeKeyValue(pdf, "Currency", order.Currency)
	writeKeyValue(pdf, "Created By", order.CreatedBy.DisplayName)
	writeKeyValue(pdf, "Notes", order.Notes)
	pdf.Ln(3)

	widths := []float64{23, 47, 15, 24, 20, 27}
	headings := []string{"Material", "Description", "Unit", "Qty/Kanban", "Kanbans", "Total Qty"}
	if includePrices {
		widths = []float64{20, 33, 12, 20, 16, 22, 28, 29}
		headings = []string{"Material", "Description", "Unit", "Qty/Kanban", "Kanbans", "Total Qty", "Unit Price", "Line Total"}
	}
	pdf.SetFont("Arial", "B", 8)
	for index, heading := range headings {
		pdf.CellFormat(widths[index], 7, heading, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Arial", "", 8)
	totalBaseQty := decimal.Zero
	for _, line := range order.Lines {
		values := []string{line.RawMaterialCode, line.RawMaterialName, line.BaseUnitCode, line.QtyPerKanbanSnapshot.String(), line.TotalKanban.String(), line.OrderedBaseQty.String()}
		if includePrices {
			values = append(values, line.UnitPriceSnapshot.String(), line.LineTotal.String())
		}
		for index, value := range values {
			pdf.CellFormat(widths[index], 7, value, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
		totalBaseQty = totalBaseQty.Add(line.OrderedBaseQty)
	}
	pdf.Ln(3)
	writeKeyValue(pdf, "Total Base Quantity", totalBaseQty.String())
	if includePrices {
		writeKeyValue(pdf, "Total Amount", order.TotalAmount.String())
	}
	return outputPDF(pdf)
}

func RenderDeliveryNotePDF(document DeliveryNoteDocument) ([]byte, error) {
	pdf := newA4PDF(document.DeliveryNoteNumber)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 9, "DELIVERY NOTE", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	writeKeyValue(pdf, "DN Number", document.DeliveryNoteNumber)
	writeKeyValue(pdf, "PO Number", document.PONumber)
	writeKeyValue(pdf, "Supplier", document.SupplierName)
	writeKeyValue(pdf, "Expected Delivery", formatPDFDate(document.ExpectedDeliveryDate))
	writeKeyValue(pdf, "Issued At", formatPDFDate(document.IssuedAt))
	pdf.Ln(3)

	widths := []float64{24, 51, 16, 28, 25, 30}
	headings := []string{"Material", "Description", "Unit", "Qty/Kanban", "Kanbans", "Total Qty"}
	pdf.SetFont("Arial", "B", 8)
	for index, heading := range headings {
		pdf.CellFormat(widths[index], 7, heading, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Arial", "", 8)
	for _, line := range document.Lines {
		values := []string{line.RawMaterialCode, line.RawMaterialName, line.BaseUnitCode, line.QtyPerKanban.String(), line.TotalKanban.String(), line.TotalQuantity.String()}
		for index, value := range values {
			pdf.CellFormat(widths[index], 7, value, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	return outputPDF(pdf)
}

var encodeKanbanBarcode = func(value string) (image.Image, error) {
	encoded, err := code128.Encode(value)
	if err != nil {
		return nil, err
	}
	return barcode.Scale(encoded, 420, 72)
}

func RenderKanbanLabelsPDF(document KanbanLabelDocument) ([]byte, error) {
	pdf := newA4PDF("KANBAN-" + document.DeliveryNoteNumber)
	pdf.SetAutoPageBreak(false, 0)
	const labelWidth, labelHeight = 90.0, 64.0
	const leftX, rightX, topY = 12.0, 108.0, 12.0

	for index, label := range document.Labels {
		rowOnPage := (index / 2) % 4
		if index%8 == 0 {
			pdf.AddPage()
		}
		x := leftX
		if index%2 == 1 {
			x = rightX
		}
		y := topY + float64(rowOnPage)*68
		pdf.Rect(x, y, labelWidth, labelHeight, "D")
		pdf.SetXY(x+3, y+2)
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(labelWidth-6, 5, "KANBAN LABEL", "", 1, "C", false, 0, "")
		pdf.SetX(x + 3)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(labelWidth-6, 5, label.KanbanID, "", 1, "C", false, 0, "")
		pdf.SetX(x + 3)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(labelWidth-6, 4, "DN: "+document.DeliveryNoteNumber+"  PO: "+document.PONumber, "", 1, "L", false, 0, "")
		pdf.SetX(x + 3)
		pdf.CellFormat(labelWidth-6, 4, label.RawMaterialCode+" - "+label.RawMaterialName, "", 1, "L", false, 0, "")
		pdf.SetX(x + 3)
		pdf.CellFormat(labelWidth-6, 4, "Lot: "+strconv.Itoa(label.LotNumber)+"  Qty: "+label.Quantity.String()+" "+label.BaseUnitCode, "", 1, "L", false, 0, "")

		barcodeImage, err := encodeKanbanBarcode(label.KanbanID)
		if err != nil {
			return nil, fmt.Errorf("encode Kanban %q: %w", label.KanbanID, err)
		}
		var barcodePNG bytes.Buffer
		if err := png.Encode(&barcodePNG, barcodeImage); err != nil {
			return nil, fmt.Errorf("render Kanban %q barcode: %w", label.KanbanID, err)
		}
		imageName := "kanban-" + strconv.Itoa(index)
		pdf.RegisterImageOptionsReader(imageName, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, bytes.NewReader(barcodePNG.Bytes()))
		pdf.ImageOptions(imageName, x+5, y+29, labelWidth-10, 0, false, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, 0, "")
	}
	return outputPDF(pdf)
}

func writeKeyValue(pdf *fpdf.Fpdf, key, value string) {
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(38, 5, key, "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(0, 5, value, "", 1, "L", false, 0, "")
}

func formatPDFDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("02 Jan 2006")
}

type documentQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type documentHeader struct {
	PONumber             string
	SupplierName         string
	ExpectedDeliveryDate time.Time
	Status               Status
	DeliveryNoteID       uuid.UUID
	DeliveryNoteNumber   string
	IssuedAt             time.Time
}

func (s *SQLStore) LoadDeliveryNoteDocument(ctx context.Context, actor Actor, purchaseOrderID uuid.UUID) (document DeliveryNoteDocument, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		document, err = loadDeliveryNoteDocument(ctx, tx, actor, purchaseOrderID)
		return err
	})
	return
}

func (s *SQLStore) LoadKanbanLabelDocument(ctx context.Context, actor Actor, purchaseOrderID uuid.UUID) (document KanbanLabelDocument, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		document, err = loadKanbanLabelDocument(ctx, tx, actor, purchaseOrderID)
		return err
	})
	return
}

func loadDeliveryNoteDocument(ctx context.Context, query documentQuerier, actor Actor, purchaseOrderID uuid.UUID) (DeliveryNoteDocument, error) {
	header, err := loadOperationalDocumentHeader(ctx, query, actor.TenantID, purchaseOrderID, "deliveryNote")
	if err != nil {
		return DeliveryNoteDocument{}, err
	}
	document := DeliveryNoteDocument{
		DeliveryNoteID: header.DeliveryNoteID, DeliveryNoteNumber: header.DeliveryNoteNumber,
		PurchaseOrderID: purchaseOrderID, PONumber: header.PONumber, SupplierName: header.SupplierName,
		ExpectedDeliveryDate: header.ExpectedDeliveryDate, IssuedAt: header.IssuedAt,
	}
	rows, err := query.Query(ctx, `SELECT pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,pol.base_unit_code_snapshot,
 pol.qty_per_kanban_snapshot,pol.total_kanban,pol.ordered_base_qty
 FROM delivery_note_lines dnl
 JOIN purchase_order_lines pol ON pol.tenant_id=dnl.tenant_id AND pol.purchase_order_id=dnl.purchase_order_id AND pol.id=dnl.purchase_order_line_id
 WHERE dnl.tenant_id=$1 AND dnl.purchase_order_id=$2
 ORDER BY pol.sort_position`, actor.TenantID, purchaseOrderID)
	if err != nil {
		return DeliveryNoteDocument{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var line DeliveryNoteLine
		if err := rows.Scan(&line.RawMaterialCode, &line.RawMaterialName, &line.BaseUnitCode, &line.QtyPerKanban, &line.TotalKanban, &line.TotalQuantity); err != nil {
			return DeliveryNoteDocument{}, err
		}
		document.Lines = append(document.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return DeliveryNoteDocument{}, err
	}
	if len(document.Lines) == 0 {
		return DeliveryNoteDocument{}, documentConflict("deliveryNote", "Delivery Note lines are unavailable")
	}
	return document, nil
}

func loadKanbanLabelDocument(ctx context.Context, query documentQuerier, actor Actor, purchaseOrderID uuid.UUID) (KanbanLabelDocument, error) {
	header, err := loadOperationalDocumentHeader(ctx, query, actor.TenantID, purchaseOrderID, "kanbanLabels")
	if err != nil {
		return KanbanLabelDocument{}, err
	}
	document := KanbanLabelDocument{
		DeliveryNoteID: header.DeliveryNoteID, DeliveryNoteNumber: header.DeliveryNoteNumber,
		PurchaseOrderID: purchaseOrderID, PONumber: header.PONumber, SupplierName: header.SupplierName,
	}
	rows, err := query.Query(ctx, `SELECT kl.kanban_id,pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,
 kl.quantity,pol.base_unit_code_snapshot,kl.lot_number
 FROM delivery_note_lines dnl
 JOIN purchase_order_lines pol ON pol.tenant_id=dnl.tenant_id AND pol.purchase_order_id=dnl.purchase_order_id AND pol.id=dnl.purchase_order_line_id
 JOIN kanban_lots kl ON kl.tenant_id=dnl.tenant_id AND kl.delivery_note_line_id=dnl.id AND kl.purchase_order_line_id=pol.id
 WHERE dnl.tenant_id=$1 AND dnl.purchase_order_id=$2
 ORDER BY pol.sort_position,kl.lot_number`, actor.TenantID, purchaseOrderID)
	if err != nil {
		return KanbanLabelDocument{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var label KanbanLabel
		if err := rows.Scan(&label.KanbanID, &label.RawMaterialCode, &label.RawMaterialName, &label.Quantity, &label.BaseUnitCode, &label.LotNumber); err != nil {
			return KanbanLabelDocument{}, err
		}
		document.Labels = append(document.Labels, label)
	}
	if err := rows.Err(); err != nil {
		return KanbanLabelDocument{}, err
	}
	if len(document.Labels) == 0 {
		return KanbanLabelDocument{}, documentConflict("kanbanLabels", "Kanban labels are unavailable")
	}
	return document, nil
}

func loadOperationalDocumentHeader(ctx context.Context, query documentQuerier, tenantID, purchaseOrderID uuid.UUID, field string) (documentHeader, error) {
	var header documentHeader
	err := query.QueryRow(ctx, `SELECT p.po_number,s.name,p.expected_delivery_date,p.status
 FROM purchase_orders p
 JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id
 WHERE p.tenant_id=$1 AND p.id=$2`, tenantID, purchaseOrderID).
		Scan(&header.PONumber, &header.SupplierName, &header.ExpectedDeliveryDate, &header.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return documentHeader{}, NotFoundError{Resource: "purchase order"}
	}
	if err != nil {
		return documentHeader{}, err
	}
	if header.Status != StatusApproved {
		return documentHeader{}, documentConflict(field, "Purchase order operational documents are unavailable")
	}
	err = query.QueryRow(ctx, `SELECT dn.id,dn.delivery_note_number,dn.issued_at
 FROM delivery_notes dn WHERE dn.tenant_id=$1 AND dn.purchase_order_id=$2`, tenantID, purchaseOrderID).
		Scan(&header.DeliveryNoteID, &header.DeliveryNoteNumber, &header.IssuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return documentHeader{}, documentConflict(field, "Operational document is unavailable")
	}
	return header, err
}

func documentConflict(field, message string) ConflictError {
	return ConflictError{Fields: FieldErrors{field: message}}
}
