package production

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

type PlanStatus string

const (
	PlanDraft    PlanStatus = "DRAFT"
	PlanApproved PlanStatus = "APPROVED"
	PlanClosed   PlanStatus = "CLOSED"
)

type OrderStatus string

const (
	OrderReleased   OrderStatus = "RELEASED"
	OrderInProgress OrderStatus = "IN_PROGRESS"
	OrderPartial    OrderStatus = "PARTIAL"
	OrderCompleted  OrderStatus = "COMPLETED"
	OrderCancelled  OrderStatus = "CANCELLED"
)

// CanTransition centralises the workflow rules used by the API and UI.
func CanTransition(from, to OrderStatus) bool {
	switch from {
	case OrderReleased:
		return to == OrderInProgress || to == OrderCancelled
	case OrderInProgress:
		return to == OrderPartial || to == OrderCompleted || to == OrderCancelled
	case OrderPartial:
		return to == OrderInProgress || to == OrderCompleted || to == OrderCancelled
	default:
		return false
	}
}

type DailyEntry struct {
	Processed decimal.Decimal
	Good      decimal.Decimal
	Rejected  decimal.Decimal
}

func (e DailyEntry) Validate(remaining decimal.Decimal) error {
	if !e.Processed.GreaterThan(decimal.Zero) {
		return errors.New("processed quantity must be greater than zero")
	}
	if e.Good.IsNegative() || e.Rejected.IsNegative() {
		return errors.New("good and rejected quantities cannot be negative")
	}
	if e.Good.Add(e.Rejected).GreaterThan(e.Processed) {
		return errors.New("good plus rejected quantity cannot exceed processed quantity")
	}
	if e.Processed.GreaterThan(remaining) {
		return fmt.Errorf("processed quantity exceeds remaining quantity (%s)", remaining.String())
	}
	return nil
}
