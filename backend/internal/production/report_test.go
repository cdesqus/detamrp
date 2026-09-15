package production

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func sampleReport() OrderReport {
	order := Order{
		OrderNumber: "PRO-0000001", PlanNumber: "PP-202609-000001", PartNumber: "FG-001", PartName: "Panel",
		UnitCode: "PCS", PlantName: "Main plant", Status: OrderInProgress, Currency: "IDR",
		PeriodStart: "2026-09-01", DueDate: "2026-09-30", PlannedQty: decimal.NewFromInt(100),
		BOMRevision: 1, RoutingRevision: 2,
		Operations: []Operation{
			{ID: uuid.New(), Code: "STAMPING", Name: "Stamping", Sequence: 1, Rate: decimal.NewFromInt(5),
				PlannedQty: decimal.NewFromInt(100), ProcessedQty: decimal.NewFromInt(30), GoodQty: decimal.NewFromInt(28),
				RejectQty: decimal.NewFromInt(2), WIPQty: decimal.NewFromInt(3)},
			{ID: uuid.New(), Code: "WELDING", Name: "Welding", Sequence: 2, Rate: decimal.NewFromInt(8),
				PlannedQty: decimal.NewFromInt(100), ProcessedQty: decimal.NewFromInt(20), GoodQty: decimal.NewFromInt(19),
				RejectQty: decimal.NewFromInt(1), WIPQty: decimal.NewFromInt(5)},
		},
	}
	cost := OrderCost{
		OrderNumber: order.OrderNumber, Currency: "IDR", PlannedQty: order.PlannedQty, GoodQty: decimal.NewFromInt(19),
		RejectQty: decimal.NewFromInt(3), MaterialEstimate: decimal.NewFromInt(1000), ProcessEstimate: decimal.NewFromInt(1300),
		MaterialActual: decimal.NewFromInt(300), ProcessActual: decimal.NewFromInt(310), WIPValue: decimal.NewFromInt(150),
		Operations: []OperationCost{
			{Code: "STAMPING", Sequence: 1, ActualCost: decimal.NewFromInt(150), CostPerPiece: decimal.NewFromInt(5)},
			{Code: "WELDING", Sequence: 2, ActualCost: decimal.NewFromInt(160), CostPerPiece: decimal.NewFromInt(8)},
		},
		Materials: []MaterialCost{{PartNumber: "RM-001", PartName: "Steel", UnitCode: "PCS",
			UsedQty: decimal.NewFromInt(32), StandardQty: decimal.NewFromInt(30), QtyVariance: decimal.NewFromInt(2),
			AveragePrice: decimal.NewFromInt(10), ActualCost: decimal.NewFromInt(320)}},
	}
	cost.summarise()
	wip := WIPOrderBalance{
		OrderNumber: order.OrderNumber, TotalQty: decimal.NewFromInt(8), TotalValue: decimal.NewFromInt(150),
		Operations: []WIPOperationBalance{{Code: "STAMPING", Received: decimal.NewFromInt(28), TransferredOut: decimal.NewFromInt(25),
			OnHand: decimal.NewFromInt(3), Balance: decimal.NewFromInt(3)}},
		Movements: []WIPMovement{{Type: MovementTransfer, SourceCode: "STAMPING", DestinationCode: "WELDING",
			Quantity: decimal.NewFromInt(25), UnitCost: decimal.NewFromInt(15), TotalCost: decimal.NewFromInt(375),
			MovementDate: "2026-09-13", EntryNumber: "DP-0000001", CreatedBy: "Operator"}},
	}
	return OrderReport{Order: order, Cost: cost, WIP: wip, GeneratedAt: time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)}
}

func sampleDashboard() ProductionDashboard {
	d := ProductionDashboard{
		Filter: DashboardFilter{From: "2026-09-01", To: "2026-09-30"},
		Totals: DashboardTotals{PlannedQty: decimal.NewFromInt(100), ProcessedQty: decimal.NewFromInt(50),
			GoodQty: decimal.NewFromInt(19), RejectQty: decimal.NewFromInt(3), WIPQty: decimal.NewFromInt(8),
			WIPValue: decimal.NewFromInt(150), MaterialActual: decimal.NewFromInt(300), ProcessActual: decimal.NewFromInt(310),
			Entries: 3, ActiveOrders: 1, OpenOrders: 1, Currency: "IDR"},
		Parts:       []PartPerformance{{PartNumber: "FG-001", PartName: "Panel", UnitCode: "PCS", Orders: 1, PlannedQty: decimal.NewFromInt(100), GoodQty: decimal.NewFromInt(19), RejectQty: decimal.NewFromInt(3)}},
		Processes:   []ProcessWIP{{Code: "STAMPING", Quantity: decimal.NewFromInt(3), Value: decimal.RequireFromString("56.25")}},
		Quality:     []OperationQuality{{Code: "WELDING", ProcessedQty: decimal.NewFromInt(20), GoodQty: decimal.NewFromInt(19), RejectQty: decimal.NewFromInt(1)}},
		Costs:       []PartCost{{PartNumber: "FG-001", PartName: "Panel", Orders: 1, GoodQty: decimal.NewFromInt(19), MaterialActual: decimal.NewFromInt(300), ProcessActual: decimal.NewFromInt(310), TotalActual: decimal.NewFromInt(610), CostPerPiece: decimal.RequireFromString("32.10"), Currency: "IDR"}},
		GeneratedAt: time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC),
	}
	d.Totals.summarise()
	return d
}

