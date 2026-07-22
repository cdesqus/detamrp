package purchaseorder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/auth"
)

type httpAuthenticator struct {
	user auth.User
	err  error
}

func (a httpAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return a.user, a.err
}

type httpRepository struct {
	order     Order
	approval  Approval
	orders    []Order
	approvals []Approval
	total     int
	err       error
	called    string
	create    OrderInput
	update    OrderInput
}

func (r *httpRepository) ListOrders(context.Context, Actor, ListQuery) ([]Order, int, error) {
	r.called = "list"
	return r.orders, r.total, r.err
}
func (r *httpRepository) GetOrder(context.Context, Actor, uuid.UUID) (Order, error) {
	r.called = "get"
	return r.order, r.err
}
func (r *httpRepository) CreateOrder(_ context.Context, _ Actor, input OrderInput) (Order, error) {
	r.called = "create"
	r.create = input
	return r.order, r.err
}
func (r *httpRepository) UpdateOrder(_ context.Context, _ Actor, _ uuid.UUID, input OrderInput) (Order, error) {
	r.called = "update"
	r.update = input
	return r.order, r.err
}
func (r *httpRepository) SubmitOrder(context.Context, Actor, uuid.UUID) (Order, error) {
	r.called = "submit"
	return r.order, r.err
}
func (r *httpRepository) CancelOrder(context.Context, Actor, uuid.UUID) (Order, error) {
	r.called = "cancel"
	return r.order, r.err
}
func (r *httpRepository) ListApprovals(context.Context, Actor, ListQuery) ([]Approval, int, error) {
	r.called = "approvals"
	return r.approvals, r.total, r.err
}
func (r *httpRepository) Approve(context.Context, Actor, uuid.UUID, DecisionInput) (Approval, error) {
	r.called = "approve"
	return r.approval, r.err
}
func (r *httpRepository) Reject(context.Context, Actor, uuid.UUID, DecisionInput) (Approval, error) {
	r.called = "reject"
	return r.approval, r.err
}

func TestPurchaseOrderRoutesRequireAuthentication(t *testing.T) {
	router := purchaseOrderRouter(t, nil, &httpRepository{})
	recorder := serve(router, http.MethodGet, "/purchase-orders", "", "")

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestPurchaseOrderRoutesEnforcePermissions(t *testing.T) {
	id := uuid.New()
	for _, test := range []struct {
		name       string
		method     string
		path       string
		body       string
		permission string
	}{
		{"list", http.MethodGet, "/purchase-orders", "", "po.view"},
		{"get", http.MethodGet, "/purchase-orders/" + id.String(), "", "po.view"},
		{"create", http.MethodPost, "/purchase-orders", validOrderJSON(), "po.create"},
		{"update", http.MethodPut, "/purchase-orders/" + id.String(), validOrderJSON(), "po.edit_draft"},
		{"submit", http.MethodPost, "/purchase-orders/" + id.String() + "/submit", "", "po.submit"},
		{"cancel", http.MethodPost, "/purchase-orders/" + id.String() + "/cancel", "", "po.edit_draft"},
		{"list approvals", http.MethodGet, "/purchase-order-approvals", "", "po.approve"},
		{"approve", http.MethodPost, "/purchase-order-approvals/" + id.String() + "/approve", `{}`, "po.approve"},
		{"reject", http.MethodPost, "/purchase-order-approvals/" + id.String() + "/reject", `{"reason":"budget"}`, "po.reject"},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := purchaseOrderRouter(t, []string{"unrelated.permission"}, &httpRepository{})
			recorder := serve(router, test.method, test.path, test.body, "session")
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d for %s", recorder.Code, http.StatusForbidden, test.permission)
			}

			allowedRouter := purchaseOrderRouter(t, []string{test.permission}, &httpRepository{})
			allowed := serve(allowedRouter, test.method, test.path, test.body, "session")
			if allowed.Code == http.StatusForbidden {
				t.Fatalf("%s permission did not grant access", test.permission)
			}
		})
	}
}

