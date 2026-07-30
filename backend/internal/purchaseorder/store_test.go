package purchaseorder

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestGetOrderHeaderQueryLocksHeaderBeforeLoadingLines(t *testing.T) {
	if !strings.Contains(strings.ToLower(getOrderHeaderQuery), "for share") {
		t.Fatal("GetOrder header query must hold a shared row lock while loading lines")
	}
}

func TestOrderSelectIncludesSupplierDisplayName(t *testing.T) {
	query := strings.ToLower(orderSelect)
	for _, fragment := range []string{
		"s.name",
		"join suppliers s on s.tenant_id=p.tenant_id and s.id=p.supplier_id",
		"join tenant_settings ts on ts.tenant_id=p.tenant_id",
		"coalesce(nullif(btrim(ts.company_name),''),'order stock')",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("order query missing %q", fragment)
		}
	}
}

func TestOrderQueriesIncludePlantAndMaterialReferenceSnapshots(t *testing.T) {
	for _, fragment := range []string{
		"p.plant_id", "p.plant_code_snapshot", "p.plant_name_snapshot", "p.plant_address_snapshot",
	} {
		if !strings.Contains(strings.ToLower(orderSelect), fragment) {
			t.Errorf("order query missing %q", fragment)
		}
	}
	for _, fragment := range []string{
		"l.category_code_snapshot", "l.category_name_snapshot",
		"l.packing_code_snapshot", "l.packing_name_snapshot",
	} {
		if !strings.Contains(strings.ToLower(orderLineSelect), fragment) {
			t.Errorf("order line query missing %q", fragment)
		}
	}
}

func TestSnapshotQueriesRequireActivePlantCategoryAndPacking(t *testing.T) {
	plantQuery := strings.ToLower(activePlantSelect)
	for _, fragment := range []string{"from plants", "tenant_id=$1", "id=$2", "active", "for share"} {
		if !strings.Contains(plantQuery, fragment) {
			t.Errorf("active Plant query missing %q", fragment)
		}
	}
	materialQuery := strings.ToLower(snapshotMaterialSelect)
	for _, fragment := range []string{
		"join categories c", "c.active", "join packings pk", "pk.active",
		"c.code", "c.name", "pk.code", "pk.name",
	} {
		if !strings.Contains(materialQuery, fragment) {
			t.Errorf("material snapshot query missing %q", fragment)
		}
	}
}

func TestApprovalSelectIncludesPurchaseOrderAndSupplierDetails(t *testing.T) {
	query := strings.ToLower(approvalSelect)
	for _, fragment := range []string{
		"p.po_number", "p.supplier_id", "s.name",
		"join purchase_orders p on p.tenant_id=a.tenant_id and p.id=a.purchase_order_id",
		"join suppliers s on s.tenant_id=p.tenant_id and s.id=p.supplier_id",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("approval query missing %q", fragment)
		}
	}
}

func TestKanbanLotInsertIsSetWiseAndBounded(t *testing.T) {
	query := strings.ToLower(insertMissingKanbanLotsSQL)
	for _, fragment := range []string{"insert into kanban_lots", "generate_series", "row_number()", "lot_number <= pol.total_kanban"} {
		if !strings.Contains(query, fragment) {
			t.Errorf("set-wise Kanban insertion missing %q", fragment)
		}
	}
}

func TestDocumentSummaryQueryIsTenantScopedAndBatched(t *testing.T) {
	query := strings.ToLower(documentSummarySelect)
	for _, fragment := range []string{
		"from delivery_notes dn",
		"join delivery_note_lines dnl on dnl.tenant_id=dn.tenant_id and dnl.delivery_note_id=dn.id and dnl.purchase_order_id=dn.purchase_order_id",
		"join kanban_lots kl on kl.tenant_id=dnl.tenant_id and kl.delivery_note_line_id=dnl.id and kl.purchase_order_line_id=dnl.purchase_order_line_id",
		"where dn.tenant_id=$1",
		"dn.purchase_order_id=any($2::uuid[])",
		"count(kl.id)",
		"group by",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("document summary query missing %q", fragment)
		}
	}
}

