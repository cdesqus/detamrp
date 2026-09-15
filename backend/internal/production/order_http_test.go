package production

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http/httptest"
	"strings"
	"testing"
)

type orderHTTPRepo struct{ OrderRepository }

func (orderHTTPRepo) CreateOrder(_ context.Context, _ Actor, i OrderInput) (Order, error) {
	return Order{OrderNumber: "PRO-0000001", PlannedQty: i.PlannedQty, Status: OrderReleased}, nil
}
func (orderHTTPRepo) OrderAction(context.Context, Actor, uuid.UUID, string, string) (Order, error) {
	return Order{}, invalid("Cannot change an order with entries")
}
func TestOrderHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		method, path, body string
		permissions        []string
		want               int
	}{
		{"GET", "/production-orders", "", nil, 403},
		{"POST", "/production-orders", `{}`, []string{"production.order"}, 422},
		{"POST", "/production-orders", `{"planLineId":"11111111-1111-1111-1111-111111111111","plannedQty":"12","dueDate":"2026-09-30"}`, []string{"production.order"}, 201},
		{"POST", "/production-orders/11111111-1111-1111-1111-111111111111/cancel", `{"reason":"test"}`, []string{"production.order"}, 422},
	} {
		r := gin.New()
		RegisterOrderRoutes(r, orderHTTPRepo{}, testAuth{tc.permissions})
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Cookie", "session=test")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
}
