package production

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type routingHTTPRepo struct{ RoutingRepository }

func (routingHTTPRepo) ListRoutings(context.Context, Actor) ([]Routing, error) {
	return []Routing{{Name: "Panel routing", Revision: 1, Active: true}}, nil
}
func (routingHTTPRepo) CreateRouting(_ context.Context, _ Actor, i RoutingInput) (Routing, error) {
	return Routing{Name: i.Name, Revision: 1, Steps: i.Steps}, nil
}
func (routingHTTPRepo) UpdateRouting(context.Context, Actor, uuid.UUID, RoutingInput) (Routing, error) {
	return Routing{}, ErrConflict
}
func (routingHTTPRepo) RoutingAction(_ context.Context, _ Actor, _ uuid.UUID, action string) (Routing, error) {
	if action == "delete" {
		return Routing{}, ErrConflict
	}
	return Routing{Name: "Panel routing", Active: action == "activate"}, nil
}

func TestRoutingHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const id = "/production-routings/11111111-1111-1111-1111-111111111111"
	const body = `{"partId":"22222222-2222-2222-2222-222222222222","kind":"FG","name":"Panel routing","currency":"IDR","steps":[{"code":"STAMPING","name":"Stamping","rate":"250"}]}`
	for _, tc := range []struct {
		name, method, path, body string
		permissions              []string
		want                     int
	}{
		{"viewers may not list without permission", "GET", "/production-routings", "", nil, 403},
		{"viewers may list", "GET", "/production-routings", "", []string{"production.view"}, 200},
		{"viewers may not create", "POST", "/production-routings", body, []string{"production.view"}, 403},
		{"routings need at least one operation", "POST", "/production-routings", `{"partId":"22222222-2222-2222-2222-222222222222","kind":"FG","name":"Panel routing","steps":[]}`, []string{"production.routing", "production.report"}, 422},
		{"duplicate operations are rejected", "POST", "/production-routings", `{"partId":"22222222-2222-2222-2222-222222222222","kind":"FG","name":"Panel","steps":[{"code":"STAMPING","name":"A","rate":"1"},{"code":"stamping","name":"B","rate":"1"}]}`, []string{"production.routing", "production.report"}, 422},
		{"managers may create", "POST", "/production-routings", body, []string{"production.routing", "production.report"}, 201},
		{"used routings cannot be edited", "PUT", id, body, []string{"production.routing", "production.report"}, 409},
		{"activation returns the routing", "POST", id + "/activate", "", []string{"production.routing", "production.report"}, 200},
		{"used routings cannot be deleted", "DELETE", id, "", []string{"production.routing", "production.report"}, 409},
		{"unknown identifiers are rejected", "GET", "/production-routings/not-a-uuid", "", []string{"production.view"}, 400},
	} {
		router := gin.New()
		RegisterRoutingRoutes(router, routingHTTPRepo{}, testAuth{tc.permissions})
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