func TestLoadDocumentSummariesUsesOneQueryForAllOrders(t *testing.T) {
	tenantID := uuid.New()
	firstOrderID := uuid.New()
	secondOrderID := uuid.New()
	firstDeliveryNoteID := uuid.New()
	secondDeliveryNoteID := uuid.New()
	issuedAt := time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC)
	tx := &documentSummaryQueryRecorder{rows: &documentSummaryRows{records: []documentSummaryRecord{
		{firstOrderID, firstDeliveryNoteID, "DN-202607-00001", 2, issuedAt},
		{secondOrderID, secondDeliveryNoteID, "DN-202607-00002", 3, issuedAt},
	}}}

	summaries, err := loadDocumentSummaries(context.Background(), tx, tenantID, []uuid.UUID{firstOrderID, secondOrderID})
	if err != nil {
		t.Fatalf("load summaries: %v", err)
	}
	if tx.queryCount != 1 {
		t.Fatalf("document summary queries = %d, want 1", tx.queryCount)
	}
	if len(tx.args) != 2 || tx.args[0] != tenantID {
		t.Fatalf("query args = %#v, want tenant ID and order IDs", tx.args)
	}
	orderIDs, ok := tx.args[1].([]uuid.UUID)
	if !ok || len(orderIDs) != 2 || orderIDs[0] != firstOrderID || orderIDs[1] != secondOrderID {
		t.Fatalf("order ID batch = %#v, want both listed order IDs", tx.args[1])
	}
	if got := summaries[firstOrderID]; got.DeliveryNoteID != firstDeliveryNoteID || got.DeliveryNoteNumber != "DN-202607-00001" || got.KanbanCount != 2 || !got.IssuedAt.Equal(issuedAt) {
		t.Fatalf("first summary = %#v", got)
	}
	if got := summaries[secondOrderID]; got.DeliveryNoteID != secondDeliveryNoteID || got.KanbanCount != 3 {
		t.Fatalf("second summary = %#v", got)
	}
}

func TestAttachDocumentSummaryRequiresApprovedStatus(t *testing.T) {
	approvedID := uuid.New()
	draftID := uuid.New()
	summary := DocumentSummary{DeliveryNoteID: uuid.New(), DeliveryNoteNumber: "DN-202607-00001", KanbanCount: 2}
	summaries := map[uuid.UUID]DocumentSummary{approvedID: summary, draftID: summary}
	approved := Order{ID: approvedID, Status: StatusApproved}
	draft := Order{ID: draftID, Status: StatusDraft}

	attachDocumentSummary(&approved, summaries)
	attachDocumentSummary(&draft, summaries)

	if approved.Documents == nil || approved.Documents.DeliveryNoteID != summary.DeliveryNoteID {
		t.Fatalf("approved documents = %#v, want summary", approved.Documents)
	}
	if draft.Documents != nil {
		t.Fatalf("draft documents = %#v, want nil", draft.Documents)
	}
}

type documentSummaryQueryRecorder struct {
	queryCount int
	query      string
	args       []any
	rows       pgx.Rows
}

func (r *documentSummaryQueryRecorder) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	r.queryCount++
	r.query = query
	r.args = args
	return r.rows, nil
}

type documentSummaryRecord struct {
	orderID            uuid.UUID
	deliveryNoteID     uuid.UUID
	deliveryNoteNumber string
	kanbanCount        int64
	issuedAt           time.Time
}

type documentSummaryRows struct {
	records []documentSummaryRecord
	index   int
	closed  bool
}

func (r *documentSummaryRows) Close()                                       { r.closed = true }
func (r *documentSummaryRows) Err() error                                   { return nil }
func (r *documentSummaryRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *documentSummaryRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *documentSummaryRows) Values() ([]any, error)                       { return nil, fmt.Errorf("not implemented") }
func (r *documentSummaryRows) RawValues() [][]byte                          { return nil }
func (r *documentSummaryRows) Conn() *pgx.Conn                              { return nil }

func (r *documentSummaryRows) Next() bool {
	if r.index >= len(r.records) {
		r.Close()
		return false
	}
	r.index++
	return true
}

func (r *documentSummaryRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.records) {
		return fmt.Errorf("Scan called without a current row")
	}
	if len(dest) != 5 {
		return fmt.Errorf("Scan destinations = %d, want 5", len(dest))
	}
	record := r.records[r.index-1]
	*dest[0].(*uuid.UUID) = record.orderID
	*dest[1].(*uuid.UUID) = record.deliveryNoteID
	*dest[2].(*string) = record.deliveryNoteNumber
	*dest[3].(*int64) = record.kanbanCount
	*dest[4].(*time.Time) = record.issuedAt
	return nil
}
