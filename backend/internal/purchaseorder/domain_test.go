package purchaseorder

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestOrderInputRejectsExpectedDeliveryBeforeOrderDate(t *testing.T) {
	input := validOrderInput()
	input.ExpectedDeliveryDate = input.OrderDate.AddDate(0, 0, -1)

	errs := input.NormalizeAndValidate(false)
	if errs["expectedDeliveryDate"] == "" {
		t.Fatalf("expected date error, got %#v", errs)
	}
}

func TestOrderInputRequiresMaterialWhenSubmitting(t *testing.T) {
	input := validOrderInput()
	input.Lines = nil

	errs := input.NormalizeAndValidate(true)
	if errs["lines"] == "" {
		t.Fatalf("expected material error, got %#v", errs)
	}
}

func TestOrderInputRejectsRepeatedMaterials(t *testing.T) {
	input := validOrderInput()
	input.Lines = append(input.Lines, input.Lines[0])

	errs := input.NormalizeAndValidate(false)
	if errs["lines[1].rawMaterialId"] == "" {
		t.Fatalf("expected duplicate material error, got %#v", errs)
	}
}

func TestOrderInputRejectsNonPositiveOrFractionalKanbanCount(t *testing.T) {
	input := validOrderInput()
	input.Lines[0].TotalKanban = decimal.RequireFromString("1.5")
	if errs := input.NormalizeAndValidate(false); errs["lines[0].totalKanban"] == "" {
		t.Fatalf("expected fractional Kanban error, got %#v", errs)
	}

	input.Lines[0].TotalKanban = decimal.Zero
	if errs := input.NormalizeAndValidate(false); errs["lines[0].totalKanban"] == "" {
		t.Fatalf("expected zero Kanban error, got %#v", errs)
	}
}

func TestOrderLineCalculatesExactDecimalAmounts(t *testing.T) {
	line := OrderLine{
		QtyPerKanbanSnapshot: decimal.RequireFromString("0.333333"),
		TotalKanban:          decimal.NewFromInt(3),
		UnitPriceSnapshot:    decimal.RequireFromString("19.999999"),
	}

	line.Recalculate()
	if got, want := line.OrderedBaseQty.StringFixed(6), "0.999999"; got != want || line.OrderedBaseQty.Exponent() != -6 {
		t.Fatalf("ordered quantity = %s, want %s", got, want)
	}
	if got, want := line.LineTotal, decimal.RequireFromString("19.999979"); !got.Equal(want) || got.Exponent() != -6 {
		t.Fatalf("line total = %s, want %s", got, want)
	}
}

func TestOrderRecalculatesExactTotalFromLines(t *testing.T) {
	order := Order{Lines: []OrderLine{
		{QtyPerKanbanSnapshot: decimal.RequireFromString("0.1"), TotalKanban: decimal.NewFromInt(3), UnitPriceSnapshot: decimal.RequireFromString("0.2")},
		{QtyPerKanbanSnapshot: decimal.RequireFromString("0.2"), TotalKanban: decimal.NewFromInt(7), UnitPriceSnapshot: decimal.RequireFromString("0.3")},
	}}

	order.RecalculateTotals()
	if got, want := order.TotalAmount, decimal.RequireFromString("0.480000"); !got.Equal(want) || got.Exponent() != -6 {
		t.Fatalf("order total = %s, want %s", got, want)
	}
}

func TestOrderLineRoundsHalfAwayFromZeroAtSixDecimalPlaces(t *testing.T) {
	line := OrderLine{
		QtyPerKanbanSnapshot: decimal.RequireFromString("0.000001"),
		TotalKanban:          decimal.NewFromInt(1),
		UnitPriceSnapshot:    decimal.RequireFromString("0.500000"),
	}

	line.Recalculate()
	if got, want := line.LineTotal, decimal.RequireFromString("0.000001"); !got.Equal(want) || got.Exponent() != -6 {
		t.Fatalf("line total = %s, want %s", got, want)
	}
}

func TestOrderInputRequiresThreeASCIIAlphabeticCurrencyLetters(t *testing.T) {
	for _, currency := range []string{"US1", "€UR", "USD1"} {
		input := validOrderInput()
		input.Currency = currency
		if errs := input.NormalizeAndValidate(false); errs["currency"] == "" {
			t.Fatalf("currency %q was accepted", currency)
		}
	}

	input := validOrderInput()
	input.Currency = " idr "
	if errs := input.NormalizeAndValidate(false); errs["currency"] != "" || input.Currency != "IDR" {
		t.Fatalf("valid currency was not normalized: input=%#v errors=%#v", input, errs)
	}
}

func validOrderInput() OrderInput {
	return OrderInput{
		SupplierID:           uuid.New(),
		OrderDate:            time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC),
		ExpectedDeliveryDate: time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC),
		Currency:             " idr ",
		Lines:                []LineInput{{RawMaterialID: uuid.New(), TotalKanban: decimal.NewFromInt(1)}},
	}
}
