package production

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type wipHTTPRepo struct{ WIPRepository }

func (wipHTTPRepo) ListWIP(context.Context, Actor) ([]WIPOrderBalance, error) {
	return []WIPOrderBalance{{
		OrderNumber: "PRO-0000001", Currency: "IDR", TotalQty: decimal.NewFromInt(30), TotalValue: decimal.NewFromInt(30000), Reconciled: true,
		Operations: []WIPOperationBalance{{Code: "STAMPING", OnHand: decimal.NewFromInt(30), Value: decimal.NewFromInt(30000)}},
	}}, nil
}
func (wipHTTPRepo) CreateTransfer(_ context.Context, _ Actor, i TransferInput) (WIPOrderBalance, error) {
	if i.Quantity.GreaterThan(decimal.NewFromInt(30)) {
		return WIPOrderBalance{}, invalid("Only 30 is available in this WIP balance")
	}
	return WIPOrderBalance{OrderNumber: "PRO-0000001", TotalQty: decimal.NewFromInt(30)}, nil
}
func (wipHTTPRepo) ReverseMovement(context.Context, Actor, uuid.UUID, string) (WIPOrderBalance, error) {
	return WIPOrderBalance{}, invalid("Only an untouched transfer can be reversed")
}

func TestWIPHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const transfer = `{"orderId":"11111111-1111-1111-1111-111111111111","sourceOperationId":"22222222-2222-2222-2222-222222222222","quantity":"10","movementDate":"2026-09-14"}`
	for _, tc := range []struct {
		name, method, path, body string
		permissions              []string
		want                     int
	}{
		{"anonymous viewers are refused", "GET", "/production-wip", "", nil, 403},
		{"viewers may read balances", "GET", "/production-wip", "", []string{"production.view"}, 200},
		{"viewers may not move WIP", "POST", "/production-wip/transfers", transfer, []string{"production.view"}, 403},
		{"transfers need a quantity", "POST", "/production-wip/transfers", `{"orderId":"11111111-1111-1111-1111-111111111111","sourceOperationId":"22222222-2222-2222-2222-222222222222","movementDate":"2026-09-14"}`, []string{"production.wip"}, 422},
		{"transfers cannot exceed the balance", "POST", "/production-wip/transfers", `{"orderId":"11111111-1111-1111-1111-111111111111","sourceOperationId":"22222222-2222-2222-2222-222222222222","quantity":"31","movementDate":"2026-09-14"}`, []string{"production.wip"}, 422},
		{"operators may transfer", "POST", "/production-wip/transfers", transfer, []string{"production.wip"}, 201},
		{"consumed transfers cannot be reversed", "POST", "/production-wip/movements/11111111-1111-1111-1111-111111111111/reverse", `{"reason":"typo"}`, []string{"production.wip"}, 422},
		{"identifiers are validated", "GET", "/production-wip/not-a-uuid", "", []string{"production.view"}, 400},
	} {
		router := gin.New()
		RegisterWIPRoutes(router, wipHTTPRepo{}, testAuth{tc.permissions})
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		request.Header.Set("Cookie", "session=test")
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != tc.want {
			t.Fatalf("%s: %s %s returned %d %s", tc.name, tc.method, tc.path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestWIPHidesCostsFromOperators(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		permissions []string
		wantValue   bool
	}{
		{[]string{"production.view"}, false},
		{[]string{"production.view", "production.report"}, true},
	} {
		router := gin.New()
		RegisterWIPRoutes(router, wipHTTPRepo{}, testAuth{tc.permissions})
		request := httptest.NewRequest("GET", "/production-wip", nil)
		request.Header.Set("Cookie", "session=test")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		var payload struct {
			Items []map[string]any `json:"items"`
		}
		if e := json.Unmarshal(recorder.Body.Bytes(), &payload); e != nil {
			t.Fatal(e)
		}
		if len(payload.Items) != 1 {
			t.Fatalf("unexpected payload: %s", recorder.Body.String())
		}
		if _, ok := payload.Items[0]["totalValue"]; ok != tc.wantValue {
			t.Fatalf("value exposure for %v: %s", tc.permissions, recorder.Body.String())
		}
	}
}
