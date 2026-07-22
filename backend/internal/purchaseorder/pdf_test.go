package purchaseorder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

func TestRenderPOPDFIncludesOrRedactsPrices(t *testing.T) {
	order := Order{
		PONumber: "PO-202607-00001", SupplierName: "PT Material", Status: StatusApproved,
		OrderDate: time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), ExpectedDeliveryDate: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		Currency: "IDR", Notes: "Handle with care", TotalAmount: decimal.NewFromInt(400000), CreatedBy: Actor{DisplayName: "Buyer One"},
		Lines: []OrderLine{{RawMaterialCode: "RM-001", RawMaterialName: "Steel", BaseUnitCode: "KG", QtyPerKanbanSnapshot: decimal.NewFromInt(10), TotalKanban: decimal.NewFromInt(2), OrderedBaseQty: decimal.NewFromInt(20), UnitPriceSnapshot: decimal.NewFromInt(200000), LineTotal: decimal.NewFromInt(400000)}},
	}

	priced, err := RenderPOPDF(order, true)
	if err != nil {
		t.Fatalf("render priced PO: %v", err)
	}
	if !bytes.HasPrefix(priced, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}
	if !pdfContainsText(priced, "200000") || !pdfContainsText(priced, "400000") {
		t.Fatal("authorized prices absent")
	}

	redacted, err := RenderPOPDF(order, false)
	if err != nil {
		t.Fatalf("render redacted PO: %v", err)
	}
	if pdfContainsText(redacted, "200000") || pdfContainsText(redacted, "400000") {
		t.Fatal("price leaked")
	}
}

func TestRenderDeliveryNotePDFIncludesEveryLine(t *testing.T) {
	document := DeliveryNoteDocument{
		DeliveryNoteNumber: "DN-202607-00001", PONumber: "PO-202607-00001", SupplierName: "PT Material",
		ExpectedDeliveryDate: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC), IssuedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC),
		Lines: []DeliveryNoteLine{
			{RawMaterialCode: "RM-ALPHA", RawMaterialName: "Alpha", BaseUnitCode: "KG", QtyPerKanban: decimal.NewFromInt(10), TotalKanban: decimal.NewFromInt(2), TotalQuantity: decimal.NewFromInt(20)},
			{RawMaterialCode: "RM-BRAVO", RawMaterialName: "Bravo", BaseUnitCode: "L", QtyPerKanban: decimal.NewFromInt(5), TotalKanban: decimal.NewFromInt(3), TotalQuantity: decimal.NewFromInt(15)},
		},
	}

	result, err := RenderDeliveryNotePDF(document)
	if err != nil {
		t.Fatalf("render delivery note: %v", err)
	}
	if !bytes.HasPrefix(result, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}
	for _, materialCode := range []string{"RM-ALPHA", "RM-BRAVO"} {
		if !pdfContainsText(result, materialCode) {
			t.Fatalf("delivery note omitted %s", materialCode)
		}
	}
}

func TestRenderPDFsEmbedUnicodeText(t *testing.T) {
	order := Order{
		PONumber:     "PO-UTF8",
		SupplierName: "PT Maju Sejahterá",
		Notes:        "Harap simpan pada suhu ±5 °C – jangan terkena air",
		CreatedBy:    Actor{DisplayName: "José Pembeli"},
		Lines: []OrderLine{{
			RawMaterialCode: "RM-Ø", RawMaterialName: "Baja élit", BaseUnitCode: "KG",
		}},
	}
	deliveryNote := DeliveryNoteDocument{
		DeliveryNoteNumber: "DN-UTF8", PONumber: "PO-UTF8", SupplierName: "PT Maju Sejahterá",
		Lines: []DeliveryNoteLine{{RawMaterialCode: "RM-Ø", RawMaterialName: "Baja élit", BaseUnitCode: "KG"}},
	}
	labels := KanbanLabelDocument{
		DeliveryNoteNumber: "DN-UTF8", PONumber: "PO-UTF8",
		Labels: []KanbanLabel{{KanbanID: "KB-UTF8-1", RawMaterialCode: "RM-Ø", RawMaterialName: "Baja élit", BaseUnitCode: "KG", LotNumber: 1}},
	}

	po, err := RenderPOPDF(order, false)
	if err != nil {
		t.Fatalf("render Unicode PO: %v", err)
	}
	dn, err := RenderDeliveryNotePDF(deliveryNote)
	if err != nil {
		t.Fatalf("render Unicode delivery note: %v", err)
	}
	labelPDF, err := RenderKanbanLabelsPDF(labels)
	if err != nil {
		t.Fatalf("render Unicode labels: %v", err)
	}

	assertEmbeddedUnicodeText(t, "PO", po, "PT Maju Sejahterá", "Harap simpan pada suhu ±5 °C – jangan terkena air", "José Pembeli", "RM-Ø", "Baja élit")
	assertEmbeddedUnicodeText(t, "delivery note", dn, "PT Maju Sejahterá", "RM-Ø", "Baja élit")
	assertEmbeddedUnicodeText(t, "labels", labelPDF, "RM-Ø - Baja élit")
}

