package purchaseorder

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeRepository struct {
	order        Order
	deliveryNote DeliveryNoteDocument
	kanbanLabels KanbanLabelDocument
	createInput  OrderInput
	updateInput  OrderInput
	listQuery    ListQuery
	decision     DecisionInput
	called       string
	err          error
}

func (r *fakeRepository) ListOrders(_ context.Context, _ Actor, q ListQuery) ([]Order, int, error) {
	r.called, r.listQuery = "list", q
	return nil, 0, r.err
}
func (r *fakeRepository) GetOrder(_ context.Context, _ Actor, _ uuid.UUID) (Order, error) {
	r.called = "get"
	return r.order, r.err
}
func (r *fakeRepository) LoadDeliveryNoteDocument(_ context.Context, _ Actor, _ uuid.UUID) (DeliveryNoteDocument, error) {
	r.called = "delivery-note"
	return r.deliveryNote, r.err
}
func (r *fakeRepository) LoadKanbanLabelDocument(_ context.Context, _ Actor, _ uuid.UUID) (KanbanLabelDocument, error) {
	r.called = "kanban-labels"
	return r.kanbanLabels, r.err
}
func (r *fakeRepository) CreateOrder(_ context.Context, _ Actor, in OrderInput) (Order, error) {
	r.called, r.createInput = "create", in
	return r.order, r.err
}
func (r *fakeRepository) UpdateOrder(_ context.Context, _ Actor, _ uuid.UUID, in OrderInput) (Order, error) {
	r.called, r.updateInput = "update", in
	return r.order, r.err
}
func (r *fakeRepository) SubmitOrder(_ context.Context, _ Actor, _ uuid.UUID) (Order, error) {
	r.called = "submit"
	return r.order, r.err
}
func (r *fakeRepository) CancelOrder(_ context.Context, _ Actor, _ uuid.UUID) (Order, error) {
	r.called = "cancel"
	return r.order, r.err
}
func (r *fakeRepository) ListApprovals(_ context.Context, _ Actor, q ListQuery) ([]Approval, int, error) {
	r.called, r.listQuery = "approvals", q
	return nil, 0, r.err
}
func (r *fakeRepository) Approve(_ context.Context, _ Actor, _ uuid.UUID, in DecisionInput) (Approval, error) {
	r.called, r.decision = "approve", in
	return Approval{}, r.err
}
func (r *fakeRepository) Reject(_ context.Context, _ Actor, _ uuid.UUID, in DecisionInput) (Approval, error) {
	r.called, r.decision = "reject", in
	return Approval{}, r.err
}

func TestServiceNormalizesBeforeCreatingDraft(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	input := validServiceOrderInput()
	input.Currency = " idr "
	input.Notes = " notes "

	if _, err := service.Create(context.Background(), serviceActor(), input); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repo.called != "create" || repo.createInput.Currency != "IDR" || repo.createInput.Notes != "notes" {
		t.Fatalf("repository input = %#v", repo.createInput)
	}
}

func TestServiceRendersPurchaseOrderPDFWithRequestedPriceVisibility(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepository{order: Order{ID: id, PONumber: "PO-1"}}
	service := NewService(repo)
	var gotOrder Order
	var gotIncludePrices bool
	service.renderPOPDF = func(order Order, includePrices bool) ([]byte, error) {
		gotOrder, gotIncludePrices = order, includePrices
		return []byte("%PDF-test"), nil
	}

	document, err := service.PurchaseOrderPDF(context.Background(), serviceActor(), id, true)
	if err != nil {
		t.Fatalf("PurchaseOrderPDF() error = %v", err)
	}
	if repo.called != "get" || gotOrder.ID != id || !gotIncludePrices {
		t.Fatalf("renderer received order=%s includePrices=%t after %q", gotOrder.ID, gotIncludePrices, repo.called)
	}
	if string(document.Content) != "%PDF-test" || document.Filename != "PO-1.pdf" {
		t.Fatalf("document = %#v", document)
	}
}

func TestServiceLoadsOperationalDocumentsBeforeRenderingPDF(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepository{
		deliveryNote: DeliveryNoteDocument{PurchaseOrderID: id, DeliveryNoteNumber: "DN-1"},
		kanbanLabels: KanbanLabelDocument{PurchaseOrderID: id, DeliveryNoteNumber: "DN-1"},
	}
	service := NewService(repo)
	service.renderDeliveryNotePDF = func(document DeliveryNoteDocument) ([]byte, error) {
		if document.PurchaseOrderID != id {
			t.Fatalf("delivery note PO = %s", document.PurchaseOrderID)
		}
		return []byte("delivery-note"), nil
	}
	service.renderKanbanLabelsPDF = func(_ context.Context, document KanbanLabelDocument) ([]byte, error) {
		if document.PurchaseOrderID != id {
			t.Fatalf("labels PO = %s", document.PurchaseOrderID)
		}
		return []byte("labels"), nil
	}

	deliveryNote, err := service.DeliveryNotePDF(context.Background(), serviceActor(), id)
	if err != nil || repo.called != "delivery-note" || deliveryNote.Filename != "DN-1.pdf" {
		t.Fatalf("DeliveryNotePDF() = %#v, %v after %q", deliveryNote, err, repo.called)
	}
	labels, err := service.KanbanLabelsPDF(context.Background(), serviceActor(), id)
	if err != nil || repo.called != "kanban-labels" || labels.Filename != "KANBAN-DN-1.pdf" {
		t.Fatalf("KanbanLabelsPDF() = %#v, %v after %q", labels, err, repo.called)
	}
}

