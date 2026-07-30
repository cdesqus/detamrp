package settings

import (
	"bytes"
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

func TestPublicBrandingDoesNotRequireSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, NewService(&fakeRepo{}), fakeAuth{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/public/branding", nil))
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"companyName":"DETA MRP"`)) {
		t.Fatalf("public branding = %d %s", w.Code, w.Body.String())
	}
	if bytes.Contains(w.Body.Bytes(), []byte("smtp")) || bytes.Contains(w.Body.Bytes(), []byte("tenant")) {
		t.Fatal("public branding leaked private settings")
	}
}