func TestRenderPDFsKeepLongTextWithinReadableLayout(t *testing.T) {
	longName := "Komponen " + strings.Repeat("W", 360)
	order := Order{
		PONumber: "PO-LONG", SupplierName: longName, Notes: strings.Repeat("Catatan penanganan harus dibaca dengan teliti. ", 30),
		CreatedBy: Actor{DisplayName: longName},
	}
	deliveryNote := DeliveryNoteDocument{DeliveryNoteNumber: "DN-LONG", PONumber: "PO-LONG", SupplierName: longName}
	for index := range 22 {
		order.Lines = append(order.Lines, OrderLine{
			RawMaterialCode: fmt.Sprintf("RM-%02d", index+1), RawMaterialName: longName, BaseUnitCode: "KG",
		})
		deliveryNote.Lines = append(deliveryNote.Lines, DeliveryNoteLine{
			RawMaterialCode: fmt.Sprintf("RM-%02d", index+1), RawMaterialName: longName, BaseUnitCode: "KG",
		})
	}
	labels := KanbanLabelDocument{
		DeliveryNoteNumber: "DN-LONG", PONumber: "PO-LONG",
		Labels: []KanbanLabel{{KanbanID: "KB-LONG-1", RawMaterialCode: "RM-LONG", RawMaterialName: longName, BaseUnitCode: "KG", LotNumber: 1}},
	}

	po, err := RenderPOPDF(order, false)
	if err != nil {
		t.Fatalf("render long PO: %v", err)
	}
	dn, err := RenderDeliveryNotePDF(deliveryNote)
	if err != nil {
		t.Fatalf("render long delivery note: %v", err)
	}
	labelPDF, err := RenderKanbanLabelsPDF(labels)
	if err != nil {
		t.Fatalf("render long labels: %v", err)
	}

	for documentType, document := range map[string][]byte{"PO": po, "delivery note": dn, "labels": labelPDF} {
		if !bytes.Contains(document, utf16BE("WW…")) {
			t.Errorf("%s PDF did not mark measured truncation with an ellipsis", documentType)
		}
	}
	for documentType, document := range map[string][]byte{"PO": po, "delivery note": dn} {
		if pages := bytes.Count(document, []byte("/Type /Page\n")); pages < 2 {
			t.Errorf("%s PDF used %d page for 22 long rows, want wrapped rows to flow across pages", documentType, pages)
		}
	}
}

func TestFitPDFTextLinesUsesMeasuredWidthAndMarksTruncation(t *testing.T) {
	pdf := newA4PDF("layout")
	pdf.AddPage()
	pdf.SetFont(pdfFontFamily, "", 8)
	const width = 32.0

	lines := fitPDFTextLines(pdf, strings.Repeat("material sangat panjang ", 12), width, 2)
	if len(lines) != 2 {
		t.Fatalf("fitted lines = %#v, want exactly two", lines)
	}
	if !strings.HasSuffix(lines[len(lines)-1], "…") {
		t.Fatalf("last fitted line = %q, want truncation marker", lines[len(lines)-1])
	}
	for _, line := range lines {
		if got := pdf.GetStringWidth(line); got > width-2 {
			t.Errorf("line %q width = %.2f, exceeds %.2f", line, got, width-2)
		}
	}
}

