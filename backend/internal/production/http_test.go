package production

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http/httptest"
	"order-stock/backend/internal/auth"
	"strings"
	"testing"
)

type testAuth struct{ permissions []string }

func (a testAuth) Authenticate(context.Context, string) (auth.User, error) {
	return auth.User{ID: uuid.New(), TenantID: uuid.New(), Permissions: a.permissions}, nil
}

type testRepo struct{ PlanRepository }

func (testRepo) Create(_ context.Context, _ Actor, p Plan) (Plan, error) {
	p.ID = uuid.New()
	p.PlanNumber = "PP-202609-000001"
	return p, nil
}
func (testRepo) Approve(context.Context, Actor, uuid.UUID) (Plan, error) { return Plan{}, ErrConflict }
func TestPlanningHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path, body string
		permissions              []string
		cookie                   bool
		want                     int
	}{
		{name: "authentication", method: "GET", path: "/production-plans", want: 401},
		{name: "permission", method: "GET", path: "/production-plans", cookie: true, want: 403},
		{name: "invalid id", method: "GET", path: "/production-plans/nope", cookie: true, permissions: []string{"production.view"}, want: 400},
		{name: "invalid input", method: "POST", path: "/production-plans", body: `{}`, cookie: true, permissions: []string{"production.plan"}, want: 422},
		{name: "conflict", method: "POST", path: "/production-plans/11111111-1111-1111-1111-111111111111/approve", cookie: true, permissions: []string{"production.plan"}, want: 409},
		{name: "create standalone", method: "POST", path: "/production-plans", body: `{"plantId":"11111111-1111-1111-1111-111111111111","periodStart":"2026-09-01T00:00:00Z","periodEnd":"2026-09-30T00:00:00Z","lines":[{"finishedGoodId":"22222222-2222-2222-2222-222222222222","plannedQty":"10"}]}`, cookie: true, permissions: []string{"production.plan"}, want: 201},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			RegisterRoutes(r, NewPlanService(testRepo{}), testAuth{tc.permissions})
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
