package salesorder

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/bom"
)

func TestCalculateRequirementsUsesEachLineSnapshot(t *testing.T) {
	snapshot, err := bom.BuildSnapshot(bom.Node{ItemID: uuid.NewString(), Kind: "FG", ItemCode: "FG-1", Name: "Finished Good", Unit: "PCS", Usage: decimal.NewFromInt(1), Children: []bom.Node{{ItemID: uuid.NewString(), Kind: "RAW_MATERIAL", ItemCode: "RM-1", Name: "Steel", Unit: "KG", Usage: decimal.RequireFromString("2.5"), UnitPrice: decimal.NewFromInt(100), Currency: "IDR"}}})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	result, err := CalculateRequirements([]Line{{ID: uuid.New(), Quantity: decimal.NewFromInt(3), Calculation: payload}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Result.Materials[0].Quantity.String() != "7.5" {
		t.Fatalf("unexpected requirements: %#v", result)
	}
}

func TestCalculateRequirementsRejectsDraftLineWithoutSnapshot(t *testing.T) {
	_, err := CalculateRequirements([]Line{{ID: uuid.New(), Quantity: decimal.NewFromInt(1)}})
	if err == nil {
		t.Fatal("expected missing snapshot error")
	}
}
