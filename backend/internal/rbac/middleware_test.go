package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequirePermissionsAllowsAndDenies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/allowed", func(c *gin.Context) { c.Set(ContextPermissionsKey, []string{"po.view"}) }, RequirePermissions("po.view"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/denied", func(c *gin.Context) { c.Set(ContextPermissionsKey, []string{"po.view"}) }, RequirePermissions("po.approve"), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet, "/allowed", nil))
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("expected allowed request, got %d", allowed.Code)
	}

	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/denied", nil))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected denied request, got %d", denied.Code)
	}
}
