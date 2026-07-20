package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"order-stock/backend/internal/auth"
)

type fakeAuthenticator struct {
	result      auth.LoginResult
	err         error
	logoutToken string
}

func (f fakeAuthenticator) Login(context.Context, string, string) (auth.LoginResult, error) {
	return f.result, f.err
}
func (f fakeAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return f.result.User, f.err
}
func (f *fakeAuthenticator) Logout(_ context.Context, token string) error {
	f.logoutToken = token
	return f.err
}

func TestLoginSetsSecureHTTPOnlySessionCookie(t *testing.T) {
	user := auth.User{ID: uuid.New(), TenantID: uuid.New(), Username: "director", DisplayName: "Director"}
	server := NewServer(WithAuthenticator(&fakeAuthenticator{result: auth.LoginResult{Token: "opaque", ExpiresAt: time.Now().Add(time.Hour), User: user}}))
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"director","password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %#v", cookies)
	}
}

func TestLoginHidesCredentialFailureDetails(t *testing.T) {
	server := NewServer(WithAuthenticator(&fakeAuthenticator{err: errors.Join(auth.ErrInvalidCredentials, errors.New("database detail"))}))
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"unknown","password":"wrong"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "database detail") {
		t.Fatal("internal error leaked")
	}
}

func TestMeRequiresSessionAndLogoutClearsIt(t *testing.T) {
	user := auth.User{ID: uuid.New(), TenantID: uuid.New(), Username: "director", DisplayName: "Director"}
	authenticator := &fakeAuthenticator{result: auth.LoginResult{User: user}}
	server := NewServer(WithAuthenticator(authenticator))

	unauthorized := httptest.NewRecorder()
	server.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing cookie 401, got %d", unauthorized.Code)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meRequest.AddCookie(&http.Cookie{Name: "session", Value: "opaque"})
	me := httptest.NewRecorder()
	server.ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), "director") {
		t.Fatalf("unexpected me response: %d %s", me.Code, me.Body.String())
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: "session", Value: "opaque"})
	logout := httptest.NewRecorder()
	server.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusNoContent || authenticator.logoutToken != "opaque" {
		t.Fatalf("logout failed: %d token=%q", logout.Code, authenticator.logoutToken)
	}
}
