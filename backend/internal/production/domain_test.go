package production

import (
	"testing"
	"github.com/shopspring/decimal"
)

func TestDailyEntryValidate(t *testing.T) {
	entry := DailyEntry{Processed: decimal.NewFromInt(700), Good: decimal.NewFromInt(680), Rejected: decimal.NewFromInt(20)}
	if err := entry.Validate(decimal.NewFromInt(1000)); err != nil { t.Fatal(err) }
	entry.Processed = decimal.NewFromInt(1100)
	if err := entry.Validate(decimal.NewFromInt(1000)); err == nil { t.Fatal("expected remaining quantity validation") }
}

func TestCanTransition(t *testing.T) {
	if !CanTransition(OrderReleased, OrderInProgress) { t.Fatal("released should start") }
	if CanTransition(OrderCompleted, OrderInProgress) { t.Fatal("completed order must be immutable") }
}
