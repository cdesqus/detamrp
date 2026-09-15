package production

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type MovementType string

const (
	MovementReceipt     MovementType = "RECEIPT"
	MovementTransfer    MovementType = "TRANSFER"
	MovementConsumption MovementType = "CONSUMPTION"
)

// WIPMovement is one signed row of the append-only work-in-progress ledger.
type WIPMovement struct {
	ID                     uuid.UUID       `json:"id"`
	OrderID                uuid.UUID       `json:"orderId"`
	OrderNumber            string          `json:"orderNumber"`
	Type                   MovementType    `json:"type"`
	SourceOperationID      uuid.UUID       `json:"sourceOperationId"`
	SourceCode             string          `json:"sourceCode"`
	DestinationOperationID uuid.UUID       `json:"destinationOperationId"`
	DestinationCode        string          `json:"destinationCode"`
	Quantity               decimal.Decimal `json:"quantity"`
	UnitCost               decimal.Decimal `json:"unitCost"`
	TotalCost              decimal.Decimal `json:"totalCost"`
	Currency               string          `json:"currency"`
	EntryID                uuid.UUID       `json:"entryId"`
	EntryNumber            string          `json:"entryNumber"`
	LotID                  uuid.UUID       `json:"lotId"`
	ReversesID             uuid.UUID       `json:"reversesId"`
	Reversed               bool            `json:"reversed"`
	MovementDate           string          `json:"movementDate"`
	Notes                  string          `json:"notes"`
	CreatedBy              string          `json:"createdBy"`
	CreatedAt              time.Time       `json:"createdAt"`
	CanReverse             bool            `json:"canReverse"`
}

// WIPOperationBalance reconciles ledger movements against the recorded entries
// of one routing operation.
type WIPOperationBalance struct {
	OperationID    uuid.UUID       `json:"operationId"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Sequence       int             `json:"sequence"`
	Final          bool            `json:"final"`
	GoodQty        decimal.Decimal `json:"goodQty"`
	ProcessedQty   decimal.Decimal `json:"processedQty"`
	Received       decimal.Decimal `json:"received"`
	TransferredOut decimal.Decimal `json:"transferredOut"`
	TransferredIn  decimal.Decimal `json:"transferredIn"`
	Consumed       decimal.Decimal `json:"consumed"`
	OnHand         decimal.Decimal `json:"onHand"`
	Staged         decimal.Decimal `json:"staged"`
	Balance        decimal.Decimal `json:"balance"`
	Value          decimal.Decimal `json:"value"`
	Reconciled     bool            `json:"reconciled"`
	Difference     decimal.Decimal `json:"difference"`
}

type WIPOrderBalance struct {
	OrderID     uuid.UUID             `json:"orderId"`
	OrderNumber string                `json:"orderNumber"`
	PlanNumber  string                `json:"planNumber"`
	PartNumber  string                `json:"partNumber"`
	PartName    string                `json:"partName"`
	UnitCode    string                `json:"unitCode"`
	PlantName   string                `json:"plantName"`
	Status      OrderStatus           `json:"status"`
	Currency    string                `json:"currency"`
	PlannedQty  decimal.Decimal       `json:"plannedQty"`
	Operations  []WIPOperationBalance `json:"operations"`
	TotalQty    decimal.Decimal       `json:"totalQty"`
	TotalValue  decimal.Decimal       `json:"totalValue"`
	Reconciled  bool                  `json:"reconciled"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	Movements   []WIPMovement         `json:"movements"`
}

type TransferInput struct {
	OrderID           uuid.UUID       `json:"orderId"`
	SourceOperationID uuid.UUID       `json:"sourceOperationId"`
	Quantity          decimal.Decimal `json:"quantity"`
	MovementDate      string          `json:"movementDate"`
	Notes             string          `json:"notes"`
}

func (i TransferInput) Validate() error {
	if i.OrderID == uuid.Nil || i.SourceOperationID == uuid.Nil {
		return invalid("Select a production order and the operation holding the WIP")
	}
	if !positiveQty(i.Quantity) {
		return invalid("Transfer quantity must be positive with at most 6 decimal places")
	}
	if _, e := parseDate(i.MovementDate); e != nil {
		return invalid("A valid movement date is required")
	}
	if len(i.Notes) > 2000 {
		return invalid("Notes cannot exceed 2000 characters")
	}
	return nil
}

// Lot is an open costed quantity a transfer or consumption can draw from.
type Lot struct {
	ID        uuid.UUID
	Remaining decimal.Decimal
	UnitCost  decimal.Decimal
	Date      string
}

// Allocation assigns part of one lot to a new movement.
type Allocation struct {
	LotID    uuid.UUID
	Quantity decimal.Decimal
	UnitCost decimal.Decimal
}

// allocateFIFO draws quantity from the oldest lots first so movement costs stay
// traceable to the entry that produced them.
func allocateFIFO(lots []Lot, quantity decimal.Decimal) ([]Allocation, error) {
	remaining := quantity
	allocations := []Allocation{}
	for _, lot := range lots {
		if !remaining.IsPositive() {
			break
		}
		if !lot.Remaining.IsPositive() {
			continue
		}
		take := decimal.Min(remaining, lot.Remaining)
		allocations = append(allocations, Allocation{LotID: lot.ID, Quantity: take, UnitCost: lot.UnitCost})
		remaining = remaining.Sub(take)
	}
	if remaining.IsPositive() {
		return nil, invalid("Only %s is available in this WIP balance", quantity.Sub(remaining))
	}
	return allocations, nil
}

func allocationCost(allocations []Allocation) decimal.Decimal {
	total := decimal.Zero
	for _, allocation := range allocations {
		total = total.Add(allocation.Quantity.Mul(allocation.UnitCost))
	}
	return total.Round(6)
}

// unitCost spreads a batch cost across its good output; rejected pieces leave
// their cost with the good ones instead of disappearing from the order.
func unitCost(total, quantity decimal.Decimal) decimal.Decimal {
	if !quantity.IsPositive() {
		return decimal.Zero
	}
	return total.Div(quantity).Round(6)
}

// reconcile checks the ledger against the entries it was derived from: every
// good piece of a non-final operation must be received, and every processed
// piece after the first operation must be consumed from staged WIP.
func (b *WIPOperationBalance) reconcile() {
	b.OnHand = b.Received.Sub(b.TransferredOut)
	b.Staged = b.TransferredIn.Sub(b.Consumed)
	b.Balance = b.OnHand.Add(b.Staged)
	expectedReceipt := b.GoodQty
	if b.Final {
		expectedReceipt = decimal.Zero
	}
	expectedConsumption := b.ProcessedQty
	if b.Sequence <= 1 {
		expectedConsumption = decimal.Zero
	}
	b.Difference = b.Received.Sub(expectedReceipt).Add(b.Consumed.Sub(expectedConsumption))
	b.Reconciled = b.Difference.IsZero() && !b.OnHand.IsNegative() && !b.Staged.IsNegative()
}

func movementLabel(m WIPMovement) string {
	if m.ReversesID != uuid.Nil {
		return "REVERSAL"
	}
	return string(m.Type)
}

func trimNotes(value string) string { return strings.TrimSpace(value) }
