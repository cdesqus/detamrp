package production

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"sort"
	"strings"
	"time"
)

type MaterialUsage struct {
	MaterialID uuid.UUID       `json:"materialId"`
	Quantity   decimal.Decimal `json:"quantity"`
	PartNumber string          `json:"partNumber"`
	PartName   string          `json:"partName"`
	UnitCode   string          `json:"unitCode"`
	UnitPrice  decimal.Decimal `json:"unitPrice"`
	Cost       decimal.Decimal `json:"cost"`
}
type EntryInput struct {
	OrderID        uuid.UUID       `json:"orderId"`
	OperationID    uuid.UUID       `json:"operationId"`
	ProductionDate string          `json:"productionDate"`
	Shift          string          `json:"shift"`
	OperatorID     uuid.UUID       `json:"operatorId"`
	Processed      decimal.Decimal `json:"processed"`
	Good           decimal.Decimal `json:"good"`
	Rejected       decimal.Decimal `json:"rejected"`
	Notes          string          `json:"notes"`
	Version        int             `json:"version"`
	Materials      []MaterialUsage `json:"materials"`
}

func (i EntryInput) Validate(edit bool) error {
	if i.OrderID == uuid.Nil || i.OperationID == uuid.Nil {
		return invalid("Select a Production Order and routing operation")
	}
	if i.OperatorID == uuid.Nil {
		return invalid("Select an operator")
	}
	if _, e := parseDate(i.ProductionDate); e != nil {
		return invalid("A valid production date is required")
	}
	if strings.TrimSpace(i.Shift) == "" || len(i.Shift) > 30 {
		return invalid("Shift is required, up to 30 characters")
	}
	if !positiveQty(i.Processed) || i.Good.IsNegative() || i.Rejected.IsNegative() || !i.Good.Equal(i.Good.Round(6)) || !i.Rejected.Equal(i.Rejected.Round(6)) {
		return invalid("Processed quantity must be positive; quantities support at most 6 decimal places")
	}
	if e := (DailyEntry{Processed: i.Processed, Good: i.Good, Rejected: i.Rejected}).Validate(i.Processed); e != nil {
		return invalid("%s", e)
	}
	if len(i.Notes) > 4000 {
		return invalid("Notes cannot exceed 4000 characters")
	}
	if edit && i.Version < 1 {
		return invalid("Reload this entry before editing")
	}
	seen := map[uuid.UUID]bool{}
	for _, m := range i.Materials {
		if m.MaterialID == uuid.Nil || seen[m.MaterialID] || !positiveQty(m.Quantity) {
			return invalid("Material usage needs unique materials and positive quantities")
		}
		seen[m.MaterialID] = true
	}
	return nil
}

type Entry struct {
	ID          uuid.UUID `json:"id"`
	EntryNumber string    `json:"entryNumber"`
	EntryInput
	OrderNumber   string          `json:"orderNumber"`
	PartNumber    string          `json:"partNumber"`
	PartName      string          `json:"partName"`
	UnitCode      string          `json:"unitCode"`
	PlantName     string          `json:"plantName"`
	OperationName string          `json:"operationName"`
	OperationCode string          `json:"operationCode"`
	Sequence      int             `json:"sequence"`
	OperatorName  string          `json:"operatorName"`
	Status        string          `json:"status"`
	VoidReason    string          `json:"voidReason"`
	Currency      string          `json:"currency"`
	MaterialCost  decimal.Decimal `json:"materialCost"`
	ProcessCost   decimal.Decimal `json:"processCost"`
	ProcessRate   decimal.Decimal `json:"processRate"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	CreatedBy     string          `json:"createdBy"`
	PeriodClosed  bool            `json:"periodClosed"`
	EffectsLocked bool            `json:"effectsLocked"`
	CanCorrect    bool            `json:"canCorrect"`
	History       []PlanHistory   `json:"history"`
}
type OperatorOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
type EntryOptions struct {
	Orders        []Order          `json:"orders"`
	Operators     []OperatorOption `json:"operators"`
	ClosedPeriods []string         `json:"closedPeriods"`
}

type OperationDay struct {
	Date                string
	Supplied, Processed decimal.Decimal
}

func inputTimelineAvailable(days []OperationDay, date string, processed decimal.Decimal) bool {
	changes := map[string]decimal.Decimal{}
	for _, day := range days {
		changes[day.Date] = changes[day.Date].Add(day.Supplied).Sub(day.Processed)
	}
	changes[date] = changes[date].Sub(processed)
	dates := make([]string, 0, len(changes))
	for d := range changes {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	balance := decimal.Zero
	for _, d := range dates {
		balance = balance.Add(changes[d])
		if balance.IsNegative() {
			return false
		}
	}
	return true
}

// operationRemaining is what an operation may still process: its order target
// for the first operation, and for every later one whatever the operation
// before it has finished and not passed on yet, plus anything already staged.
func operationRemaining(o Order, index int) decimal.Decimal {
	if index > 0 {
		return decimal.Max(decimal.Zero, o.Operations[index].StagedQty.Add(o.Operations[index-1].OnHandQty))
	}
	return decimal.Max(decimal.Zero, o.PlannedQty.Sub(o.Operations[index].ProcessedQty))
}
func derivedOrderStatus(o Order) OrderStatus {
	if len(o.Operations) == 0 {
		return OrderInProgress
	}
	complete := true
	for i, op := range o.Operations {
		limit := o.PlannedQty
		if i > 0 {
			limit = o.Operations[i-1].GoodQty
		}
		if op.ProcessedQty.LessThan(limit) {
			complete = false
		}
	}
	if complete {
		return OrderCompleted
	}
	if o.Operations[len(o.Operations)-1].GoodQty.IsPositive() {
		return OrderPartial
	}
	return OrderInProgress
}