func TestPurchaseOrderRoutesRejectInvalidJSONAndUUIDs(t *testing.T) {
	router := purchaseOrderRouter(t, []string{"po.create", "po.view", "po.approve"}, &httpRepository{})
	for _, test := range []struct {
		name string
		path string
		body string
	}{
		{"invalid JSON", "/purchase-orders", `{`},
		{"invalid order ID", "/purchase-orders/not-a-uuid", ""},
		{"invalid approval ID", "/purchase-order-approvals/not-a-uuid/approve", `{}`},
		{"invalid supplier filter", "/purchase-orders?supplierId=not-a-uuid", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			method := http.MethodGet
			if test.name == "invalid JSON" {
				method = http.MethodPost
			}
			if test.name == "invalid approval ID" {
				method = http.MethodPost
			}
			recorder := serve(router, method, test.path, test.body, "session")
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
		})
	}
}

func TestPurchaseOrderRoutesRejectUnsupportedStatusFilter(t *testing.T) {
	router := purchaseOrderRouter(t, []string{"po.view"}, &httpRepository{})
	recorder := serve(router, http.MethodGet, "/purchase-orders?status=WAITING_FOR_VP", "", "session")

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"error":"invalid_filter"`) || !strings.Contains(recorder.Body.String(), `"status"`) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestPurchaseOrderRoutesAcceptDateOnlyOrderDates(t *testing.T) {
	body := `{"supplierId":"` + uuid.New().String() + `","orderDate":"2026-07-21","expectedDeliveryDate":"2026-07-22","currency":"IDR","lines":[{"rawMaterialId":"` + uuid.New().String() + `","totalKanban":"1"}]}`
	for _, test := range []struct {
		name       string
		method     string
		path       string
		permission string
		status     int
	}{
		{"create", http.MethodPost, "/purchase-orders", "po.create", http.StatusCreated},
		{"update", http.MethodPut, "/purchase-orders/" + uuid.New().String(), "po.edit_draft", http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &httpRepository{order: Order{Status: StatusDraft}}
			router := purchaseOrderRouter(t, []string{test.permission}, repository)
			recorder := serve(router, test.method, test.path, body, "session")
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, test.status, recorder.Body.String())
			}
			input := repository.create
			if test.name == "update" {
				input = repository.update
			}
			if got := input.OrderDate; !got.Equal(time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC)) || got.Location() != time.UTC {
				t.Fatalf("order date = %s (%s), want normalized UTC date", got, got.Location())
			}
			if got := input.ExpectedDeliveryDate; !got.Equal(time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC)) || got.Location() != time.UTC {
				t.Fatalf("expected delivery date = %s (%s), want normalized UTC date", got, got.Location())
			}
		})
	}
}

func TestPurchaseOrderRoutesRejectRFC3339OrderDates(t *testing.T) {
	router := purchaseOrderRouter(t, []string{"po.create"}, &httpRepository{})
	body := `{"supplierId":"` + uuid.New().String() + `","orderDate":"2026-07-21T23:30:00-05:00","expectedDeliveryDate":"2026-07-22","currency":"IDR"}`
	recorder := serve(router, http.MethodPost, "/purchase-orders", body, "session")

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"orderDate"`) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestPurchaseOrderGetResponsesRedactPricesWithoutPricePermission(t *testing.T) {
	line := OrderLine{RawMaterialID: uuid.New(), UnitPriceSnapshot: decimal.NewFromInt(7), LineTotal: decimal.NewFromInt(21)}
	order := Order{ID: uuid.New(), SupplierName: "Acme", TotalAmount: decimal.NewFromInt(21), Lines: []OrderLine{line}}
	for _, test := range []struct {
		name string
		path string
	}{
		{name: "list", path: "/purchase-orders"},
		{name: "detail", path: "/purchase-orders/" + order.ID.String()},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &httpRepository{order: order, orders: []Order{order}, total: 1}
			viewer := serve(purchaseOrderRouter(t, []string{"po.view"}, repository), http.MethodGet, test.path, "", "session")
			if viewer.Code != http.StatusOK {
				t.Fatalf("viewer status = %d: %s", viewer.Code, viewer.Body.String())
			}
			for _, forbidden := range []string{`"totalAmount"`, `"unitPriceSnapshot"`, `"lineTotal"`} {
				if strings.Contains(viewer.Body.String(), forbidden) {
					t.Errorf("viewer response exposed %s: %s", forbidden, viewer.Body.String())
				}
			}

			authorized := serve(purchaseOrderRouter(t, []string{"po.view", "po.price.view"}, repository), http.MethodGet, test.path, "", "session")
			for _, required := range []string{`"totalAmount"`, `"unitPriceSnapshot"`, `"lineTotal"`} {
				if !strings.Contains(authorized.Body.String(), required) {
					t.Errorf("authorized response omitted %s: %s", required, authorized.Body.String())
				}
			}
		})
	}
}

