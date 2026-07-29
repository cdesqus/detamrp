package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"order-stock/backend/internal/auth"
)

type routeAuthenticator struct{}

func (routeAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return auth.User{}, nil
}

func TestRegisterRoutesExposesDashboard(t *testing.T) {
	router := gin.New()
	RegisterRoutes(router, nil, routeAuthenticator{})
	for _, route := range router.Routes() {
		if route.Method == "GET" && route.Path == "/dashboard" {
			return
		}
	}
	t.Fatal("GET /dashboard is not registered")
}

type permissionAuthenticator struct{ permissions []string }

func (a permissionAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return auth.User{Permissions: a.permissions}, nil
}

func TestDashboardRequiresDedicatedViewPermission(t *testing.T) {
	for _, test := range []struct {
		name        string
		permissions []string
		status      int
	}{
		{name: "dashboard view reaches handler", permissions: []string{"dashboard.view"}, status: http.StatusUnprocessableEntity},
		{name: "inventory view is denied", permissions: []string{"inventory.view"}, status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			RegisterRoutes(router, nil, permissionAuthenticator{permissions: test.permissions})
			request := httptest.NewRequest(http.MethodGet, "/dashboard?from=invalid", nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}
