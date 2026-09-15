package production

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const maxRoutingSteps = 20

// RoutingInput is the caller-supplied routing definition; revision and active
// state are decided by the store so a client can never assign them.
type RoutingInput struct {
	PartID   uuid.UUID     `json:"partId"`
	Kind     string        `json:"kind"`
	Name     string        `json:"name"`
	Currency string        `json:"currency"`
	Steps    []RoutingStep `json:"steps"`
}

type Routing struct {
	ID           uuid.UUID       `json:"id"`
	PartID       uuid.UUID       `json:"partId"`
	Kind         string          `json:"kind"`
	PartNumber   string          `json:"partNumber"`
	PartName     string          `json:"partName"`
	UnitCode     string          `json:"unitCode"`
	Name         string          `json:"name"`
	Currency     string          `json:"currency"`
	Revision     int             `json:"revision"`
	Active       bool            `json:"active"`
	Steps        []RoutingStep   `json:"steps"`
	TotalRate    decimal.Decimal `json:"totalRate"`
	LinkedOrders int             `json:"linkedOrders"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	CreatedBy    string          `json:"createdBy"`
	History      []PlanHistory   `json:"history"`
}

type RoutingOptions struct {
	Parts []PartOption `json:"parts"`
}

func normalizeStepCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func validStepCode(code string) bool {
	if code == "" || len(code) > 20 {
		return false
	}
	for _, r := range code {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// Normalize trims text fields and upper-cases codes so stored routings and the
// snapshots taken from them stay comparable.
func (i *RoutingInput) Normalize() {
	i.Kind = strings.ToUpper(strings.TrimSpace(i.Kind))
	i.Name = strings.TrimSpace(i.Name)
	i.Currency = strings.ToUpper(strings.TrimSpace(i.Currency))
	if i.Currency == "" {
		i.Currency = "IDR"
	}
	for index := range i.Steps {
		i.Steps[index].Code = normalizeStepCode(i.Steps[index].Code)
		i.Steps[index].Name = strings.TrimSpace(i.Steps[index].Name)
	}
}

func (i RoutingInput) Validate(create bool) error {
	if create {
		if i.PartID == uuid.Nil {
			return invalid("Select the part this routing produces")
		}
		if i.Kind != "FG" && i.Kind != "RAW_MATERIAL" {
			return invalid("Output type must be a finished good or a process material")
		}
	}
	if i.Name == "" || len(i.Name) > 120 {
		return invalid("Routing name is required and cannot exceed 120 characters")
	}
	if len(i.Currency) != 3 {
		return invalid("Currency must be a three letter code")
	}
	if len(i.Steps) == 0 {
		return invalid("A routing needs at least one operation")
	}
	if len(i.Steps) > maxRoutingSteps {
		return invalid("A routing cannot exceed %d operations", maxRoutingSteps)
	}
	seen := map[string]bool{}
	for _, step := range i.Steps {
		if !validStepCode(step.Code) {
			return invalid("Operation code %q must be up to 20 letters, digits, dash or underscore", step.Code)
		}
		if seen[step.Code] {
			return invalid("Operation %s appears more than once", step.Code)
		}
		seen[step.Code] = true
		if step.Name == "" || len(step.Name) > 120 {
			return invalid("Operation %s requires a name of at most 120 characters", step.Code)
		}
		if step.Rate.IsNegative() || !step.Rate.Equal(step.Rate.Round(6)) || step.Rate.GreaterThanOrEqual(decimal.New(1, 14)) {
			return invalid("Operation %s needs a cost per piece of at most 6 decimal places", step.Code)
		}
	}
	return nil
}

func (i RoutingInput) TotalRate() decimal.Decimal {
	total := decimal.Zero
	for _, step := range i.Steps {
		total = total.Add(step.Rate)
	}
	return total
}

// canEditRouting keeps frozen routings intact: once an order snapshot points at
// a routing its steps and rates may no longer change.
func canEditRouting(linkedOrders int) bool { return linkedOrders == 0 }