func TestRenderKanbanLabelsPDFProducesOneOrderedCode128LabelPerLot(t *testing.T) {
	document := KanbanLabelDocument{
		DeliveryNoteNumber: "DN-202607-00001", PONumber: "PO-202607-00001",
		Labels: []KanbanLabel{
			{KanbanID: "KB-202607-000001", RawMaterialCode: "RM-A", RawMaterialName: "Alpha", Quantity: decimal.NewFromInt(10), BaseUnitCode: "KG", LotNumber: 1},
			{KanbanID: "KB-202607-000002", RawMaterialCode: "RM-A", RawMaterialName: "Alpha", Quantity: decimal.NewFromInt(10), BaseUnitCode: "KG", LotNumber: 2},
			{KanbanID: "KB-202607-000003", RawMaterialCode: "RM-B", RawMaterialName: "Bravo", Quantity: decimal.NewFromInt(5), BaseUnitCode: "L", LotNumber: 1},
		},
	}

	original := encodeKanbanBarcode
	var encoded []string
	encodeKanbanBarcode = func(value string) (image.Image, error) {
		encoded = append(encoded, value)
		return image.NewRGBA(image.Rect(0, 0, 420, 72)), nil
	}
	t.Cleanup(func() { encodeKanbanBarcode = original })

	result, err := RenderKanbanLabelsPDF(document)
	if err != nil {
		t.Fatalf("render labels: %v", err)
	}
	if !bytes.HasPrefix(result, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}
	if got := pdfTextCount(result, "KANBAN LABEL"); got != len(document.Labels) {
		t.Fatalf("labels=%d, want %d", got, len(document.Labels))
	}
	for index, label := range document.Labels {
		if encoded[index] != label.KanbanID {
			t.Fatalf("encoded[%d]=%q, want %q", index, encoded[index], label.KanbanID)
		}
	}
}

func TestRenderKanbanLabelsPDFRejectsOversizedExportBeforeEncoding(t *testing.T) {
	document := KanbanLabelDocument{Labels: make([]KanbanLabel, 1001)}
	for index := range document.Labels {
		document.Labels[index].KanbanID = fmt.Sprintf("KB-%04d", index+1)
	}

	original := encodeKanbanBarcode
	encodeCalls := 0
	encodeKanbanBarcode = func(string) (image.Image, error) {
		encodeCalls++
		return nil, errors.New("barcode encoding must not start")
	}
	t.Cleanup(func() { encodeKanbanBarcode = original })

	_, err := RenderKanbanLabelsPDF(document)
	var limit DocumentExportLimitError
	if !errors.As(err, &limit) || limit.Limit != 1000 || !strings.Contains(err.Error(), "1000") {
		t.Fatalf("oversized export error = %v, want safe 1000-label limit", err)
	}
	if encodeCalls != 0 {
		t.Fatalf("oversized export encoded %d barcodes, want none", encodeCalls)
	}
}

