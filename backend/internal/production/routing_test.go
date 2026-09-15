package production

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func routingSteps() []RoutingStep {
	return []RoutingStep{
		{Code: "STAMPING", Name: "Stamping", Rate: decimal.NewFromInt(250)},
		{Code: "WELDING", Name: "Welding", Rate: decimal.NewFromInt(400)},
	}
}

func validRouting() RoutingInput {
	return RoutingInput{PartID: uuid.New(), Kind: "FG", Name: "Panel routing", Currency: "IDR", Steps: routingSteps()}
}

func TestRoutingInputNormalisation(t *testing.T) {
	input := RoutingInput{PartID: uuid.New(), Kind: " fg ", Name: "  Panel routing  ", Steps: []RoutingStep{{Code: " stamping ", Name: " Stamping ", Rate: decimal.NewFromInt(250)}}}
	input.Normalize()
	if input.Kind != "FG" || input.Name != "Panel routing" || input.Currency != "IDR" {
		t.Fatalf("unexpected normalisation: %+v", input)
	}
	if input.Steps[0].Code != "STAMPING" || input.Steps[0].Name != "Stamping" {
		t.Fatalf("step not normalised: %+v", input.Steps[0])
	}
	if e := input.Validate(true); e != nil {
		t.Fatal(e)
	}
}

func TestRoutingInputValidation(t *testing.T) {
	if e := validRouting().Validate(true); e != nil {
		t.Fatal(e)
	}
	for name, change := range map[string]func(*RoutingInput){
		"no part":        func(v *RoutingInput) { v.PartID = uuid.Nil },
		"unknown kind":   func(v *RoutingInput) { v.Kind = "ASSEMBLY" },
		"no name":        func(v *RoutingInput) { v.Name = "" },
		"bad currency":   func(v *RoutingInput) { v.Currency = "RUPIAH" },
		"no steps":       func(v *RoutingInput) { v.Steps = nil },
		"duplicate step": func(v *RoutingInput) { v.Steps[1].Code = "STAMPING" },
		"empty code":     func(v *RoutingInput) { v.Steps[0].Code = "" },
		"spaced code":    func(v *RoutingInput) { v.Steps[0].Code = "SUB ASSEMBLY" },
		"no step name":   func(v *RoutingInput) { v.Steps[0].Name = "" },
		"negative rate":  func(v *RoutingInput) { v.Steps[0].Rate = decimal.NewFromInt(-1) },
		"long rate":      func(v *RoutingInput) { v.Steps[0].Rate = decimal.RequireFromString("0.0000001") },
	} {
		input := validRouting()
		change(&input)
		if input.Validate(true) == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestRoutingStepLimit(t *testing.T) {
	input := validRouting()
	input.Steps = nil
	for i := 0; i <= maxRoutingSteps; i++ {
		input.Steps = append(input.Steps, RoutingStep{Code: "OP" + decimal.NewFromInt(int64(i)).String(), Name: "Operation", Rate: decimal.Zero})
	}
	if input.Validate(true) == nil {
		t.Fatal("oversized routing accepted")
	}
}

func TestRoutingTotalRateAndEditGuard(t *testing.T) {
	if got := validRouting().TotalRate(); !got.Equal(decimal.NewFromInt(650)) {
		t.Fatalf("total rate %s", got)
	}
	if canEditRouting(1) {
		t.Fatal("routing used by an order is editable")
	}
	if !canEditRouting(0) {
		t.Fatal("unused routing is not editable")
	}
}
