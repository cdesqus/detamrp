package purchaseorder

import (
	"context"
	"encoding/json"
	"errors"
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
}

func (r *httpRepository) ListOrders(context.Context, Actor, ListQuery) ([]Order, int, error) {
	r.called = "list"
	return r.orders, r.total, r.err
}
func (r *httpRepository) GetOrder(context.Context, Actor, uuid.UUID) (Order, error) {
	r.called = "get"
	return r.order, r.err
}
func (r *httpRepository) CreateOrder(context.Context, Actor, OrderInput) (Order, error) {
	r.called = "create"
	return r.order, r.err
}
func (r *httpRepository) UpdateOrder(context.Context, Actor, uuid.UUID, OrderInput) (Order, error) {
	r.called = "update"
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
	return `{"supplierId":"` + uuid.New().String() + `","orderDate":"` + time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC).Format(time.RFC3339) + `","expectedDeliveryDate":"2026-07-22T00:00:00Z","currency":"IDR","lines":[{"rawMaterialId":"` + uuid.New().String() + `","totalKanban":"` + decimal.NewFromInt(1).String() + `"}]}`
}
