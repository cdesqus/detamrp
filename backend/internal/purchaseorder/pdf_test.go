package purchaseorder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

type cancelOnPixelImage struct {
	image.Image
	cancel context.CancelFunc
}

func TestFormatPDFValuesTrimDatabaseScale(t *testing.T) {
	if got := formatPDFMoney(decimal.RequireFromString("10000000.000000"), "IDR"); got != "IDR 10.000.000" {
		t.Fatalf("IDR = %q", got)
	}
	if got := formatPDFDecimal(decimal.RequireFromString("12345.250000"), 6); got != "12.345,25" {
		t.Fatalf("quantity = %q", got)
	}
}

func (image cancelOnPixelImage) At(x, y int) color.Color {
	image.cancel()
	return image.Image.At(x, y)
}

func TestRenderPOPDFIncludesOrRedactsPrices(t *testing.T) {
	order := Order{
		PONumber: "PO-202607-00001", CompanyName: "PT Buyer Indonesia", SupplierName: "PT Material", Status: StatusApproved,
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
	if !pdfContainsText(priced, "IDR 200.000") || !pdfContainsText(priced, "IDR 400.000") {
		t.Fatal("authorized prices absent")
	}
	if !pdfContainsText(priced, "PT Buyer Indonesia") {
		t.Fatal("buyer company is missing from priced PO")
	}
	if pdfContainsLegacyDarkFill(priced) {
		t.Fatal("priced PO still uses solid section fills")
	}

	redacted, err := RenderPOPDF(order, false)
	if err != nil {
		t.Fatalf("render redacted PO: %v", err)
	}
	if pdfContainsText(redacted, "IDR 200.000") || pdfContainsText(redacted, "IDR 400.000") {
		t.Fatal("price leaked")
	}
	if !pdfContainsText(redacted, "PT Buyer Indonesia") {
		t.Fatal("buyer company is missing from redacted PO")
	}
	if pdfContainsLegacyDarkFill(redacted) {
		t.Fatal("redacted PO still uses solid section fills")
	}
}

func TestRenderPOPDFUsesModernBusinessSections(t *testing.T) {
	order := Order{PONumber: "PO-MODERN", SupplierName: "PT Modern", Status: StatusApproved, CreatedBy: Actor{DisplayName: "Buyer"}, SubmittedApproverDisplayName: "Director", Notes: "Deliver carefully"}
	result, err := RenderPOPDF(order, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, heading := range []string{"SUPPLIER DETAILS", "ORDER DETAILS", "APPROVAL", "ORDER SUMMARY"} {
		if !pdfContainsText(result, heading) {
			t.Errorf("missing modern section %q", heading)
		}
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

func TestDeliveryNoteQRUsesExactDNNumber(t *testing.T) {
	encoded, err := encodeDeliveryNoteQR("DN-202607-00002")
	if err != nil {
		t.Fatal(err)
	}
	if encoded.Content() != "DN-202607-00002" {
		t.Fatalf("QR content = %q", encoded.Content())
	}
}

func TestDeliveryNoteQRRejectsEmptyDNNumber(t *testing.T) {
	if _, err := encodeDeliveryNoteQR("   "); err == nil {
		t.Fatal("empty DN number was accepted")
	}
}

func TestDeliveryNoteQRPNG(t *testing.T) {
	result, err := deliveryNoteQRPNG("DN-202607-00002")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) < 8 || !bytes.Equal(result[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		t.Fatal("QR is not PNG data")
	}
}

func TestKanbanQRUsesExactKanbanID(t *testing.T) {
	encoded, err := encodeKanbanQR("KB-202607-00028")
	if err != nil {
		t.Fatal(err)
	}
	if encoded.Content() != "KB-202607-00028" {
		t.Fatalf("QR content = %q", encoded.Content())
	}
}

func TestKanbanQRRejectsEmptyID(t *testing.T) {
	if _, err := encodeKanbanQR("  "); err == nil {
		t.Fatal("empty Kanban ID was accepted")
	}
}

func TestKanbanQRPNG(t *testing.T) {
	result, err := kanbanQRPNG("KB-202607-00028")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) < 8 || !bytes.Equal(result[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		t.Fatal("Kanban QR is not PNG data")
	}
}

func TestDeliveryNoteUnitTotals(t *testing.T) {
	lines := []DeliveryNoteLine{
		{BaseUnitCode: "PCS", TotalQuantity: decimal.NewFromInt(20)},
		{BaseUnitCode: "KG", TotalQuantity: decimal.NewFromInt(10)},
		{BaseUnitCode: "KG", TotalQuantity: decimal.NewFromInt(5)},
	}
	got := deliveryNoteUnitTotals(lines)
	want := []string{"KG 15", "PCS 20"}
	if !slices.Equal(got, want) {
		t.Fatalf("totals = %#v, want %#v", got, want)
	}
}

func TestRenderDeliveryNotePDFUsesModernLayout(t *testing.T) {
	document := DeliveryNoteDocument{
		DeliveryNoteNumber: "DN-MODERN", PONumber: "PO-MODERN", SupplierName: "PT Modern",
		ExpectedDeliveryDate: time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		IssuedAt:             time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC),
		Lines: []DeliveryNoteLine{
			{RawMaterialCode: "RM-PCS", RawMaterialName: "Modern Part", BaseUnitCode: "PCS", QtyPerKanban: decimal.NewFromInt(10), TotalKanban: decimal.NewFromInt(2), TotalQuantity: decimal.NewFromInt(20)},
			{RawMaterialCode: "RM-KG", RawMaterialName: "Modern Coil", BaseUnitCode: "KG", QtyPerKanban: decimal.NewFromInt(5), TotalKanban: decimal.NewFromInt(3), TotalQuantity: decimal.NewFromInt(15)},
		},
	}

	result, err := RenderDeliveryNotePDF(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"Order Stock", "DELIVERY NOTE", "SCAN FOR RECEIVING", "MATERIAL DETAILS",
		"REMARKS", "SUPPLIER", "Prepared By", "RECEIVER", "Received By",
		"Total Kanban", "Total Quantity", "PCS 20", "KG 15",
	} {
		if !pdfContainsText(result, text) {
			t.Errorf("missing modern DN text %q", text)
		}
	}
	for _, redundant := range []string{"10.000000", "2.000000", "20.000000"} {
		if pdfContainsText(result, redundant) {
			t.Errorf("found redundant quantity %q", redundant)
		}
	}
}

func TestDeliveryNotePagination(t *testing.T) {
	lines := make([]DeliveryNoteLine, 45)
	for index := range lines {
		lines[index] = DeliveryNoteLine{
			RawMaterialCode: fmt.Sprintf("RM-%03d", index+1),
			RawMaterialName: "Long production material description requiring a wrapped table row",
			BaseUnitCode:    "PCS",
			QtyPerKanban:    decimal.NewFromInt(10),
			TotalKanban:     decimal.NewFromInt(2),
			TotalQuantity:   decimal.NewFromInt(20),
		}
	}
	result, err := RenderDeliveryNotePDF(DeliveryNoteDocument{
		DeliveryNoteNumber: "DN-PAGED",
		PONumber:           "PO-PAGED",
		SupplierName:       "PT Pagination",
		Lines:              lines,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pages := bytes.Count(result, []byte("/Type /Page\n")); pages < 2 {
		t.Fatalf("page count = %d, want multiple pages", pages)
	}
	if !pdfContainsText(result, "DELIVERY NOTE / CONTINUED") {
		t.Fatal("missing continuation heading")
	}
	if !pdfContainsText(result, "RM-045") || !pdfContainsText(result, "REMARKS") || !pdfContainsText(result, "RECEIVER") {
		t.Fatal("final row or footer is missing")
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
		Labels: []KanbanLabel{{KanbanID: "KB-UTF8-1", RawMaterialCode: "RM-Ø", RawMaterialName: "Baja élit", BaseUnitCode: "KG", CardNumber: 1, CardTotal: 1}},
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
	assertEmbeddedUnicodeText(t, "labels", labelPDF, "RM-Ø", "Baja élit")
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
		Labels: []KanbanLabel{{KanbanID: "KB-LONG-1", RawMaterialCode: "RM-LONG", RawMaterialName: longName, BaseUnitCode: "KG", CardNumber: 1, CardTotal: 1}},
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

func TestRenderKanbanLabelsPDFProducesOneOrderedQRCardPerLot(t *testing.T) {
	document := KanbanLabelDocument{
		DeliveryNoteNumber: "DN-202607-00001", PONumber: "PO-202607-00001",
		Labels: []KanbanLabel{
			{KanbanID: "KB-202607-000001", RawMaterialCode: "RM-A", RawMaterialName: "Alpha", Quantity: decimal.NewFromInt(10), BaseUnitCode: "KG", CardNumber: 1, CardTotal: 2},
			{KanbanID: "KB-202607-000002", RawMaterialCode: "RM-A", RawMaterialName: "Alpha", Quantity: decimal.NewFromInt(10), BaseUnitCode: "KG", CardNumber: 2, CardTotal: 2},
			{KanbanID: "KB-202607-000003", RawMaterialCode: "RM-B", RawMaterialName: "Bravo", Quantity: decimal.NewFromInt(5), BaseUnitCode: "L", CardNumber: 1, CardTotal: 1},
		},
	}

	original := encodeKanbanQRImage
	var encoded []string
	encodeKanbanQRImage = func(value string) (image.Image, error) {
		encoded = append(encoded, value)
		return image.NewRGBA(image.Rect(0, 0, 260, 260)), nil
	}
	t.Cleanup(func() { encodeKanbanQRImage = original })

	result, err := RenderKanbanLabelsPDF(document)
	if err != nil {
		t.Fatalf("render labels: %v", err)
	}
	if !bytes.HasPrefix(result, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}
	if got := pdfTextCount(result, "KANBAN CARD"); got != len(document.Labels) {
		t.Fatalf("cards=%d, want %d", got, len(document.Labels))
	}
	for index, label := range document.Labels {
		if encoded[index] != label.KanbanID {
			t.Fatalf("encoded[%d]=%q, want %q", index, encoded[index], label.KanbanID)
		}
	}
}

func TestRenderWideKanbanCard(t *testing.T) {
	document := KanbanLabelDocument{
		DeliveryNoteNumber: "DN-202607-00004",
		PONumber:           "PO-202607-00009",
		CompanyName:        "PT Buyer Indonesia",
		SupplierName:       "PT Supplier Sentosa",
		OrderDate:          time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC),
		Labels: []KanbanLabel{{
			KanbanID: "KB-202607-00028", RawMaterialCode: "BRG-123-00",
			RawMaterialName: "COIL MATERIAL", Quantity: decimal.RequireFromString("5.000000"),
			BaseUnitCode: "PC", CardNumber: 1, CardTotal: 5,
		}},
	}
	result, err := RenderKanbanLabelsPDF(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"PT Buyer Indonesia", "KANBAN CARD", "KANBAN ID", "PART NUMBER", "PART NAME",
		"PT Supplier Sentosa", "ORDER DATE", "21 Jul 2026", "QUANTITY", "CARD", "1/5",
		"DELIVERY NOTE", "PURCHASE ORDER", "KB-202607-00028", "5 PC",
	} {
		if !pdfContainsText(result, text) {
			t.Errorf("missing Kanban Card text %q", text)
		}
	}
	if pdfContainsText(result, "KANBAN LABEL") {
		t.Fatal("legacy Kanban Label title is still present")
	}
	if pdfContainsText(result, "LOT") {
		t.Fatal("legacy LOT label is still present")
	}
	if pdfContainsLegacyDarkFill(result) {
		t.Fatal("Kanban still uses a dark solid fill")
	}
}

func TestKanbanCardPagination(t *testing.T) {
	document := KanbanLabelDocument{DeliveryNoteNumber: "DN-PAGED", PONumber: "PO-PAGED"}
	for index := 1; index <= 4; index++ {
		document.Labels = append(document.Labels, KanbanLabel{
			KanbanID: fmt.Sprintf("KB-PAGED-%02d", index), RawMaterialCode: "RM-PAGED",
			RawMaterialName: "Wide Kanban Card", Quantity: decimal.NewFromInt(5),
			BaseUnitCode: "PC", CardNumber: index, CardTotal: 4,
		})
	}
	result, err := RenderKanbanLabelsPDF(document)
	if err != nil {
		t.Fatal(err)
	}
	if pages := bytes.Count(result, []byte("/Type /Page\n")); pages != 2 {
		t.Fatalf("page count = %d, want 2", pages)
	}
	if got := pdfTextCount(result, "KANBAN CARD"); got != 4 {
		t.Fatalf("card count = %d, want 4", got)
	}
	if !pdfContainsText(result, "KB-PAGED-04") || !pdfContainsText(result, "CUT HERE") {
		t.Fatal("fourth card or cut line is missing")
	}
}

func TestKanbanCardsDoNotUseCode128(t *testing.T) {
	original := encodeKanbanBarcode
	calls := 0
	encodeKanbanBarcode = func(string) (image.Image, error) {
		calls++
		return nil, errors.New("Code128 must not be used")
	}
	t.Cleanup(func() { encodeKanbanBarcode = original })

	_, err := RenderKanbanLabelsPDF(KanbanLabelDocument{Labels: []KanbanLabel{{
		KanbanID: "KB-QR-ONLY", RawMaterialCode: "RM", RawMaterialName: "QR only",
		Quantity: decimal.NewFromInt(1), BaseUnitCode: "PC", CardNumber: 1, CardTotal: 1,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("Code128 encoder called %d times", calls)
	}
}

func TestRenderKanbanLabelsPDFRejectsOversizedExportBeforeEncoding(t *testing.T) {
	document := KanbanLabelDocument{Labels: make([]KanbanLabel, 1001)}
	for index := range document.Labels {
		document.Labels[index].KanbanID = fmt.Sprintf("KB-%04d", index+1)
	}

	original := encodeKanbanQRImage
	encodeCalls := 0
	encodeKanbanQRImage = func(string) (image.Image, error) {
		encodeCalls++
		return nil, errors.New("QR encoding must not start")
	}
	t.Cleanup(func() { encodeKanbanQRImage = original })

	_, err := RenderKanbanLabelsPDF(document)
	var limit DocumentExportLimitError
	if !errors.As(err, &limit) || limit.Limit != 1000 || !strings.Contains(err.Error(), "1000") {
		t.Fatalf("oversized export error = %v, want safe 1000-label limit", err)
	}
	if encodeCalls != 0 {
		t.Fatalf("oversized export encoded %d QR codes, want none", encodeCalls)
	}
}

func TestRenderKanbanLabelsPDFStopsAfterRequestCancellation(t *testing.T) {
	document := KanbanLabelDocument{Labels: []KanbanLabel{
		{KanbanID: "KB-0001"},
		{KanbanID: "KB-0002"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	original := encodeKanbanQRImage
	encodeCalls := 0
	encodeKanbanQRImage = func(string) (image.Image, error) {
		encodeCalls++
		cancel()
		return image.NewRGBA(image.Rect(0, 0, 260, 260)), nil
	}
	t.Cleanup(func() { encodeKanbanQRImage = original })

	_, err := renderKanbanLabelsPDF(ctx, document)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled render error = %v, want context cancellation", err)
	}
	if encodeCalls != 1 {
		t.Fatalf("cancelled render encoded %d QR codes, want one", encodeCalls)
	}
}

func TestRenderKanbanLabelsPDFStopsBeforeSerializationAfterLateCancellation(t *testing.T) {
	document := KanbanLabelDocument{Labels: []KanbanLabel{{KanbanID: "KB-0001"}}}
	ctx, cancel := context.WithCancel(context.Background())
	original := encodeKanbanQRImage
	encodeKanbanQRImage = func(string) (image.Image, error) {
		return cancelOnPixelImage{Image: image.NewRGBA(image.Rect(0, 0, 260, 260)), cancel: cancel}, nil
	}
	t.Cleanup(func() { encodeKanbanQRImage = original })

	result, err := renderKanbanLabelsPDF(ctx, document)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("late-cancelled render error = %v, want context cancellation", err)
	}
	if len(result) != 0 {
		t.Fatalf("late-cancelled render returned %d PDF bytes, want none", len(result))
	}
}

func TestLoadDeliveryNoteDocumentUsesTenantFilteredQueriesAndAllLines(t *testing.T) {
	tenantID, orderID, deliveryNoteID := uuid.New(), uuid.New(), uuid.New()
	orderDate := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	issuedAt := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	query := &documentQueryRecorder{
		rows: []pgx.Row{
			documentRow(func(dest ...any) error {
				*dest[0].(*string), *dest[1].(*string), *dest[2].(*string) = "Buyer PT", "PO-1", "Supplier"
				*dest[3].(*time.Time), *dest[4].(*time.Time), *dest[5].(*Status) = orderDate, issuedAt, StatusApproved
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
	if document.DeliveryNoteID != deliveryNoteID || document.CompanyName != "Buyer PT" || !document.OrderDate.Equal(orderDate) || len(document.Lines) != 2 {
		t.Fatalf("document = %#v", document)
	}
	assertTenantQueries(t, query.calls, tenantID, orderID)
}

func TestLoadKanbanLabelDocumentUsesStableTenantFilteredOrdering(t *testing.T) {
	tenantID, orderID, deliveryNoteID := uuid.New(), uuid.New(), uuid.New()
	orderDate := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	issuedAt := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	query := &documentQueryRecorder{
		rows: []pgx.Row{
			documentRow(func(dest ...any) error {
				*dest[0].(*string), *dest[1].(*string), *dest[2].(*string) = "Buyer PT", "PO-1", "Supplier"
				*dest[3].(*time.Time), *dest[4].(*time.Time), *dest[5].(*Status) = orderDate, issuedAt, StatusApproved
				return nil
			}),
			documentRow(func(dest ...any) error {
				*dest[0].(*uuid.UUID), *dest[1].(*string), *dest[2].(*time.Time) = deliveryNoteID, "DN-1", issuedAt
				return nil
			}),
		},
		rowSets: []pgx.Rows{&documentRows{scans: []func(...any) error{
			func(dest ...any) error { setKanbanLabel(dest, "KB-1", "RM-A", "Alpha", "10", "KG", 1, 2); return nil },
			func(dest ...any) error { setKanbanLabel(dest, "KB-2", "RM-A", "Alpha", "10", "KG", 2, 2); return nil },
			func(dest ...any) error { setKanbanLabel(dest, "KB-3", "RM-B", "Bravo", "5", "L", 1, 1); return nil },
		}}},
	}

	document, err := loadKanbanLabelDocument(context.Background(), query, Actor{TenantID: tenantID}, orderID)
	if err != nil {
		t.Fatalf("load labels: %v", err)
	}
	if len(document.Labels) != 3 || document.CompanyName != "Buyer PT" || !document.OrderDate.Equal(orderDate) {
		t.Fatalf("labels = %#v", document.Labels)
	}
	got := [][2]int{
		{document.Labels[0].CardNumber, document.Labels[0].CardTotal},
		{document.Labels[1].CardNumber, document.Labels[1].CardTotal},
		{document.Labels[2].CardNumber, document.Labels[2].CardTotal},
	}
	want := [][2]int{{1, 2}, {2, 2}, {1, 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("card positions = %#v, want %#v", got, want)
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
				*dest[0].(*string), *dest[1].(*string), *dest[2].(*string) = "Buyer PT", "PO-1", "Supplier"
				*dest[3].(*time.Time), *dest[4].(*time.Time), *dest[5].(*Status) = time.Now(), time.Now(), StatusPendingApproval
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

func setKanbanLabel(dest []any, id, code, name, quantity, unit string, cardNumber, cardTotal int) {
	*dest[0].(*string), *dest[1].(*string), *dest[2].(*string) = id, code, name
	*dest[3].(*decimal.Decimal) = decimal.RequireFromString(quantity)
	*dest[4].(*string), *dest[5].(*int), *dest[6].(*int) = unit, cardNumber, cardTotal
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

func pdfContainsLegacyDarkFill(document []byte) bool {
	return bytes.Contains(document, []byte(" re f"))
}