func TestServiceRejectsOversizedKanbanLabelPDFBeforeRendering(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepository{kanbanLabels: KanbanLabelDocument{
		PurchaseOrderID: id,
		Labels:          make([]KanbanLabel, 1001),
	}}
	service := NewService(repo)
	renderCalls := 0
	service.renderKanbanLabelsPDF = func(context.Context, KanbanLabelDocument) ([]byte, error) {
		renderCalls++
		return []byte("unexpected"), nil
	}

	_, err := service.KanbanLabelsPDF(context.Background(), serviceActor(), id)
	var limit DocumentExportLimitError
	if !errors.As(err, &limit) || limit.Limit != 1000 || !strings.Contains(err.Error(), "1000") {
		t.Fatalf("KanbanLabelsPDF() error = %v, want safe 1000-label limit", err)
	}
	if renderCalls != 0 {
		t.Fatalf("oversized export rendered %d times, want none", renderCalls)
	}
}

func TestServiceDoesNotStartKanbanRenderingAfterCancellation(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepository{kanbanLabels: KanbanLabelDocument{
		PurchaseOrderID: id,
		Labels:          []KanbanLabel{{KanbanID: "KB-1"}},
	}}
	service := NewService(repo)
	renderCalls := 0
	service.renderKanbanLabelsPDF = func(context.Context, KanbanLabelDocument) ([]byte, error) {
		renderCalls++
		return []byte("unexpected"), nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.KanbanLabelsPDF(ctx, serviceActor(), id)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("KanbanLabelsPDF() error = %v, want context cancellation", err)
	}
	if renderCalls != 0 {
		t.Fatalf("cancelled export rendered %d times, want none", renderCalls)
	}
}

func TestServiceWrapsPDFRendererFailuresWithDocumentContext(t *testing.T) {
	id := uuid.New()
	cause := errors.New("render failed")
	repo := &fakeRepository{order: Order{ID: id, PONumber: "PO-1"}}
	service := NewService(repo)
	service.renderPOPDF = func(Order, bool) ([]byte, error) { return nil, cause }

	_, err := service.PurchaseOrderPDF(context.Background(), serviceActor(), id, false)
	var failure DocumentRenderError
	if !errors.As(err, &failure) || !errors.Is(err, cause) {
		t.Fatalf("PurchaseOrderPDF() error = %v, want wrapped renderer error", err)
	}
	if failure.PurchaseOrderID != id || failure.DocumentType != "purchase_order" {
		t.Fatalf("renderer failure context = %#v", failure)
	}
}

func TestServiceRejectsEditingNonDraftOrder(t *testing.T) {
	repo := &fakeRepository{order: Order{Status: StatusPendingApproval}}
	service := NewService(repo)

	_, err := service.Update(context.Background(), serviceActor(), uuid.New(), validServiceOrderInput())
	var conflict ConflictError
	if !errors.As(err, &conflict) || conflict.Fields["status"] == "" {
		t.Fatalf("Update() error = %v, want draft conflict", err)
	}
	if repo.called != "get" {
		t.Fatalf("repository call = %q, want get only", repo.called)
	}
}

func TestServiceRejectsSubmittingDraftWithoutMaterials(t *testing.T) {
	repo := &fakeRepository{order: Order{Status: StatusDraft}}
	service := NewService(repo)

	_, err := service.Submit(context.Background(), serviceActor(), uuid.New())
	var validation ValidationError
	if !errors.As(err, &validation) || validation.Fields["lines"] == "" {
		t.Fatalf("Submit() error = %v, want lines validation", err)
	}
	if repo.called != "get" {
		t.Fatalf("repository call = %q, want get only", repo.called)
	}
}

func TestServiceRequiresReasonWhenRejecting(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	_, err := service.Reject(context.Background(), serviceActor(), uuid.New(), DecisionInput{Reason: "  "})
	var validation ValidationError
	if !errors.As(err, &validation) || validation.Fields["reason"] == "" {
		t.Fatalf("Reject() error = %v, want reason validation", err)
	}
	if repo.called != "" {
		t.Fatalf("repository should not be called, got %q", repo.called)
	}
}

func TestServicePropagatesRepositoryErrors(t *testing.T) {
	want := errors.New("database unavailable")
	repo := &fakeRepository{err: want}
	service := NewService(repo)

	if _, err := service.Create(context.Background(), serviceActor(), validServiceOrderInput()); !errors.Is(err, want) {
		t.Fatalf("Create() error = %v, want %v", err, want)
	}
}

func TestServiceNormalizesListQueries(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	if _, _, err := service.List(context.Background(), serviceActor(), ListQuery{Search: " material ", Limit: 500, Offset: -1}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repo.listQuery.Search != "material" || repo.listQuery.Limit != 200 || repo.listQuery.Offset != 0 {
		t.Fatalf("list query = %#v", repo.listQuery)
	}
}

func serviceActor() Actor {
	return Actor{TenantID: uuid.New(), UserID: uuid.New()}
}

func validServiceOrderInput() OrderInput {
	return OrderInput{
		SupplierID:           uuid.New(),
		OrderDate:            time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC),
		ExpectedDeliveryDate: time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC),
		Currency:             "IDR",
		Lines:                []LineInput{{RawMaterialID: uuid.New(), TotalKanban: decimal.NewFromInt(1)}},
	}
}