func TestRenderKanbanLabelsPDFStopsAfterRequestCancellation(t *testing.T) {
	document := KanbanLabelDocument{Labels: []KanbanLabel{
		{KanbanID: "KB-0001"},
		{KanbanID: "KB-0002"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	original := encodeKanbanBarcode
	encodeCalls := 0
	encodeKanbanBarcode = func(string) (image.Image, error) {
		encodeCalls++
		cancel()
		return image.NewRGBA(image.Rect(0, 0, 420, 72)), nil
	}
	t.Cleanup(func() { encodeKanbanBarcode = original })

	_, err := renderKanbanLabelsPDF(ctx, document)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled render error = %v, want context cancellation", err)
	}
	if encodeCalls != 1 {
		t.Fatalf("cancelled render encoded %d barcodes, want one", encodeCalls)
	}
}

func TestLoadDeliveryNoteDocumentUsesTenantFilteredQueriesAndAllLines(t *testing.T) {
	tenantID, orderID, deliveryNoteID := uuid.New(), uuid.New(), uuid.New()
	issuedAt := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	query := &documentQueryRecorder{
		rows: []pgx.Row{
			documentRow(func(dest ...any) error {
				*dest[0].(*string), *dest[1].(*string), *dest[2].(*time.Time), *dest[3].(*Status) = "PO-1", "Supplier", issuedAt, StatusApproved
				return nil
			}),
			documentRow(func(dest ...any) error {
				*dest[0].(*uuid.UUID), *dest[1].(*string), *dest[2].(*time.Time) = deliveryNoteID, "DN-1", issuedAt
				return nil
			}),
		},
		rowSets: []pgx.Rows{&documentRows{scans: []func(...any) error{
			func(dest ...any) error { setDeliveryNoteLine(dest, "RM-A", "Alpha", "KG", "10", "2", "20"); return nil },
			func(dest ...any) error { setDeliveryNoteLine(dest, "RM-B", "Bravo", "L", "5", "3", "15"); return nil },
		}}},
	}

	document, err := loadDeliveryNoteDocument(context.Background(), query, Actor{TenantID: tenantID}, orderID)
	if err != nil {
		t.Fatalf("load delivery note: %v", err)
	}
	if document.DeliveryNoteID != deliveryNoteID || len(document.Lines) != 2 {
		t.Fatalf("document = %#v", document)
	}
	assertTenantQueries(t, query.calls, tenantID, orderID)
}

func TestLoadKanbanLabelDocumentUsesStableTenantFilteredOrdering(t *testing.T) {
	tenantID, orderID, deliveryNoteID := uuid.New(), uuid.New(), uuid.New()
	issuedAt := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	query := &documentQueryRecorder{
		rows: []pgx.Row{
			documentRow(func(dest ...any) error {
				*dest[0].(*string), *dest[1].(*string), *dest[2].(*time.Time), *dest[3].(*Status) = "PO-1", "Supplier", issuedAt, StatusApproved
				return nil
			}),
			documentRow(func(dest ...any) error {
				*dest[0].(*uuid.UUID), *dest[1].(*string), *dest[2].(*time.Time) = deliveryNoteID, "DN-1", issuedAt
				return nil
			}),
		},
		rowSets: []pgx.Rows{&documentRows{scans: []func(...any) error{
			func(dest ...any) error { setKanbanLabel(dest, "KB-2", "RM-A", "Alpha", "10", "KG", 2); return nil },
			func(dest ...any) error { setKanbanLabel(dest, "KB-3", "RM-B", "Bravo", "5", "L", 1); return nil },
		}}},
	}

	document, err := loadKanbanLabelDocument(context.Background(), query, Actor{TenantID: tenantID}, orderID)
	if err != nil {
		t.Fatalf("load labels: %v", err)
	}
	if len(document.Labels) != 2 || document.Labels[0].KanbanID != "KB-2" || document.Labels[1].KanbanID != "KB-3" {
		t.Fatalf("labels = %#v", document.Labels)
	}
	if !strings.Contains(strings.ToLower(query.calls[len(query.calls)-1].sql), "order by pol.sort_position,kl.lot_number") {
		t.Fatalf("labels query lacks stable ordering: %s", query.calls[len(query.calls)-1].sql)
	}
	labelsCall := query.calls[len(query.calls)-1]
	if !strings.Contains(strings.ToLower(labelsCall.sql), "limit $3") || len(labelsCall.args) != 3 || labelsCall.args[2] != 1001 {
		t.Fatalf("labels query is not bounded to max+1: %s args=%#v", labelsCall.sql, labelsCall.args)
	}
	assertTenantQueries(t, query.calls, tenantID, orderID)
}

func TestLoadDeliveryNoteDocumentDistinguishesOtherTenantFromUnavailableDocument(t *testing.T) {
	tenantID, orderID := uuid.New(), uuid.New()
	t.Run("other tenant", func(t *testing.T) {
		query := &documentQueryRecorder{rows: []pgx.Row{documentRow(func(...any) error { return pgx.ErrNoRows })}}
		_, err := loadDeliveryNoteDocument(context.Background(), query, Actor{TenantID: tenantID}, orderID)
		var notFound NotFoundError
		if !errors.As(err, &notFound) {
			t.Fatalf("error = %v, want NotFoundError", err)
		}
	})
	t.Run("operational document unavailable", func(t *testing.T) {
		query := &documentQueryRecorder{rows: []pgx.Row{
			documentRow(func(dest ...any) error {
				*dest[0].(*string), *dest[1].(*string), *dest[2].(*time.Time), *dest[3].(*Status) = "PO-1", "Supplier", time.Now(), StatusPendingApproval
				return nil
			}),
			documentRow(func(...any) error { return pgx.ErrNoRows }),
		}}
		_, err := loadDeliveryNoteDocument(context.Background(), query, Actor{TenantID: tenantID}, orderID)
		var conflict ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("error = %v, want ConflictError", err)
		}
	})
}

type documentQueryCall struct {
	sql  string
	args []any
}

type documentQueryRecorder struct {
	rows    []pgx.Row
	rowSets []pgx.Rows
	calls   []documentQueryCall
}

func (r *documentQueryRecorder) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	r.calls = append(r.calls, documentQueryCall{sql: sql, args: args})
	row := r.rows[0]
	r.rows = r.rows[1:]
	return row
}

