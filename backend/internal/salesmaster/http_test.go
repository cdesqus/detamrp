package salesmaster

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/auth"
)

type testAuthenticator struct{ user auth.User }

func (a testAuthenticator) Authenticate(context.Context, string) (auth.User, error) {
	return a.user, nil
}

type fakeRepository struct {
	finishedGood FinishedGood
	updateCalls  int
}

func (r *fakeRepository) ListCustomers(context.Context, Actor, ListQuery) ([]Customer, int, error) {
	return nil, 0, nil
}
func (r *fakeRepository) GetCustomer(context.Context, Actor, uuid.UUID) (Customer, error) {
	return Customer{}, nil
}
func (r *fakeRepository) CreateCustomer(context.Context, Actor, CustomerInput) (Customer, error) {
	return Customer{}, nil
}
func (r *fakeRepository) UpdateCustomer(context.Context, Actor, uuid.UUID, CustomerInput) (Customer, error) {
	return Customer{}, nil
}
func (r *fakeRepository) ListFinishedGoods(context.Context, Actor, ListQuery) ([]FinishedGood, int, error) {
	return nil, 0, nil
}
func (r *fakeRepository) GetFinishedGood(context.Context, Actor, uuid.UUID) (FinishedGood, error) {
	return r.finishedGood, nil
}
func (r *fakeRepository) CreateFinishedGood(context.Context, Actor, FinishedGoodInput) (FinishedGood, error) {
	return FinishedGood{}, nil
}
func (r *fakeRepository) UpdateFinishedGood(_ context.Context, _ Actor, _ uuid.UUID, input FinishedGoodInput) (FinishedGood, error) {
	r.updateCalls++
	r.finishedGood.ItemCode = input.ItemCode
	r.finishedGood.Name = input.Name
	r.finishedGood.BaseUnitID = input.BaseUnitID
	r.finishedGood.SalesPrice = input.SalesPrice
	r.finishedGood.Currency = input.Currency
	return r.finishedGood, nil
}

func TestFinishedGoodUpdateDeniesPriceChangeWithoutPricePermission(t *testing.T) {
	repository := &fakeRepository{finishedGood: FinishedGood{
		ID: uuid.New(), ItemCode: "FG-100", Name: "Assembly", BaseUnitID: uuid.New(),
		SalesPrice: decimal.RequireFromString("100.000000"), Currency: "IDR", PriceVersion: 1, Active: true,
	}}
	router := salesMasterRouter(repository, []string{"fg.view", "fg.manage"})
	body := `{"itemCode":"FG-100","name":"Assembly","baseUnitId":"` + repository.finishedGood.BaseUnitID.String() + `","salesPrice":"125.000000","currency":"IDR","active":true}`

	response := performRequest(router, http.MethodPut, "/finished-goods/"+repository.finishedGood.ID.String(), body)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", response.Code, response.Body.String())
	}
	if repository.updateCalls != 0 {
		t.Fatalf("repository update calls = %d, want 0", repository.updateCalls)
	}
}

func TestFinishedGoodUpdateAllowsMetadataWithoutPricePermission(t *testing.T) {
	repository := &fakeRepository{finishedGood: FinishedGood{
		ID: uuid.New(), ItemCode: "FG-100", Name: "Assembly", BaseUnitID: uuid.New(),
		SalesPrice: decimal.RequireFromString("100.000000"), Currency: "IDR", PriceVersion: 1, Active: true,
	}}
	router := salesMasterRouter(repository, []string{"fg.view", "fg.manage"})
	body := `{"itemCode":"FG-100","name":"Renamed Assembly","baseUnitId":"` + repository.finishedGood.BaseUnitID.String() + `","salesPrice":"100.000000","currency":"IDR","active":true}`

	response := performRequest(router, http.MethodPut, "/finished-goods/"+repository.finishedGood.ID.String(), body)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if repository.updateCalls != 1 {
		t.Fatalf("repository update calls = %d, want 1", repository.updateCalls)
	}
}

func TestFinishedGoodPatchUpdatesMetadata(t *testing.T) {
	repository := &fakeRepository{finishedGood: FinishedGood{
		ID: uuid.New(), ItemCode: "FG-100", Name: "Assembly", BaseUnitID: uuid.New(),
		SalesPrice: decimal.RequireFromString("100.000000"), Currency: "IDR", PriceVersion: 1, Active: true,
	}}
	router := salesMasterRouter(repository, []string{"fg.view", "fg.manage"})
	body := `{"itemCode":"FG-100","name":"Renamed Assembly","baseUnitId":"` + repository.finishedGood.BaseUnitID.String() + `","salesPrice":"100.000000","currency":"IDR","active":true}`

	response := performRequest(router, http.MethodPatch, "/finished-goods/"+repository.finishedGood.ID.String(), body)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if repository.updateCalls != 1 {
		t.Fatalf("repository update calls = %d, want 1", repository.updateCalls)
	}
}

func TestSalesMasterListRoutesUseIndependentViewPermissions(t *testing.T) {
	repository := &fakeRepository{}
	customerRouter := salesMasterRouter(repository, []string{"customer.view"})
	if response := performRequest(customerRouter, http.MethodGet, "/customers", ""); response.Code != http.StatusOK || response.Body.String() != `{"items":[],"total":0}` {
		t.Fatalf("customer list = %d %s", response.Code, response.Body.String())
	}
	if response := performRequest(customerRouter, http.MethodGet, "/finished-goods", ""); response.Code != http.StatusForbidden {
		t.Fatalf("finished-good status = %d, want 403", response.Code)
	}
}

func salesMasterRouter(repository Repository, permissions []string) http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, NewService(repository), testAuthenticator{user: auth.User{
		ID: uuid.New(), TenantID: uuid.New(), Permissions: permissions,
	}})
	return router
}

func performRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
