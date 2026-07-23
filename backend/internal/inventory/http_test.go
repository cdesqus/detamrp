package inventory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/auth"
)

type fakeAuthenticator struct {
	user auth.User
}

func (f fakeAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return f.user, nil
}

func TestInventoryRoutesRejectInvalidStatusBeforeQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, nil, fakeAuthenticator{user: auth.User{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Permissions: []string{"inventory.view"},
	}})
	request := httptest.NewRequest(http.MethodGet, "/inventory/stock?status=UNKNOWN", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestInventoryRoutesRequireViewPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, nil, fakeAuthenticator{user: auth.User{ID: uuid.New(), TenantID: uuid.New()}})
	request := httptest.NewRequest(http.MethodGet, "/inventory/stock", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}
