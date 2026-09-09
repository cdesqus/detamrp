package salesmaster

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestFinishedGoodInputValidWithoutCustomer(t *testing.T) {
	input := FinishedGoodInput{
		ItemCode:   " fg-100 ",
		Name:       " Finished Assembly ",
		BaseUnitID: uuid.New(),
		SalesPrice: decimal.RequireFromString("125000.50"),
		Currency:   " idr ",
	}

	if fields := input.NormalizeAndValidate(); len(fields) != 0 {
		t.Fatalf("validation fields = %#v, want none", fields)
	}
	if input.ItemCode != "FG-100" || input.Name != "Finished Assembly" || input.Currency != "IDR" {
		t.Fatalf("normalized input = %#v", input)
	}

	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	var shape map[string]any
	if err := json.Unmarshal(payload, &shape); err != nil {
		t.Fatalf("decode input: %v", err)
	}
	if _, exists := shape["customerId"]; exists {
		t.Fatal("finished-good input unexpectedly contains customerId")
	}
}

func TestFinishedGoodInputRejectsNonpositiveSalesPrice(t *testing.T) {
	for _, price := range []decimal.Decimal{decimal.Zero, decimal.NewFromInt(-1)} {
		input := validFinishedGoodInput()
		input.SalesPrice = price
		if got := input.NormalizeAndValidate()["salesPrice"]; got != "Sales Price must be greater than zero" {
			t.Fatalf("price %s validation = %q", price, got)
		}
	}
}

func TestFinishedGoodInputRequiresBaseUnit(t *testing.T) {
	input := validFinishedGoodInput()
	input.BaseUnitID = uuid.Nil
	if got := input.NormalizeAndValidate()["baseUnitId"]; got != "Base Unit is required" {
		t.Fatalf("base unit validation = %q", got)
	}
}

func TestFinishedGoodInputRejectsUnsupportedCurrency(t *testing.T) {
	input := validFinishedGoodInput()
	input.Currency = "GBP"
	if got := input.NormalizeAndValidate()["currency"]; got != "Select a supported currency" {
		t.Fatalf("currency validation = %q", got)
	}
}

func validFinishedGoodInput() FinishedGoodInput {
	return FinishedGoodInput{
		ItemCode:   "FG-100",
		Name:       "Finished Assembly",
		BaseUnitID: uuid.New(),
		SalesPrice: decimal.RequireFromString("100.000000"),
		Currency:   "IDR",
	}
}
