package masterdata

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMeasurementInputNormalizesAndRejectsKanban(t *testing.T) {
	input := MeasurementInput{Code: " kanban ", Name: " Lot "}
	errs := input.NormalizeAndValidate()
	if errs["code"] == "" {
		t.Fatal("KANBAN was accepted")
	}
	if input.Code != "KANBAN" || input.Name != "Lot" {
		t.Fatalf("not normalized: %#v", input)
	}
}

func TestRawMaterialInputRejectsInvalidQuantities(t *testing.T) {
	input := RawMaterialInput{Code: "RM", SageItemCode: "S-RM", Name: "Material", SupplierID: uuid.New(), BaseUnitID: uuid.New(), QtyPerKanban: decimal.Zero, MinimumStock: decimal.NewFromInt(-1), StandardUnitPrice: decimal.NewFromInt(-1)}
	errs := input.NormalizeAndValidate()
	for _, field := range []string{"qtyPerKanban", "minimumStock", "standardUnitPrice"} {
		if errs[field] == "" {
			t.Fatalf("missing %s error: %#v", field, errs)
		}
	}
}

func TestSupplierInputValidatesEmailAndCurrency(t *testing.T) {
	input := SupplierInput{Code: " sup ", SageSupplierCode: " x3 ", Name: "Supplier", Email: "bad", Currency: "ABC"}
	errs := input.NormalizeAndValidate()
	if errs["email"] == "" || errs["currency"] == "" {
		t.Fatalf("got %#v", errs)
	}
	if input.Code != "SUP" || input.SageSupplierCode != "X3" {
		t.Fatalf("not normalized: %#v", input)
	}
}

func TestListQueryLimits(t *testing.T) {
	query := ListQuery{Limit: 500, Offset: -1}
	query.Normalize()
	if query.Limit != 200 || query.Offset != 0 {
		t.Fatalf("got %#v", query)
	}
}
