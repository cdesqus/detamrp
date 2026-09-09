package bom

import (
	"encoding/json"
	"github.com/shopspring/decimal"
	"testing"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func fixtureRoot() Node {
	plate := Node{ItemID: "plate", Kind: "RAW_MATERIAL", ItemCode: "001", Name: "Plate", Unit: "KG", Usage: dec("0.3"), UnitPrice: dec("20"), Currency: "IDR", QtyPerKanban: dec("50")}
	x := Node{ItemID: "x", Kind: "RAW_MATERIAL", ItemCode: "X", Unit: "PCS", Usage: dec("2"), Children: []Node{plate}}
	y := Node{ItemID: "y", Kind: "RAW_MATERIAL", ItemCode: "Y", Unit: "PCS", Usage: dec("1"), UnitPrice: dec("5"), Currency: "IDR", QtyPerKanban: dec("10")}
	return Node{ItemID: "a", Kind: "FG", ItemCode: "A", Unit: "PCS", Usage: dec("1"), Children: []Node{x, y}}
}
func TestCalculateNestedRequirementsValuesOnlyLeaves(t *testing.T) {
	snapshot, err := BuildSnapshot(fixtureRoot())
	if err != nil {
		t.Fatal(err)
	}
	got, err := Calculate(snapshot, dec("10"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 4 || len(got.Materials) != 2 {
		t.Fatalf("unexpected rows: %#v", got)
	}
	if !got.Totals["IDR"].Equal(dec("170")) {
		t.Fatalf("value = %v", got.Totals)
	}
	quantities := map[string]decimal.Decimal{}
	for _, row := range got.Materials {
		quantities[row.ItemID] = row.Quantity
	}
	if !quantities["plate"].Equal(dec("6")) || !quantities["y"].Equal(dec("10")) {
		t.Fatal(quantities)
	}
}
func TestCommonChildIsCountedForEveryPath(t *testing.T) {
	root := fixtureRoot()
	x := root.Children[0]
	x.Usage = dec("1")
	root.Children = append(root.Children, Node{ItemID: "branch", Kind: "RAW_MATERIAL", ItemCode: "B", Unit: "PCS", Usage: dec("1"), Children: []Node{x}})
	snapshot, err := BuildSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Calculate(snapshot, dec("10"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range got.Materials {
		if row.ItemID == "plate" && !row.Quantity.Equal(dec("9")) {
			t.Fatal(row.Quantity)
		}
	}
	if !got.Totals["IDR"].Equal(dec("230")) {
		t.Fatal(got.Totals)
	}
}
func TestSnapshotIsIndependentAndRoundTrips(t *testing.T) {
	root := fixtureRoot()
	snapshot, err := BuildSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	root.Children[0].Children[0].UnitPrice = dec("999")
	data, _ := json.Marshal(snapshot)
	var saved Snapshot
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	got, err := Calculate(saved, dec("10"))
	if err != nil || !got.Totals["IDR"].Equal(dec("170")) {
		t.Fatalf("%v %v", got, err)
	}
}
func TestSnapshotRejectsCyclesAndInvalidUsage(t *testing.T) {
	root := fixtureRoot()
	root.Children[0].ItemID = "a"
	root.Children[0].Kind = "FG"
	if _, err := BuildSnapshot(root); err == nil {
		t.Fatal("cycle accepted")
	}
	root = fixtureRoot()
	root.Children[0].Usage = dec("0")
	if _, err := BuildSnapshot(root); err == nil {
		t.Fatal("zero usage accepted")
	}
}
func TestKanbanSuggestionDoesNotChangeQuantityOrValue(t *testing.T) {
	root := Node{ItemID: "f", Kind: "FG", ItemCode: "F", Unit: "PCS", Usage: dec("1"), Children: []Node{{ItemID: "m", Kind: "RAW_MATERIAL", ItemCode: "M", Unit: "PCS", Usage: dec("1"), UnitPrice: dec("2000"), Currency: "IDR", QtyPerKanban: dec("50")}}}
	snapshot, _ := BuildSnapshot(root)
	got, err := Calculate(snapshot, dec("120"))
	if err != nil {
		t.Fatal(err)
	}
	row := got.Materials[0]
	if !row.Quantity.Equal(dec("120")) || !row.KanbanEquivalent.Equal(dec("2.4")) || !row.PurchaseKanban.Equal(dec("3")) || !got.Totals["IDR"].Equal(dec("240000")) {
		t.Fatalf("%+v", row)
	}
}
func TestZeroPriceAndEmptyRemaining(t *testing.T) {
	root := fixtureRoot()
	root.Children[1].UnitPrice = dec("0")
	snapshot, _ := BuildSnapshot(root)
	got, err := Calculate(snapshot, dec("10"))
	if err != nil || got.CostComplete {
		t.Fatalf("%v %v", got, err)
	}
	got, err = Calculate(snapshot, dec("0"))
	if err != nil || len(got.Materials) != 0 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err = Calculate(snapshot, dec("-1")); err == nil {
		t.Fatal("negative order accepted")
	}
}
