package production

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type costHTTPRepo struct{ CostRepository }

func (costHTTPRepo) ListOrderCosts(context.Context, Actor) ([]OrderCost, error) {
	c := OrderCost{OrderNumber: "PRO-0000001", Currency: "IDR", PlannedQty: decimal.NewFromInt(100), GoodQty: decimal.NewFromInt(40),
		MaterialEstimate: decimal.NewFromInt(1000), ProcessEstimate: decimal.NewFromInt(500),
		MaterialActual: decimal.NewFromInt(1200), ProcessActual: decimal.NewFromInt(600), WIPValue: decimal.NewFromInt(300)}
	c.summarise()
	return []OrderCost{c}, nil
}
func (costHTTPRepo) GetOrderCost(context.Context, Actor, uuid.UUID) (OrderCost, error) {
	return OrderCost{OrderNumber: "PRO-0000001"}, nil
}
func (costHTTPRepo) ListPeriodCosts(context.Context, Actor) ([]PeriodCost, error) {
	return []PeriodCost{{Period: "2026-09", Closed: true, Orders: 1, Entries: 3, TotalActual: decimal.NewFromInt(1800)}}, nil
}

func TestCostHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, path  string
		permissions []string
		want        int
	}{
		{"costs need the report permission", "/production-costs", []string{"production.view"}, 403},
		{"entry clerks may not read costs", "/production-costs/periods", []string{"production.entry"}, 403},
		{"controllers may read order costs", "/production-costs", []string{"production.report"}, 200},
		{"controllers may read period costs", "/production-costs/periods", []string{"production.report"}, 200},
		{"controllers may read one order", "/production-costs/11111111-1111-1111-1111-111111111111", []string{"production.report"}, 200},
		{"identifiers are validated", "/production-costs/not-a-uuid", []string{"production.report"}, 400},
	} {
		router := gin.New()
		RegisterCostRoutes(router, costHTTPRepo{}, testAuth{tc.permissions})
		request := httptest.NewRequest("GET", tc.path, nil)
		request.Header.Set("Cookie", "session=test")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != tc.want {
			t.Fatalf("%s: GET %s returned %d %s", tc.name, tc.path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestCostPayloadCarriesTheDerivedTotals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterCostRoutes(router, costHTTPRepo{}, testAuth{[]string{"production.report"}})
	request := httptest.NewRequest("GET", "/production-costs", nil)
	request.Header.Set("Cookie", "session=test")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if e := json.Unmarshal(recorder.Body.Bytes(), &payload); e != nil {
		t.Fatal(e)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("unexpected payload: %s", recorder.Body.String())
	}
	for field, want := range map[string]string{"totalActual": "1800", "finishedCost": "1500", "actualUnitCost": "37.5", "variancePercent": "150"} {
		if payload.Items[0][field] != want {
			t.Fatalf("%s was %v, want %s", field, payload.Items[0][field], want)
		}
	}
}