func (r *documentQueryRecorder) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	r.calls = append(r.calls, documentQueryCall{sql: sql, args: args})
	rows := r.rowSets[0]
	r.rowSets = r.rowSets[1:]
	return rows, nil
}

type documentRow func(...any) error

func (r documentRow) Scan(dest ...any) error { return r(dest...) }

type documentRows struct {
	scans  []func(...any) error
	index  int
	closed bool
}

func (r *documentRows) Close()                                       { r.closed = true }
func (r *documentRows) Err() error                                   { return nil }
func (r *documentRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *documentRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *documentRows) Values() ([]any, error)                       { return nil, fmt.Errorf("not implemented") }
func (r *documentRows) RawValues() [][]byte                          { return nil }
func (r *documentRows) Conn() *pgx.Conn                              { return nil }
func (r *documentRows) Next() bool {
	if r.index >= len(r.scans) {
		r.Close()
		return false
	}
	r.index++
	return true
}
func (r *documentRows) Scan(dest ...any) error { return r.scans[r.index-1](dest...) }

func setDeliveryNoteLine(dest []any, code, name, unit, qtyPerKanban, totalKanban, totalQuantity string) {
	*dest[0].(*string), *dest[1].(*string), *dest[2].(*string) = code, name, unit
	*dest[3].(*decimal.Decimal) = decimal.RequireFromString(qtyPerKanban)
	*dest[4].(*decimal.Decimal) = decimal.RequireFromString(totalKanban)
	*dest[5].(*decimal.Decimal) = decimal.RequireFromString(totalQuantity)
}

func setKanbanLabel(dest []any, id, code, name, quantity, unit string, lot int) {
	*dest[0].(*string), *dest[1].(*string), *dest[2].(*string) = id, code, name
	*dest[3].(*decimal.Decimal) = decimal.RequireFromString(quantity)
	*dest[4].(*string), *dest[5].(*int) = unit, lot
}

func assertTenantQueries(t *testing.T, calls []documentQueryCall, tenantID, orderID uuid.UUID) {
	t.Helper()
	for _, call := range calls {
		if !strings.Contains(strings.ToLower(call.sql), "tenant_id=$1") {
			t.Errorf("query lacks tenant predicate: %s", call.sql)
		}
		if len(call.args) < 2 || call.args[0] != tenantID || call.args[1] != orderID {
			t.Errorf("query args = %#v, want tenant and order", call.args)
		}
	}
}

func assertEmbeddedUnicodeText(t *testing.T, documentType string, document []byte, values ...string) {
	t.Helper()
	if !bytes.Contains(document, []byte("/ToUnicode")) {
		t.Fatalf("%s PDF has no embedded Unicode character map", documentType)
	}
	for _, value := range values {
		if !bytes.Contains(document, utf16BE(value)) {
			t.Errorf("%s PDF does not contain UTF-16BE text %q", documentType, value)
		}
	}
}

func utf16BE(value string) []byte {
	result := make([]byte, 0, len(value)*2)
	for _, character := range value {
		result = append(result, byte(character>>8), byte(character))
	}
	return result
}

func pdfContainsText(document []byte, value string) bool {
	return bytes.Contains(document, []byte(value)) || bytes.Contains(document, utf16BE(value))
}

func pdfTextCount(document []byte, value string) int {
	return bytes.Count(document, []byte(value)) + bytes.Count(document, utf16BE(value))
}
