package production

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OperationCost is the actual process cost booked at one routing operation.
type OperationCost struct {
	OperationID  uuid.UUID       `json:"operationId"`
	Code         string          `json:"code"`
	Name         string          `json:"name"`
	Sequence     int             `json:"sequence"`
	Final        bool            `json:"final"`
	ProcessedQty decimal.Decimal `json:"processedQty"`
	GoodQty      decimal.Decimal `json:"goodQty"`
	RejectQty    decimal.Decimal `json:"rejectQty"`
	RateSnapshot decimal.Decimal `json:"rateSnapshot"`
	EstimateCost decimal.Decimal `json:"estimateCost"`
	ActualCost   decimal.Decimal `json:"actualCost"`
	CostPerPiece decimal.Decimal `json:"costPerPiece"`
	Entries      int             `json:"entries"`
}

// MaterialCost aggregates the material a production order actually consumed at
// the prices frozen when each entry was posted.
type MaterialCost struct {
	MaterialID   uuid.UUID       `json:"materialId"`
	PartNumber   string          `json:"partNumber"`
	PartName     string          `json:"partName"`
	UnitCode     string          `json:"unitCode"`
	UsedQty      decimal.Decimal `json:"usedQty"`
	StandardQty  decimal.Decimal `json:"standardQty"`
	QtyVariance  decimal.Decimal `json:"qtyVariance"`
	AveragePrice decimal.Decimal `json:"averagePrice"`
	ActualCost   decimal.Decimal `json:"actualCost"`
}

// OrderCost is the actual cost of one production order: material plus every
// process step, less whatever cost is still sitting in work in progress.
type OrderCost struct {
	OrderID          uuid.UUID       `json:"orderId"`
	OrderNumber      string          `json:"orderNumber"`
	PlanNumber       string          `json:"planNumber"`
	PartNumber       string          `json:"partNumber"`
	PartName         string          `json:"partName"`
	UnitCode         string          `json:"unitCode"`
	PlantName        string          `json:"plantName"`
	Status           OrderStatus     `json:"status"`
	Currency         string          `json:"currency"`
	PeriodStart      string          `json:"periodStart"`
	DueDate          string          `json:"dueDate"`
	PlannedQty       decimal.Decimal `json:"plannedQty"`
	GoodQty          decimal.Decimal `json:"goodQty"`
	RejectQty        decimal.Decimal `json:"rejectQty"`
	MaterialEstimate decimal.Decimal `json:"materialEstimate"`
	ProcessEstimate  decimal.Decimal `json:"processEstimate"`
	TotalEstimate    decimal.Decimal `json:"totalEstimate"`
	MaterialActual   decimal.Decimal `json:"materialActual"`
	ProcessActual    decimal.Decimal `json:"processActual"`
	TotalActual      decimal.Decimal `json:"totalActual"`
	WIPValue         decimal.Decimal `json:"wipValue"`
	FinishedCost     decimal.Decimal `json:"finishedCost"`
	PlannedUnitCost  decimal.Decimal `json:"plannedUnitCost"`
	ActualUnitCost   decimal.Decimal `json:"actualUnitCost"`
	Variance         decimal.Decimal `json:"variance"`
	VariancePercent  decimal.Decimal `json:"variancePercent"`
	EntryCount       int             `json:"entryCount"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	Operations       []OperationCost `json:"operations"`
	Materials        []MaterialCost  `json:"materials"`
}

// PeriodCost totals one production month and says whether it is still open.
type PeriodCost struct {
	Period         string          `json:"period"`
	Closed         bool            `json:"closed"`
	ClosedBy       string          `json:"closedBy"`
	Orders         int             `json:"orders"`
	Entries        int             `json:"entries"`
	ProcessedQty   decimal.Decimal `json:"processedQty"`
	GoodQty        decimal.Decimal `json:"goodQty"`
	RejectQty      decimal.Decimal `json:"rejectQty"`
	MaterialActual decimal.Decimal `json:"materialActual"`
	ProcessActual  decimal.Decimal `json:"processActual"`
	TotalActual    decimal.Decimal `json:"totalActual"`
	Currency       string          `json:"currency"`
}

// finishedCost is what left the order as finished output: everything booked on
// it minus the cost still held in work in progress.
func finishedCost(totalActual, wipValue decimal.Decimal) decimal.Decimal {
	cost := totalActual.Sub(wipValue)
	if cost.IsNegative() {
		return decimal.Zero
	}
	return cost.Round(6)
}

// perUnit divides a cost over produced pieces, tolerating orders that have not
// produced anything yet.
func perUnit(cost, quantity decimal.Decimal) decimal.Decimal {
	if !quantity.IsPositive() {
		return decimal.Zero
	}
	return cost.Div(quantity).Round(6)
}

// costVariance compares actual unit cost with the estimate frozen at release,
// so a cheaper run reads as a negative variance.
func costVariance(actualUnit, plannedUnit decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	difference := actualUnit.Sub(plannedUnit).Round(6)
	if !plannedUnit.IsPositive() {
		return difference, decimal.Zero
	}
	return difference, difference.Div(plannedUnit).Mul(decimal.NewFromInt(100)).Round(2)
}

// convertTo restates every amount in the reporting currency. Ratios such as the
// variance percentage are unaffected because both sides scale together.
func (c *OrderCost) convertTo(rates Rates) {
	source := c.Currency
	for _, field := range []*decimal.Decimal{&c.MaterialEstimate, &c.ProcessEstimate, &c.TotalEstimate,
		&c.MaterialActual, &c.ProcessActual, &c.TotalActual, &c.WIPValue, &c.FinishedCost,
		&c.PlannedUnitCost, &c.ActualUnitCost, &c.Variance} {
		*field = rates.mustConvert(*field, source)
	}
	for index := range c.Operations {
		operation := &c.Operations[index]
		operation.RateSnapshot = rates.mustConvert(operation.RateSnapshot, source)
		operation.EstimateCost = rates.mustConvert(operation.EstimateCost, source)
		operation.ActualCost = rates.mustConvert(operation.ActualCost, source)
		operation.CostPerPiece = rates.mustConvert(operation.CostPerPiece, source)
	}
	for index := range c.Materials {
		material := &c.Materials[index]
		material.AveragePrice = rates.mustConvert(material.AveragePrice, source)
		material.ActualCost = rates.mustConvert(material.ActualCost, source)
	}
	c.Currency = ReportingCurrency
}

func (c *OrderCost) summarise() {
	c.TotalEstimate = c.MaterialEstimate.Add(c.ProcessEstimate)
	c.TotalActual = c.MaterialActual.Add(c.ProcessActual)
	c.WIPValue = c.WIPValue.Round(6)
	c.FinishedCost = finishedCost(c.TotalActual, c.WIPValue)
	c.PlannedUnitCost = perUnit(c.TotalEstimate, c.PlannedQty)
	c.ActualUnitCost = perUnit(c.FinishedCost, c.GoodQty)
	c.Variance, c.VariancePercent = costVariance(c.ActualUnitCost, c.PlannedUnitCost)
}
