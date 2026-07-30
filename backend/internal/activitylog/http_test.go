package activitylog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-stock/backend/internal/auth"
)

type testAuthenticator struct {
	user auth.User
}

func (a testAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return a.user, nil
}

type recordingLister struct {
	query Query
	page  Page
}

func (l *recordingLister) List(_ context.Context, _ Actor, query Query) (Page, error) {
	l.query = query
	return l.page, nil
}

func TestActivityLogRouteRequiresDedicatedPermission(t *testing.T) {
	for _, test := range []struct {
		name        string
		permissions []string
		want        int
	}{
		{name: "allowed", permissions: []string{"activity_log.view"}, want: http.StatusOK},
		{name: "other settings permission denied", permissions: []string{"configuration.manage"}, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			RegisterRoutes(router, &recordingLister{}, testAuthenticator{
				user: auth.User{ID: uuid.New(), TenantID: uuid.New(), Permissions: test.permissions},
			})
			request := httptest.NewRequest(http.MethodGet, "/activity-logs", nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.want, response.Body.String())
			}
		})
	}
}

func TestActivityLogRouteRequiresAuthentication(t *testing.T) {
	router := gin.New()
	RegisterRoutes(router, &recordingLister{}, testAuthenticator{})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/activity-logs", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestActivityLogRouteParsesFiltersAndReturnsStableContract(t *testing.T) {
	lister := &recordingLister{page: Page{Items: []Item{}, Total: 0, Page: 2, PageSize: 25, Filters: FilterOptions{Actors: []ActorOption{}}}}
	router := gin.New()
	RegisterRoutes(router, lister, testAuthenticator{
		user: auth.User{ID: uuid.New(), TenantID: uuid.New(), Permissions: []string{"activity_log.view"}},
	})
	request := httptest.NewRequest(http.MethodGet, "/activity-logs?module=inventory&action=moved&page=2", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	if lister.query.Module != "INVENTORY" || lister.query.Action != "MOVED" || lister.query.Page != 2 {
		t.Fatalf("unexpected parsed query: %#v", lister.query)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"items", "total", "page", "pageSize", "filters"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("response missing %q: %s", key, response.Body.String())
		}
	}
}
