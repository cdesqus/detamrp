package production

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Plan struct {
	ID          uuid.UUID  `json:"id"`
	PlanNumber  string     `json:"planNumber"`
	PeriodStart time.Time  `json:"periodStart"`
	PeriodEnd   time.Time  `json:"periodEnd"`
	PlantID     uuid.UUID  `json:"plantId"`
	Status      PlanStatus `json:"status"`
	Notes       string     `json:"notes"`
	Lines       []PlanLine `json:"lines"`
}

type PlanLine struct {
	FinishedGoodID uuid.UUID       `json:"finishedGoodId"`
	RawMaterialID  uuid.UUID       `json:"rawMaterialId"`
	PlannedQty     decimal.Decimal `json:"plannedQty"`
	UnitCode       string          `json:"unitCode"`
}

func ValidatePlanInput(p Plan) error {
	if strings.TrimSpace(p.PlanNumber) == "" {
		return errors.New("plan number is required")
	}
	if p.PeriodStart.IsZero() || p.PeriodEnd.IsZero() {
		return errors.New("planning period is required")
	}
	if p.PeriodEnd.Before(p.PeriodStart) {
		return errors.New("period end cannot precede period start")
	}
	if len(p.Lines) == 0 {
		return errors.New("at least one planning line is required")
	}
	for _, line := range p.Lines {
		if (line.FinishedGoodID == uuid.Nil) == (line.RawMaterialID == uuid.Nil) {
			return errors.New("each line must reference exactly one part")
		}
		if !line.PlannedQty.GreaterThan(decimal.Zero) {
			return errors.New("planned quantity must be greater than zero")
		}
		if strings.TrimSpace(line.UnitCode) == "" {
			return errors.New("unit is required")
		}
	}
	return nil
}
