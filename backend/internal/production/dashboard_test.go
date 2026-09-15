package production

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestDashboardFilterDefaultsToTheRunningMonth(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	filter, e := ParseDashboardFilter("", "", now)
	if e != nil {
		t.Fatal(e)
	}
	if filter.From != "2026-09-01" || filter.To != "2026-09-15" {
		t.Fatalf("wrong default window: %+v", filter)
	}
}

func TestDashboardFilterValidation(t *testing.T) {
	now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	if _, e := ParseDashboardFilter("2026-09-01", "2026-09-30", now); e != nil {
		t.Fatal(e)
	}
	for name, window := range map[string][2]string{
		"bad start":   {"2026-13-01", "2026-09-30"},
		"bad end":     {"2026-09-01", "2026-02-30"},
		"reversed":    {"2026-09-30", "2026-09-01"},
		"over a year": {"2025-01-01", "2026-09-30"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, e := ParseDashboardFilter(window[0], window[1], now); e == nil {
				t.Fatal("invalid window accepted")
			}
		})
	}
}

func TestDashboardTotalsDeriveRatesAndAchievement(t *testing.T) {
	totals := DashboardTotals{
		PlannedQty: decimal.NewFromInt(100), ProcessedQty: decimal.NewFromInt(80),
		GoodQty: decimal.NewFromInt(72), RejectQty: decimal.NewFromInt(8),
		MaterialActual: decimal.NewFromInt(500), ProcessActual: decimal.NewFromInt(300),
	}
	totals.summarise()
	if !totals.RejectRate.Equal(decimal.NewFromInt(10)) || !totals.Achievement.Equal(decimal.NewFromInt(72)) {
		t.Fatalf("wrong rates: %+v", totals)
	}
	if !totals.TotalActual.Equal(decimal.NewFromInt(800)) {
		t.Fatalf("wrong cost total: %s", totals.TotalActual)
	}
	empty := DashboardTotals{}
	empty.summarise()
	if !empty.RejectRate.IsZero() || !empty.Achievement.IsZero() {
		t.Fatalf("rates without production: %+v", empty)
	}
}

func TestDashboardWithoutCostsKeepsQuantities(t *testing.T) {
	d := ProductionDashboard{
		Totals:    DashboardTotals{GoodQty: decimal.NewFromInt(72), WIPQty: decimal.NewFromInt(5), WIPValue: decimal.NewFromInt(500), TotalActual: decimal.NewFromInt(800)},
		Processes: []ProcessWIP{{Code: "STAMPING", Quantity: decimal.NewFromInt(5), Value: decimal.NewFromInt(500)}},
		Costs:     []PartCost{{PartNumber: "FG-001"}},
	}
	stripped := d.withoutCosts()
	if !stripped.Totals.GoodQty.Equal(decimal.NewFromInt(72)) || !stripped.Totals.WIPQty.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("quantities were stripped: %+v", stripped.Totals)
	}
	if !stripped.Totals.WIPValue.IsZero() || !stripped.Totals.TotalActual.IsZero() || len(stripped.Costs) != 0 {
		t.Fatalf("costs leaked: %+v", stripped.Totals)
	}
	if !stripped.Processes[0].Value.IsZero() || !stripped.Processes[0].Quantity.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("process value leaked: %+v", stripped.Processes[0])
	}
}