func TestPurchaseOrderMutationResponsesRedactPricesWithoutPricePermission(t *testing.T) {
	line := OrderLine{RawMaterialID: uuid.New(), TotalKanban: decimal.NewFromInt(1), UnitPriceSnapshot: decimal.NewFromInt(7), LineTotal: decimal.NewFromInt(21)}
	order := Order{ID: uuid.New(), Status: StatusDraft, TotalAmount: decimal.NewFromInt(21), Lines: []OrderLine{line}}
	for _, test := range []struct {
		name       string
		method     string
		path       string
		body       string
		permission string
	}{
		{name: "create", method: http.MethodPost, path: "/purchase-orders", body: validOrderJSON(), permission: "po.create"},
		{name: "update", method: http.MethodPut, path: "/purchase-orders/" + order.ID.String(), body: validOrderJSON(), permission: "po.edit_draft"},
		{name: "submit", method: http.MethodPost, path: "/purchase-orders/" + order.ID.String() + "/submit", permission: "po.submit"},
		{name: "cancel", method: http.MethodPost, path: "/purchase-orders/" + order.ID.String() + "/cancel", permission: "po.edit_draft"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &httpRepository{order: order}
			redacted := serve(purchaseOrderRouter(t, []string{test.permission}, repository), test.method, test.path, test.body, "session")
			if redacted.Code < 200 || redacted.Code >= 300 {
				t.Fatalf("redacted status = %d: %s", redacted.Code, redacted.Body.String())
			}
			for _, forbidden := range []string{`"totalAmount"`, `"unitPriceSnapshot"`, `"lineTotal"`} {
				if strings.Contains(redacted.Body.String(), forbidden) {
					t.Errorf("response without price permission exposed %s: %s", forbidden, redacted.Body.String())
				}
			}

			authorized := serve(purchaseOrderRouter(t, []string{test.permission, "po.price.view"}, repository), test.method, test.path, test.body, "session")
			for _, required := range []string{`"totalAmount"`, `"unitPriceSnapshot"`, `"lineTotal"`} {
				if !strings.Contains(authorized.Body.String(), required) {
					t.Errorf("authorized response omitted %s: %s", required, authorized.Body.String())
				}
			}
		})
	}
}

