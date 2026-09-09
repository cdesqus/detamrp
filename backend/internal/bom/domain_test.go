package bom

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

func TestBOMInputRejectsNoComponentsDuplicatesAndNonpositiveUsage(t *testing.T) {
	id := uuid.New()
	cases := []struct {
		name  string
		input Input
	}{
		{"empty", Input{Output: OutputRef{Kind: "FG", ID: id}}},
		{"duplicate", Input{Output: OutputRef{Kind: "FG", ID: id}, Components: []ComponentInput{{RawMaterialID: id, UsageQty: decimal.NewFromInt(1)}, {RawMaterialID: id, UsageQty: decimal.NewFromInt(1)}}}},
		{"zero", Input{Output: OutputRef{Kind: "FG", ID: id}, Components: []ComponentInput{{RawMaterialID: id, UsageQty: decimal.Zero}}}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.input.NormalizeAndValidate()) == 0 {
				t.Fatal("input accepted")
			}
		})
	}
}
func TestBOMInputAllowsRawMaterialOutput(t *testing.T) {
	input := Input{Output: OutputRef{Kind: "RAW_MATERIAL", ID: uuid.New()}, Components: []ComponentInput{{RawMaterialID: uuid.New(), UsageQty: decimal.RequireFromString("0.3")}}}
	if fields := input.NormalizeAndValidate(); len(fields) != 0 {
		t.Fatal(fields)
	}
}

func TestBOMInputRejectsRawMaterialAsItsOwnComponent(t *testing.T) {
	id := uuid.New()
	input := Input{Output: OutputRef{Kind: "RAW_MATERIAL", ID: id}, Components: []ComponentInput{{RawMaterialID: id, UsageQty: decimal.NewFromInt(1)}}}
	if len(input.NormalizeAndValidate()) == 0 { t.Fatal("self reference accepted") }
}
