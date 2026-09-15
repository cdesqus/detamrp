package production

import (
	"time"

	"github.com/shopspring/decimal"
)

// DashboardFilter bounds the daily entries a dashboard reads. Stock figures such
// as work in progress are always current, never filtered by date.
type DashboardFilter struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ParseDashboardFilter defaults to the running month, which is also the period
// production is closed by.
func ParseDashboardFilter(from, to string, now time.Time) (DashboardFilter, error) {
	filter := DashboardFilter{From: from, To: to}
	if filter.From == "" {
		filter.From = now.Format("2006-01") + "-01"
	}
	if filter.To == "" {
		filter.To = now.Format("2006-01-02")
	}
	start, e := parseDate(filter.From)
	if e != nil {
		return filter, invalid("A valid start date is required")
	}
	end, e := parseDate(filter.To)
	if e != nil {
		return filter, invalid("A valid end date is required")
	}
	if end.Before(start) {
		return filter, invalid("The end date cannot precede the start date")
	}
	if end.Sub(start) > 366*24*time.Hour {
		return filter, invalid("Choose a range of at most one year")
	}
	return filter, nil
}

type DashboardTotals struct {
	PlannedQty     decimal.Decimal `json:"plannedQty"`
	ProcessedQty   decimal.Decimal `json:"processedQty"`
	GoodQty        decimal.Decimal `json:"goodQty"`
	RejectQty      decimal.Decimal `json:"rejectQty"`
	RejectRate     decimal.Decimal `json:"rejectRate"`
	Achievement    decimal.Decimal `json:"achievement"`
	WIPQty         decimal.Decimal `json:"wipQty"`
	WIPValue       decimal.Decimal `json:"wipValue"`
	MaterialActual decimal.Decimal `json:"materialActual"`
	ProcessActual  decimal.Decimal `json:"processActual"`
	TotalActual    decimal.Decimal `json:"totalActual"`
	Entries        int             `json:"entries"`
	ActiveOrders   int             `json:"activeOrders"`
	OpenOrders     int             `json:"openOrders"`
	CompletedOrder int             `json:"completedOrders"`
	Currency       string          `json:"currency"`
}

// PartPerformance compares what was planned for a part with what it produced.
type PartPerformance struct {
	PartNumber  string          `json:"partNumber"`
	PartName    string          `json:"partName"`
	UnitCode    string          `json:"unitCode"`
	Orders      int             `json:"orders"`
	PlannedQty  decimal.Decimal `json:"plannedQty"`
	GoodQty     decimal.Decimal `json:"goodQty"`
	RejectQty   decimal.Decimal `json:"rejectQty"`
	Achievement decimal.Decimal `json:"achievement"`
}

// ProcessWIP is the current balance waiting at one operation code, valued in
// the reporting currency.
type ProcessWIP struct {
	Currency string          `json:"currency"`
	Code     string          `json:"code"`
	Quantity decimal.Decimal `json:"quantity"`
	Value    decimal.Decimal `json:"value"`
}

// OperationQuality is the reject performance of one operation code.
type OperationQuality struct {
	Code         string          `json:"code"`
	ProcessedQty decimal.Decimal `json:"processedQty"`
	GoodQty      decimal.Decimal `json:"goodQty"`
	RejectQty    decimal.Decimal `json:"rejectQty"`
	RejectRate   decimal.Decimal `json:"rejectRate"`
}

// PartCost groups booked costs and good output in the window by part and currency.
// FinishedCost, LifetimeGoodQty and CostPerPiece are cumulative as of now for
// orders with postings in the window; live WIP is deducted only from lifetime costs.
type PartCost struct {
	PartNumber      string          `json:"partNumber"`
	PartName        string          `json:"partName"`
	Orders          int             `json:"orders"`
	GoodQty         decimal.Decimal `json:"goodQty"`
	MaterialActual  decimal.Decimal `json:"materialActual"`
	ProcessActual   decimal.Decimal `json:"processActual"`
	TotalActual     decimal.Decimal `json:"totalActual"`
	CostPerPiece    decimal.Decimal `json:"costPerPiece"`
	LifetimeGoodQty decimal.Decimal `json:"lifetimeGoodQty"`
	FinishedCost    decimal.Decimal `json:"finishedCost"`
	Currency        string          `json:"currency"`
}

type unusedCurrencyCost struct {
	Currency       string          `json:"currency"`
	MaterialActual decimal.Decimal `json:"materialActual"`
	ProcessActual  decimal.Decimal `json:"processActual"`
	TotalActual    decimal.Decimal `json:"totalActual"`
	WIPValue       decimal.Decimal `json:"wipValue"`
}

type ProductionDashboard struct {
	Filter      DashboardFilter    `json:"filter"`
	Totals      DashboardTotals    `json:"totals"`
	Parts       []PartPerformance  `json:"parts"`
	Processes   []ProcessWIP       `json:"processes"`
	Quality     []OperationQuality `json:"quality"`
	Costs       []PartCost         `json:"costs"`
	GeneratedAt time.Time          `json:"generatedAt"`
}

// ratio expresses part of total as a percentage, guarding the empty case.
func ratio(part, total decimal.Decimal) decimal.Decimal {
	if !total.IsPositive() {
		return decimal.Zero
	}
	return part.Div(total).Mul(decimal.NewFromInt(100)).Round(2)
}

func (t *DashboardTotals) summarise() {
	t.RejectRate = ratio(t.RejectQty, t.ProcessedQty)
	t.Achievement = ratio(t.GoodQty, t.PlannedQty)
	t.TotalActual = t.MaterialActual.Add(t.ProcessActual)
}
