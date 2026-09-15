package production

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
	"time"
)

func validPlan() Plan {
	return Plan{PlantID: uuid.New(), PeriodStart: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), Lines: []PlanLine{{FinishedGoodID: uuid.New(), PlannedQty: decimal.NewFromInt(10)}}}
}
func TestPlanInputUsesServerNumberAndUnit(t *testing.T) {
	if err := ValidatePlanInput(validPlan()); err != nil {
		t.Fatal(err)
	}
}
func TestPlanInputRejectsInvalidTargets(t *testing.T) {
	cases := map[string]func(*Plan){"missing plant": func(p *Plan) { p.PlantID = uuid.Nil }, "reversed period": func(p *Plan) { p.PeriodEnd = p.PeriodStart.Add(-24 * time.Hour) }, "no lines": func(p *Plan) { p.Lines = nil }, "zero quantity": func(p *Plan) { p.Lines[0].PlannedQty = decimal.Zero }, "two outputs": func(p *Plan) { p.Lines[0].RawMaterialID = uuid.New() }, "no output": func(p *Plan) { p.Lines[0].FinishedGoodID = uuid.Nil }, "excess precision": func(p *Plan) { p.Lines[0].PlannedQty = decimal.RequireFromString("0.0000001") }}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			p := validPlan()
			change(&p)
			if ValidatePlanInput(p) == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
