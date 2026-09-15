package production

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestRatesResolveTheReportingCurrency(t *testing.T) {
	rates := Rates{"USD": decimal.NewFromInt(16000)}
	for _, currency := range []string{"IDR", "idr", ""} {
		value, ok := rates.rate(currency)
		if !ok || !value.Equal(decimal.NewFromInt(1)) {
			t.Fatalf("%q should report one to one: %s %v", currency, value, ok)
		}
	}
	if _, ok := rates.rate("EUR"); ok {
		t.Fatal("unknown currency reported a rate")
	}
}

func TestConvertRefusesToGuessAMissingRate(t *testing.T) {
	rates := Rates{"USD": decimal.NewFromInt(16000)}
	converted, e := rates.convert(decimal.NewFromInt(25), "USD")
	if e != nil || !converted.Equal(decimal.NewFromInt(400000)) {
		t.Fatalf("conversion: %s %v", converted, e)
	}
	if kept, e := rates.convert(decimal.NewFromInt(150), "IDR"); e != nil || !kept.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("rupiah should pass through: %s %v", kept, e)
	}
	if _, e = rates.convert(decimal.NewFromInt(10), "EUR"); e == nil {
		t.Fatal("an unrated currency was converted anyway")
	}
	if dropped := rates.mustConvert(decimal.NewFromInt(10), "EUR"); !dropped.IsZero() {
		t.Fatalf("an unrated amount was invented: %s", dropped)
	}
}

func TestOrderCostConvertsWithoutDistortingRatios(t *testing.T) {
	c := OrderCost{
		Currency: "USD", PlannedQty: decimal.NewFromInt(100), GoodQty: decimal.NewFromInt(40),
		MaterialEstimate: decimal.NewFromInt(10), ProcessEstimate: decimal.NewFromInt(5),
		MaterialActual: decimal.NewFromInt(12), ProcessActual: decimal.NewFromInt(6),
		Operations: []OperationCost{{Code: "STAMPING", ActualCost: decimal.NewFromInt(6), RateSnapshot: decimal.NewFromInt(2)}},
		Materials:  []MaterialCost{{PartNumber: "RM-001", AveragePrice: decimal.NewFromInt(3), ActualCost: decimal.NewFromInt(12)}},
	}
	c.summarise()
	before := c.VariancePercent
	c.convertTo(Rates{"USD": decimal.NewFromInt(16000)})
	if c.Currency != ReportingCurrency {
		t.Fatalf("currency stayed %s", c.Currency)
	}
	if !c.TotalActual.Equal(decimal.NewFromInt(18*16000)) || !c.MaterialActual.Equal(decimal.NewFromInt(12*16000)) {
		t.Fatalf("totals not converted: %+v", c)
	}
	if !c.Operations[0].ActualCost.Equal(decimal.NewFromInt(6*16000)) || !c.Materials[0].ActualCost.Equal(decimal.NewFromInt(12*16000)) {
		t.Fatalf("detail rows not converted: %+v %+v", c.Operations[0], c.Materials[0])
	}
	if !c.VariancePercent.Equal(before) {
		t.Fatalf("a ratio moved with the rate: %s then %s", before, c.VariancePercent)
	}
}
