package production

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Plan struct {
	ID                     uuid.UUID       `json:"id"`
	PlanNumber             string          `json:"planNumber"`
	PeriodStart            time.Time       `json:"periodStart"`
	PeriodEnd              time.Time       `json:"periodEnd"`
	PlantID                uuid.UUID       `json:"plantId"`
	Status                 PlanStatus      `json:"status"`
	Notes                  string          `json:"notes"`
	Lines                  []PlanLine      `json:"lines"`
	PlantName              string          `json:"plantName"`
	CreatedBy              string          `json:"createdBy"`
	UpdatedAt              time.Time       `json:"updatedAt"`
	TotalPart              int             `json:"totalPart"`
	TotalPlannedQty        decimal.Decimal `json:"totalPlannedQty"`
	LinkedProductionOrders int             `json:"linkedProductionOrders"`
	Orders                 []LinkedOrder   `json:"orders"`
	History                []PlanHistory   `json:"history"`
}

type LinkedOrder struct {
	ID          uuid.UUID       `json:"id"`
	OrderNumber string          `json:"orderNumber"`
	Status      string          `json:"status"`
	PlannedQty  decimal.Decimal `json:"plannedQty"`
	GoodQty     decimal.Decimal `json:"goodQty"`
}
type PlanHistory struct {
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	OccurredAt time.Time `json:"occurredAt"`
}

type PlanLine struct {
	ID             uuid.UUID       `json:"id"`
	PartNumber     string          `json:"partNumber"`
	PartName       string          `json:"partName"`
	FinishedGoodID uuid.UUID       `json:"finishedGoodId"`
	RawMaterialID  uuid.UUID       `json:"rawMaterialId"`
	PlannedQty     decimal.Decimal `json:"plannedQty"`
	UnitCode       string          `json:"unitCode"`
}

func ValidatePlanInput(p Plan) error {
	if p.PlantID == uuid.Nil {
		return errors.New("plant is required")
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
		if !line.PlannedQty.Equal(line.PlannedQty.Round(6)) || line.PlannedQty.GreaterThanOrEqual(decimal.New(1, 14)) {
			return errors.New("quantity must fit 14 integer digits and at most 6 decimal places")
		}
	}
	return nil
}