func TestRenderedDocumentsAreNativeFiles(t *testing.T) {
	report, dashboard := sampleReport(), sampleDashboard()
	for name, render := range map[string]func() ([]byte, error){
		"order xlsx":     func() ([]byte, error) { return RenderOrderReportXLSX(report) },
		"dashboard xlsx": func() ([]byte, error) { return RenderDashboardXLSX(dashboard) },
	} {
		document, e := render()
		if e != nil {
			t.Fatalf("%s: %v", name, e)
		}
		// A real workbook is a zip archive, never a screenshot or HTML.
		if len(document) < 1024 || !bytes.HasPrefix(document, []byte("PK\x03\x04")) {
			t.Fatalf("%s is not a spreadsheet: %d bytes", name, len(document))
		}
	}
	for name, render := range map[string]func() ([]byte, error){
		"order pdf":     func() ([]byte, error) { return RenderOrderReportPDF(report) },
		"dashboard pdf": func() ([]byte, error) { return RenderDashboardPDF(dashboard) },
	} {
		document, e := render()
		if e != nil {
			t.Fatalf("%s: %v", name, e)
		}
		if len(document) < 1024 || !bytes.HasPrefix(document, []byte("%PDF-")) {
			t.Fatalf("%s is not a PDF: %d bytes", name, len(document))
		}
		if !bytes.Contains(document, []byte("Production")) {
			t.Fatalf("%s lost its title", name)
		}
	}
}

func TestOrderReportPDFCarriesTheExecutionFigures(t *testing.T) {
	document, e := RenderOrderReportPDF(sampleReport())
	if e != nil {
		t.Fatal(e)
	}
	for _, expected := range []string{"PRO-0000001", "STAMPING", "WELDING", "RM-001", "Finished output cost"} {
		if !bytes.Contains(document, []byte(expected)) {
			t.Fatalf("document is missing %q", expected)
		}
	}
}

type dashboardHTTPRepo struct{ DashboardRepository }

func (dashboardHTTPRepo) Dashboard(_ context.Context, _ Actor, filter DashboardFilter) (ProductionDashboard, error) {
	d := sampleDashboard()
	d.Filter = filter
	return d, nil
}
func (dashboardHTTPRepo) OrderReport(context.Context, Actor, uuid.UUID) (OrderReport, error) {
	return sampleReport(), nil
}

func TestDashboardHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, path  string
		permissions []string
		want        int
		contentType string
	}{
		{"floor may read the dashboard", "/production-dashboard", []string{"production.view"}, 200, "application/json"},
		{"strangers may not", "/production-dashboard", nil, 403, ""},
		{"invalid windows are rejected", "/production-dashboard?from=2026-09-30&to=2026-09-01", []string{"production.view"}, 422, ""},
		{"exports need the report permission", "/production-reports/dashboard.xlsx", []string{"production.view"}, 403, ""},
		{"dashboard workbook", "/production-reports/dashboard.xlsx", []string{"production.report"}, 200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"dashboard document", "/production-reports/dashboard.pdf", []string{"production.report"}, 200, "application/pdf"},
		{"order workbook", "/production-reports/orders/11111111-1111-1111-1111-111111111111/execution.xlsx", []string{"production.report"}, 200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"order document", "/production-reports/orders/11111111-1111-1111-1111-111111111111/execution.pdf", []string{"production.report"}, 200, "application/pdf"},
		{"identifiers are validated", "/production-reports/orders/not-a-uuid/execution.pdf", []string{"production.report"}, 400, ""},
	} {
		router := gin.New()
		RegisterDashboardRoutes(router, dashboardHTTPRepo{}, testAuth{tc.permissions})
		request := httptest.NewRequest("GET", tc.path, nil)
		request.Header.Set("Cookie", "session=test")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != tc.want {
			t.Fatalf("%s: GET %s returned %d %s", tc.name, tc.path, recorder.Code, recorder.Body.String())
		}
		if tc.contentType != "" && !strings.HasPrefix(recorder.Header().Get("Content-Type"), tc.contentType) {
			t.Fatalf("%s: content type %s", tc.name, recorder.Header().Get("Content-Type"))
		}
		if tc.want == 200 && tc.contentType != "application/json" && recorder.Header().Get("Content-Disposition") == "" {
			t.Fatalf("%s: document is not offered as a download", tc.name)
		}
	}
}

func TestDashboardHidesCostsFromTheFloor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		permissions []string
		wantCosts   bool
	}{
		{[]string{"production.view"}, false},
		{[]string{"production.view", "production.report"}, true},
	} {
		router := gin.New()
		RegisterDashboardRoutes(router, dashboardHTTPRepo{}, testAuth{tc.permissions})
		request := httptest.NewRequest("GET", "/production-dashboard", nil)
		request.Header.Set("Cookie", "session=test")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		body := recorder.Body.String()
		if bytes.Contains([]byte(body), []byte(`"totalActual":"610"`)) != tc.wantCosts {
			t.Fatalf("cost exposure for %v: %s", tc.permissions, body)
		}
		if !bytes.Contains([]byte(body), []byte(`"goodQty":"19"`)) {
			t.Fatalf("quantities missing for %v: %s", tc.permissions, body)
		}
	}
}
