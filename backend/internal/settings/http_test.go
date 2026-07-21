package settings

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"order-stock/backend/internal/auth"
	"testing"
)

type fakeAuth struct{ user auth.User }

func (f fakeAuth) Authenticate(context.Context, string) (auth.User, error) { return f.user, nil }
func TestSettingsRoutesRequireSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, NewService(&fakeRepo{}), fakeAuth{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/settings/users", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}