func TestPurchaseOrderRoutesReturnFieldErrorsForInvalidDates(t *testing.T) {
	router := purchaseOrderRouter(t, []string{"po.create"}, &httpRepository{})
	body := `{"supplierId":"` + uuid.New().String() + `","orderDate":"2026-02-30","expectedDeliveryDate":"not-a-date","currency":"IDR"}`
	recorder := serve(router, http.MethodPost, "/purchase-orders", body, "session")

	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"fields"`) || !strings.Contains(recorder.Body.String(), `"orderDate"`) || !strings.Contains(recorder.Body.String(), `"expectedDeliveryDate"`) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestPurchaseOrderResponseModelsUseLowerCamelJSON(t *testing.T) {
	actor := Actor{TenantID: uuid.New(), UserID: uuid.New(), DisplayName: "Buyer", Email: "buyer@example.test"}
	line := OrderLine{ID: uuid.New(), TenantID: uuid.New(), PurchaseOrderID: uuid.New(), RawMaterialID: uuid.New(), RawMaterialCode: "RM-1", RawMaterialName: "Resin", BaseUnitID: uuid.New(), BaseUnitCode: "KG", QtyPerKanbanSnapshot: decimal.NewFromInt(2), TotalKanban: decimal.NewFromInt(3), OrderedBaseQty: decimal.NewFromInt(6), UnitPriceSnapshot: decimal.NewFromInt(4), LineTotal: decimal.NewFromInt(24), SortPosition: 1, CreatedBy: actor, CreatedAt: time.Now(), UpdatedBy: actor, UpdatedAt: time.Now()}
	order := Order{ID: uuid.New(), TenantID: uuid.New(), PONumber: "PO-202607-00001", SupplierID: uuid.New(), SupplierName: "Acme", OrderDate: time.Now(), ExpectedDeliveryDate: time.Now(), Currency: "IDR", Notes: "notes", Status: StatusDraft, Version: 1, TotalAmount: decimal.NewFromInt(24), SagePurchaseOrderNumber: "SAGE-1", SubmittedApproverUserID: uuid.New(), SubmittedApproverDisplayName: "Director", SubmittedApproverEmail: "director@example.test", CreatedBy: actor, CreatedAt: time.Now(), UpdatedBy: actor, UpdatedAt: time.Now(), Lines: []OrderLine{line}}
	approval := Approval{ID: uuid.New(), TenantID: uuid.New(), PurchaseOrderID: order.ID, PONumber: order.PONumber, SupplierID: order.SupplierID, SupplierName: "Acme", Version: 1, ApproverUserID: uuid.New(), ApproverDisplayName: "Director", ApproverEmail: "director@example.test", Status: ApprovalPending, DecisionReason: "", DecidedByUserID: uuid.New(), CreatedBy: actor, CreatedAt: time.Now(), UpdatedBy: actor, UpdatedAt: time.Now()}

	assertJSONKeys(t, order, []string{"id", "tenantId", "poNumber", "supplierId", "supplierName", "orderDate", "expectedDeliveryDate", "currency", "notes", "status", "version", "totalAmount", "sagePurchaseOrderNumber", "submittedApproverUserId", "submittedApproverDisplayName", "submittedApproverEmail", "createdBy", "createdAt", "updatedBy", "updatedAt", "lines", "documents"})
	assertJSONKeys(t, line, []string{"id", "tenantId", "purchaseOrderId", "rawMaterialId", "rawMaterialCode", "rawMaterialName", "baseUnitId", "baseUnitCode", "qtyPerKanbanSnapshot", "totalKanban", "orderedBaseQty", "unitPriceSnapshot", "lineTotal", "sortPosition", "createdBy", "createdAt", "updatedBy", "updatedAt"})
	assertJSONKeys(t, approval, []string{"id", "tenantId", "purchaseOrderId", "poNumber", "supplierId", "supplierName", "version", "approverUserId", "approverDisplayName", "approverEmail", "status", "decisionReason", "decidedAt", "decidedByUserId", "createdBy", "createdAt", "updatedBy", "updatedAt"})
	assertJSONKeys(t, actor, []string{"tenantId", "userId", "displayName", "email"})
}

func TestDomainJSONContract(t *testing.T) {
	deliveryNoteID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	issuedAt := time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC)
	approved := Order{
		Status: StatusApproved,
		Documents: &DocumentSummary{
			DeliveryNoteID:     deliveryNoteID,
			DeliveryNoteNumber: "DN-202607-00001",
			KanbanCount:        10,
			IssuedAt:           issuedAt,
		},
	}

	encoded, err := json.Marshal(approved)
	if err != nil {
		t.Fatalf("marshal approved order: %v", err)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatalf("decode approved order: %v", err)
	}
	want := `{"deliveryNoteId":"10000000-0000-0000-0000-000000000001","deliveryNoteNumber":"DN-202607-00001","kanbanCount":10,"issuedAt":"2026-07-22T00:00:00Z"}`
	if got := string(response["documents"]); got != want {
		t.Fatalf("approved documents = %s, want %s", got, want)
	}

	encoded, err = json.Marshal(Order{Status: StatusDraft})
	if err != nil {
		t.Fatalf("marshal draft order: %v", err)
	}
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatalf("decode draft order: %v", err)
	}
	if got := string(response["documents"]); got != "null" {
		t.Fatalf("draft documents = %s, want null", got)
	}
}

func TestPurchaseOrderDocumentSummaryResponses(t *testing.T) {
	order := Order{
		ID:     uuid.New(),
		Status: StatusApproved,
		Documents: &DocumentSummary{
			DeliveryNoteID:     uuid.MustParse("10000000-0000-0000-0000-000000000001"),
			DeliveryNoteNumber: "DN-202607-00001",
			KanbanCount:        10,
			IssuedAt:           time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC),
		},
	}
	repository := &httpRepository{order: order, orders: []Order{order}, total: 1}
	router := purchaseOrderRouter(t, []string{"po.view"}, repository)

	for _, path := range []string{"/purchase-orders", "/purchase-orders/" + order.ID.String()} {
		response := serve(router, http.MethodGet, path, "", "session")
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d: %s", path, response.Code, response.Body.String())
		}
		for _, fragment := range []string{
			`"documents":{"deliveryNoteId":"10000000-0000-0000-0000-000000000001"`,
			`"deliveryNoteNumber":"DN-202607-00001"`,
			`"kanbanCount":10`,
			`"issuedAt":"2026-07-22T00:00:00Z"`,
		} {
			if !strings.Contains(response.Body.String(), fragment) {
				t.Errorf("GET %s omitted %s: %s", path, fragment, response.Body.String())
			}
		}
	}

	draft := Order{ID: uuid.New(), Status: StatusDraft, Documents: order.Documents}
	draftResponse := serve(purchaseOrderRouter(t, []string{"po.view"}, &httpRepository{order: draft}), http.MethodGet, "/purchase-orders/"+draft.ID.String(), "", "session")
	if !strings.Contains(draftResponse.Body.String(), `"documents":null`) {
		t.Fatalf("draft response documents were not null: %s", draftResponse.Body.String())
	}
}

func TestPurchaseOrderRoutesReturnExpectedSuccessCodes(t *testing.T) {
	id := uuid.New()
	repository := &httpRepository{
		order:     Order{ID: id, Status: StatusDraft, Lines: []OrderLine{{RawMaterialID: uuid.New(), TotalKanban: decimal.NewFromInt(1)}}},
		orders:    []Order{{ID: id, Status: StatusDraft}},
		approval:  Approval{ID: id, Status: ApprovalPending},
		approvals: []Approval{{ID: id, Status: ApprovalPending}},
		total:     1,
	}
	permissions := []string{"po.view", "po.create", "po.edit_draft", "po.submit", "po.approve", "po.reject"}
	router := purchaseOrderRouter(t, permissions, repository)
	for _, test := range []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{"list", http.MethodGet, "/purchase-orders", "", http.StatusOK},
		{"get", http.MethodGet, "/purchase-orders/" + id.String(), "", http.StatusOK},
		{"create", http.MethodPost, "/purchase-orders", validOrderJSON(), http.StatusCreated},
		{"update", http.MethodPut, "/purchase-orders/" + id.String(), validOrderJSON(), http.StatusOK},
		{"submit", http.MethodPost, "/purchase-orders/" + id.String() + "/submit", "", http.StatusOK},
		{"cancel", http.MethodPost, "/purchase-orders/" + id.String() + "/cancel", "", http.StatusOK},
		{"list approvals", http.MethodGet, "/purchase-order-approvals", "", http.StatusOK},
		{"approve", http.MethodPost, "/purchase-order-approvals/" + id.String() + "/approve", `{}`, http.StatusOK},
		{"reject", http.MethodPost, "/purchase-order-approvals/" + id.String() + "/reject", `{"reason":"budget"}`, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := serve(router, test.method, test.path, test.body, "session")
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, test.status, recorder.Body.String())
			}
		})
	}
}

func TestPurchaseOrderRoutesWriteStructuredServiceErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		err        error
		wantStatus int
		wantError  string
	}{
		{"validation", ValidationError{Fields: FieldErrors{"supplierId": "Supplier is required"}}, http.StatusBadRequest, "validation_failed"},
		{"conflict", ConflictError{Fields: FieldErrors{"status": "Only draft purchase orders can be submitted"}}, http.StatusConflict, "conflict"},
		{"capacity", CapacityError{Field: "documents", Message: "Monthly Kanban label capacity is exhausted"}, http.StatusConflict, "capacity_exceeded"},
		{"not found", NotFoundError{Resource: "purchase order"}, http.StatusNotFound, "not_found"},
		{"internal", errors.New("database unavailable"), http.StatusInternalServerError, "internal_error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &httpRepository{err: test.err}
			router := purchaseOrderRouter(t, []string{"po.create"}, repository)
			recorder := serve(router, http.MethodPost, "/purchase-orders", validOrderJSON(), "session")
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["error"] != test.wantError {
				t.Fatalf("error = %#v, want %q", body["error"], test.wantError)
			}
			if test.wantError == "validation_failed" && body["fields"] == nil {
				t.Fatal("validation response did not include fields")
			}
		})
	}
}

func TestPurchaseOrderRoutesLogUnexpectedErrorsWithRequestContext(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	actor := auth.User{ID: uuid.New(), TenantID: uuid.New(), DisplayName: "Buyer", Permissions: []string{"po.view", "po.approve"}}
	purchaseOrderID, approvalID := uuid.New(), uuid.New()

	for _, request := range []struct {
		method, path string
		err          error
		contextKeys  []string
	}{
		{http.MethodGet, "/purchase-orders/" + purchaseOrderID.String(), errors.New("database unavailable"), []string{"purchase_order_id"}},
		{http.MethodPost, "/purchase-order-approvals/" + approvalID.String() + "/approve", ApprovalDocumentError{ApprovalID: approvalID, PurchaseOrderID: purchaseOrderID, Err: errors.New("generator unavailable")}, []string{"approval_id", "purchase_order_id"}},
		{http.MethodPost, "/purchase-order-approvals/" + approvalID.String() + "/approve", ApprovalDocumentError{ApprovalID: approvalID, PurchaseOrderID: purchaseOrderID, Err: CapacityError{Field: "documents", Message: "Monthly Kanban label capacity is exhausted"}}, []string{"approval_id", "purchase_order_id"}},
	} {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		RegisterRoutes(router, NewService(&httpRepository{err: request.err}), httpAuthenticator{user: actor})
		response := serve(router, request.method, request.path, `{}`, "session")
		if response.Code != http.StatusInternalServerError && response.Code != http.StatusConflict {
			t.Fatalf("unexpected sanitized response: %d %s", response.Code, response.Body.String())
		}
		for _, contextKey := range request.contextKeys {
			if !strings.Contains(logs.String(), contextKey) {
				t.Fatalf("structured log missing %s: %s", contextKey, logs.String())
			}
		}
		if !strings.Contains(logs.String(), actor.TenantID.String()) {
			t.Fatalf("structured log missing request context: %s", logs.String())
		}
		logs.Reset()
	}
}

func TestPurchaseOrderRoutesValidateRejectionReason(t *testing.T) {
	router := purchaseOrderRouter(t, []string{"po.reject"}, &httpRepository{})
	recorder := serve(router, http.MethodPost, "/purchase-order-approvals/"+uuid.New().String()+"/reject", `{"reason":" "}`, "session")
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "reason") {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func purchaseOrderRouter(t *testing.T, permissions []string, repository *httpRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	user := auth.User{ID: uuid.New(), TenantID: uuid.New(), DisplayName: "Buyer", Permissions: permissions}
	RegisterRoutes(router, NewService(repository), httpAuthenticator{user: user})
	return router
}

func serve(router http.Handler, method, path, body, session string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if session != "" {
		request.AddCookie(&http.Cookie{Name: "session", Value: session})
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func validOrderJSON() string {
	return `{"supplierId":"` + uuid.New().String() + `","orderDate":"2026-07-21","expectedDeliveryDate":"2026-07-22","currency":"IDR","lines":[{"rawMaterialId":"` + uuid.New().String() + `","totalKanban":"` + decimal.NewFromInt(1).String() + `"}]}`
}

func assertJSONKeys(t *testing.T, value any, keys []string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal response model: %v", err)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatalf("decode response model: %v", err)
	}
	for _, key := range keys {
		if _, ok := response[key]; !ok {
			t.Errorf("response missing JSON key %q: %s", key, encoded)
		}
	}
	for key := range response {
		if key != "" && key[0] >= 'A' && key[0] <= 'Z' {
			t.Errorf("response exposed non-camel JSON key %q: %s", key, encoded)
		}
	}
}
