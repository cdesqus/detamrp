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
	if !bytes.Contains(priced, []byte("200000")) || !bytes.Contains(priced, []byte("400000")) {
		t.Fatal("authorized prices absent")
	}

	redacted, err := RenderPOPDF(order, false)
	if err != nil {
		t.Fatalf("render redacted PO: %v", err)
	}
	if bytes.Contains(redacted, []byte("200000")) || bytes.Contains(redacted, []byte("400000")) {
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
		if !bytes.Contains(result, []byte(materialCode)) {
			t.Fatalf("delivery note omitted %s", materialCode)
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
	if got := bytes.Count(result, []byte("KANBAN LABEL")); got != len(document.Labels) {
		t.Fatalf("labels=%d, want %d", got, len(document.Labels))
	}
	for index, label := range document.Labels {
		if encoded[index] != label.KanbanID {
			t.Fatalf("encoded[%d]=%q, want %q", index, encoded[index], label.KanbanID)
		}
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
		if len(call.args) != 2 || call.args[0] != tenantID || call.args[1] != orderID {
			t.Errorf("query args = %#v, want tenant and order", call.args)
		}
	}
}
