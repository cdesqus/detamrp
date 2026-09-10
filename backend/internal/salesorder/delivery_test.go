package salesorder

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

func TestDeliveryInputRequiresAtLeastOnePositiveLine(t *testing.T) {
	input := DeliveryInput{Lines: []DeliveryLineInput{{SalesOrderLineID: uuid.Nil, Quantity: decimal.Zero}}}
	fields := input.NormalizeAndValidate()
	if fields["lines[0].salesOrderLineId"] == "" || fields["lines[0].quantity"] == "" {
		t.Fatal(fields)
	}
}
