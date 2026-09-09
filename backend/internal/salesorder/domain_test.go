package salesorder

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

func TestInputRequiresCustomerAndUniquePositiveFinishedGoods(t *testing.T) {
	fg := uuid.New()
	input := Input{CustomerID: uuid.Nil, Lines: []LineInput{{FinishedGoodID: fg, Quantity: decimal.NewFromInt(1)}, {FinishedGoodID: fg, Quantity: decimal.Zero}}}
	fields := input.NormalizeAndValidate()
	if fields["customerId"] == "" || fields["lines[1].finishedGoodId"] == "" || fields["lines[1].quantity"] == "" {
		t.Fatal(fields)
	}
}
func TestInputAcceptsMultipleFinishedGoods(t *testing.T) {
	input := Input{CustomerID: uuid.New(), Lines: []LineInput{{FinishedGoodID: uuid.New(), Quantity: decimal.NewFromInt(3)}, {FinishedGoodID: uuid.New(), Quantity: decimal.RequireFromString("2.5")}}}
	if fields := input.NormalizeAndValidate(); len(fields) != 0 {
		t.Fatal(fields)
	}
}
