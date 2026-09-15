package production

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type ValidationError string

func (e ValidationError) Error() string         { return string(e) }
func invalid(format string, args ...any) error  { return ValidationError(fmt.Sprintf(format, args...)) }
func parseDate(value string) (time.Time, error) { return time.Parse("2006-01-02", value) }
func positiveQty(q decimal.Decimal) bool {
	return q.IsPositive() && q.Equal(q.Round(6)) && q.LessThan(decimal.New(1, 14))
}

type OrderInput struct {
	PlanLineID uuid.UUID       `json:"planLineId"`
	PlannedQty decimal.Decimal `json:"plannedQty"`
	DueDate    string          `json:"dueDate"`
	Notes      string          `json:"notes"`
}

func (i OrderInput) Validate(create bool) error {
	if create && i.PlanLineID == uuid.Nil {
		return invalid("Select a planning line")
	}
	if !positiveQty(i.PlannedQty) {
		return invalid("Quantity must be positive with at most 6 decimal places")
	}
	if _, e := time.Parse("2006-01-02", i.DueDate); e != nil {
		return invalid("A valid due date is required")
	}
	return nil
}
func canEditOrder(s OrderStatus, entries int) bool {
	return entries == 0 && (s == OrderReleased || s == OrderInProgress)
}

type MaterialSnapshot struct {
	ID         uuid.UUID       `json:"id"`
	PartNumber string          `json:"partNumber"`
	PartName   string          `json:"partName"`
	UnitCode   string          `json:"unitCode"`
	UsageQty   decimal.Decimal `json:"usageQty"`
	UnitPrice  decimal.Decimal `json:"unitPrice"`
	Currency   string          `json:"currency"`
}
type Operation struct {
	ID           uuid.UUID       `json:"id"`
	Code         string          `json:"code"`
	Name         string          `json:"name"`
	Sequence     int             `json:"sequence"`
	Rate         decimal.Decimal `json:"rate"`
	PlannedQty   decimal.Decimal `json:"plannedQty"`
	ProcessedQty decimal.Decimal `json:"processedQty"`
	GoodQty      decimal.Decimal `json:"goodQty"`
	RejectQty    decimal.Decimal `json:"rejectQty"`
	WIPQty       decimal.Decimal `json:"wipQty"`
	OnHandQty    decimal.Decimal `json:"onHandQty"`
	StagedQty    decimal.Decimal `json:"stagedQty"`
}
type Order struct {
	ID               uuid.UUID          `json:"id"`
	OrderNumber      string             `json:"orderNumber"`
	PlanID           uuid.UUID          `json:"planId"`
	PlanLineID       uuid.UUID          `json:"planLineId"`
	PlanNumber       string             `json:"planNumber"`
	RoutingID        uuid.UUID          `json:"routingId"`
	RoutingRevision  int                `json:"routingRevision"`
	PartNumber       string             `json:"partNumber"`
	PartName         string             `json:"partName"`
	UnitCode         string             `json:"unitCode"`
	PlantName        string             `json:"plantName"`
	PlannedQty       decimal.Decimal    `json:"plannedQty"`
	PeriodStart      string             `json:"periodStart"`
	DueDate          string             `json:"dueDate"`
	Status           OrderStatus        `json:"status"`
	Notes            string             `json:"notes"`
	BOMRevision      int                `json:"bomRevision"`
	Currency         string             `json:"currency"`
	MaterialEstimate decimal.Decimal    `json:"materialEstimate"`
	ProcessEstimate  decimal.Decimal    `json:"processEstimate"`
	ActualGood       decimal.Decimal    `json:"actualGood"`
	RejectQty        decimal.Decimal    `json:"rejectQty"`
	MaterialCost     decimal.Decimal    `json:"materialCost"`
	ProcessCost      decimal.Decimal    `json:"processCost"`
	WIPQty           decimal.Decimal    `json:"wipQty"`
	EntryCount       int                `json:"entryCount"`
	UpdatedAt        time.Time          `json:"updatedAt"`
	CreatedBy        string             `json:"createdBy"`
	Operations       []Operation        `json:"operations"`
	Materials        []MaterialSnapshot `json:"materials"`
	History          []PlanHistory      `json:"history"`
}
type OrderLineOption struct {
	ID           uuid.UUID       `json:"id"`
	PlanID       uuid.UUID       `json:"planId"`
	PlanNumber   string          `json:"planNumber"`
	PartNumber   string          `json:"partNumber"`
	PartName     string          `json:"partName"`
	UnitCode     string          `json:"unitCode"`
	PlantName    string          `json:"plantName"`
	PeriodStart  string          `json:"periodStart"`
	PeriodEnd    string          `json:"periodEnd"`
	RemainingQty decimal.Decimal `json:"remainingQty"`
}
type OrderOptions struct {
	Lines  []OrderLineOption `json:"lines"`
	Parts  []PartOption      `json:"parts"`
	Plants []PlantOption     `json:"plants"`
}
