package production

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http/httptest"
	"order-stock/backend/internal/rbac"
	"strings"
	"testing"
)

type entryHTTPRepo struct{ EntryRepository }

func (entryHTTPRepo) CreateEntry(_ context.Context, _ Actor, i EntryInput) (Entry, error) {
	return Entry{EntryNumber: "DP-0000001", EntryInput: i, Status: "POSTED"}, nil
}

func TestDailyCostPayloadPermissions(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	payload := productionEntryPayload(c, Entry{EntryNumber: "DP-1"})
	raw, _ := json.Marshal(payload)
	if strings.Contains(string(raw), "materialCost") || strings.Contains(string(raw), "processRate") {
		t.Fatal("cost fields exposed without permission", string(raw))
	}
	c.Set(rbac.ContextPermissionsKey, []string{"production.report"})
	raw, _ = json.Marshal(productionEntryPayload(c, Entry{EntryNumber: "DP-1"}))
	if !strings.Contains(string(raw), "materialCost") {
		t.Fatal("cost hidden from report user")
	}
}

func (orderHTTPRepo) ListOrders(context.Context, Actor) ([]Order, error) { return []Order{{OrderNumber:"PRO-1"}},nil }

func TestProductionViewCannotReadOrderOrRoutingMoney(t *testing.T) {
	for _, path := range []string{"/production-orders","/production-routings"} {
		r:=gin.New()
		RegisterOrderRoutes(r,orderHTTPRepo{},testAuth{[]string{"production.view"}})
		RegisterRoutingRoutes(r,routingHTTPRepo{},testAuth{[]string{"production.view"}})
		req:=httptest.NewRequest("GET",path,nil);req.Header.Set("Cookie","session=test")
		w:=httptest.NewRecorder();r.ServeHTTP(w,req)
		if w.Code!=200 {t.Fatal(w.Code,w.Body.String())}
		for _,key:=range []string{"materialCost","processCost","materialEstimate","processEstimate","totalRate"} {
			if strings.Contains(w.Body.String(),`"`+key+`"`) {t.Fatalf("%s exposes %s without report permission",path,key)}
		}
	}
}
func (entryHTTPRepo) VoidEntry(context.Context, Actor, uuid.UUID, int, string) (Entry, error) {
	return Entry{}, invalid("The period is closed")
}
func TestDailyProductionHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path, body string
		permissions              []string
		cookie                   bool
		want                     int
	}{
		{name: "requires auth", method: "GET", path: "/production-entries", want: 401},
		{name: "requires permission", method: "POST", path: "/production-entries", body: `{}`, cookie: true, want: 403},
		{name: "validates input", method: "POST", path: "/production-entries", body: `{}`, cookie: true, permissions: []string{"production.entry"}, want: 422},
		{name: "invalid ID", method: "GET", path: "/production-entries/wrong", cookie: true, permissions: []string{"production.view"}, want: 400},
		{name: "no hard delete", method: "DELETE", path: "/production-entries/11111111-1111-1111-1111-111111111111", cookie: true, permissions: []string{"production.void"}, want: 404},
		{name: "void has separate permission", method: "POST", path: "/production-entries/11111111-1111-1111-1111-111111111111/void", body: `{"version":1,"reason":"wrong"}`, cookie: true, permissions: []string{"production.entry"}, want: 403},
		{name: "void rejected closed", method: "POST", path: "/production-entries/11111111-1111-1111-1111-111111111111/void", body: `{"version":1,"reason":"wrong"}`, cookie: true, permissions: []string{"production.void"}, want: 422},
		{name: "create", method: "POST", path: "/production-entries", body: `{"orderId":"11111111-1111-1111-1111-111111111111","operationId":"22222222-2222-2222-2222-222222222222","operatorId":"33333333-3333-3333-3333-333333333333","productionDate":"2026-09-14","shift":"1","processed":"10","good":"9","rejected":"1"}`, cookie: true, permissions: []string{"production.entry"}, want: 201},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			RegisterEntryRoutes(r, entryHTTPRepo{}, testAuth{tc.permissions})
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.cookie {
				req.Header.Set("Cookie", "session=test")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
