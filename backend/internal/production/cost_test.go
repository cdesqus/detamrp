package production

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestFinishedCostExcludesWorkInProgress(t *testing.T) {
	// 1,000 was booked on the order but 300 of it is still sitting between
	// operations, so only 700 left the order as finished output.
	if got := finishedCost(decimal.NewFromInt(1000), decimal.NewFromInt(300)); !got.Equal(decimal.NewFromInt(700)) {
		t.Fatalf("wrong finished cost: %s", got)
	}
	// A rounding overshoot must never report a negative finished cost.
	if got := finishedCost(decimal.NewFromInt(100), decimal.NewFromInt(120)); !got.IsZero() {
		t.Fatalf("negative finished cost: %s", got)
	}
}

func TestPerUnitHandlesOrdersWithoutOutput(t *testing.T) {
	if got := perUnit(decimal.NewFromInt(700), decimal.NewFromInt(8)); !got.Equal(decimal.RequireFromString("87.5")) {
		t.Fatalf("wrong unit cost: %s", got)
	}
	if got := perUnit(decimal.NewFromInt(700), decimal.Zero); !got.IsZero() {
		t.Fatalf("unit cost without output: %s", got)
	}
}

func TestCostVarianceComparesAgainstTheReleaseEstimate(t *testing.T) {
	difference, percent := costVariance(decimal.NewFromInt(110), decimal.NewFromInt(100))
	if !difference.Equal(decimal.NewFromInt(10)) || !percent.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("overrun not reported: %s %s", difference, percent)
	}
	difference, percent = costVariance(decimal.NewFromInt(90), decimal.NewFromInt(100))
	if !difference.Equal(decimal.NewFromInt(-10)) || !percent.Equal(decimal.NewFromInt(-10)) {
		t.Fatalf("saving not reported: %s %s", difference, percent)
	}
	if _, percent = costVariance(decimal.NewFromInt(10), decimal.Zero); !percent.IsZero() {
		t.Fatalf("percentage without an estimate: %s", percent)
	}
}

func TestOrderCostSummary(t *testing.T) {
	// 100 planned at 15 per piece estimated; 1,200 material and 600 process were
	// booked, 300 of which is still in WIP, over 40 good pieces.
	c := OrderCost{
		PlannedQty:       decimal.NewFromInt(100),
		GoodQty:          decimal.NewFromInt(40),
		MaterialEstimate: decimal.NewFromInt(1000),
		ProcessEstimate:  decimal.NewFromInt(500),
		MaterialActual:   decimal.NewFromInt(1200),
		ProcessActual:    decimal.NewFromInt(600),
		WIPValue:         decimal.NewFromInt(300),
	}
	c.summarise()
	if !c.TotalEstimate.Equal(decimal.NewFromInt(1500)) || !c.TotalActual.Equal(decimal.NewFromInt(1800)) {
		t.Fatalf("wrong totals: %+v", c)
	}
	if !c.FinishedCost.Equal(decimal.NewFromInt(1500)) {
		t.Fatalf("wrong finished cost: %s", c.FinishedCost)
	}
	if !c.PlannedUnitCost.Equal(decimal.NewFromInt(15)) || !c.ActualUnitCost.Equal(decimal.RequireFromString("37.5")) {
		t.Fatalf("wrong unit costs: %s %s", c.PlannedUnitCost, c.ActualUnitCost)
	}
	if !c.Variance.Equal(decimal.RequireFromString("22.5")) || !c.VariancePercent.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("wrong variance: %s %s", c.Variance, c.VariancePercent)
	}
}

func TestOrderCostWithoutProductionStaysNeutral(t *testing.T) {
	c := OrderCost{PlannedQty: decimal.NewFromInt(50), MaterialEstimate: decimal.NewFromInt(500)}
	c.summarise()
	if !c.ActualUnitCost.IsZero() || !c.FinishedCost.IsZero() || !c.PlannedUnitCost.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("unstarted order reported production: %+v", c)
	}
}
