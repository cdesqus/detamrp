package masterdata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/auth"
)

type referenceAuth struct{ user auth.User }

func (f referenceAuth) Authenticate(context.Context, string) (auth.User, error) { return f.user, nil }

func TestReferenceMasterListRoutesReturnEmptyEnvelopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authenticator := referenceAuth{user: auth.User{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Permissions: []string{"master_data.view"},
	}}
	RegisterCategoryRoutes(router, NewCategoryService(&fakeCategoryRepository{}), authenticator)
	RegisterPackingRoutes(router, NewPackingService(&fakePackingRepository{}), authenticator)
	RegisterPlantRoutes(router, NewPlantService(&fakePlantRepository{}), authenticator)

	for _, path := range []string{"/master-data/categories", "/master-data/packings", "/master-data/plants"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.String() != `{"items":[],"total":0}` {
			t.Errorf("%s returned %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestReferenceMasterRoutesEnforceManagePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authenticator := referenceAuth{user: auth.User{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		Permissions: []string{"master_data.view"},
	}}
	RegisterCategoryRoutes(router, NewCategoryService(&fakeCategoryRepository{}), authenticator)

	request := httptest.NewRequest(http.MethodPost, "/master-data/categories", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}
