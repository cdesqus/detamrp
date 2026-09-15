package production

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestTransferInputValidation(t *testing.T) {
	good := TransferInput{OrderID: uuid.New(), SourceOperationID: uuid.New(), Quantity: decimal.NewFromInt(25), MovementDate: "2026-09-14"}
	if e := good.Validate(); e != nil {
		t.Fatal(e)
	}
	for name, change := range map[string]func(*TransferInput){
		"no order":     func(v *TransferInput) { v.OrderID = uuid.Nil },
		"no operation": func(v *TransferInput) { v.SourceOperationID = uuid.Nil },
		"zero":         func(v *TransferInput) { v.Quantity = decimal.Zero },
		"negative":     func(v *TransferInput) { v.Quantity = decimal.NewFromInt(-5) },
		"precision":    func(v *TransferInput) { v.Quantity = decimal.RequireFromString("1.0000001") },
		"bad date":     func(v *TransferInput) { v.MovementDate = "2026-02-30" },
	} {
		t.Run(name, func(t *testing.T) {
			input := good
			change(&input)
			if input.Validate() == nil {
				t.Fatal("invalid transfer accepted")
			}
		})
	}
}

func TestAllocateFIFOConsumesOldestLotsFirst(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	lots := []Lot{
		{ID: first, Remaining: decimal.NewFromInt(20), UnitCost: decimal.NewFromInt(1000), Date: "2026-09-01"},
		{ID: second, Remaining: decimal.NewFromInt(30), UnitCost: decimal.NewFromInt(1100), Date: "2026-09-05"},
	}
	allocations, e := allocateFIFO(lots, decimal.NewFromInt(25))
	if e != nil {
		t.Fatal(e)
	}
	if len(allocations) != 2 || allocations[0].LotID != first || !allocations[0].Quantity.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("oldest lot not drained first: %+v", allocations)
	}
	if allocations[1].LotID != second || !allocations[1].Quantity.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("remainder not taken from the next lot: %+v", allocations)
	}
	// 20 × 1000 + 5 × 1100 = 25,500
	if !allocationCost(allocations).Equal(decimal.NewFromInt(25500)) {
		t.Fatalf("wrong allocation cost: %s", allocationCost(allocations))
	}
}

func TestAllocateFIFORejectsOverdraw(t *testing.T) {
	lots := []Lot{{ID: uuid.New(), Remaining: decimal.NewFromInt(10), UnitCost: decimal.NewFromInt(5)}}
	if _, e := allocateFIFO(lots, decimal.NewFromInt(11)); e == nil {
		t.Fatal("overdrawn WIP accepted")
	}
	if _, e := allocateFIFO(nil, decimal.NewFromInt(1)); e == nil {
		t.Fatal("empty balance accepted")
	}
	// A reversed lot leaves no balance behind.
	empty := []Lot{{ID: uuid.New(), Remaining: decimal.Zero, UnitCost: decimal.NewFromInt(5)}}
	if _, e := allocateFIFO(empty, decimal.NewFromInt(1)); e == nil {
		t.Fatal("exhausted lot accepted")
	}
}

func TestUnitCostSpreadsRejectsOverGoodOutput(t *testing.T) {
	// 10 processed pieces cost 1,000 but only 8 are good: 125 per good piece.
	if got := unitCost(decimal.NewFromInt(1000), decimal.NewFromInt(8)); !got.Equal(decimal.NewFromInt(125)) {
		t.Fatalf("wrong unit cost: %s", got)
	}
	if got := unitCost(decimal.NewFromInt(1000), decimal.Zero); !got.IsZero() {
		t.Fatalf("cost without output: %s", got)
	}
}

func TestOperationBalanceReconciliation(t *testing.T) {
	stamping := WIPOperationBalance{
		Sequence: 1, GoodQty: decimal.NewFromInt(90), ProcessedQty: decimal.NewFromInt(100),
		Received: decimal.NewFromInt(90), TransferredOut: decimal.NewFromInt(60),
	}
	stamping.reconcile()
	if !stamping.Reconciled || !stamping.OnHand.Equal(decimal.NewFromInt(30)) || !stamping.Balance.Equal(decimal.NewFromInt(30)) {
		t.Fatalf("stamping balance: %+v", stamping)
	}
	welding := WIPOperationBalance{
		Sequence: 2, Final: true, GoodQty: decimal.NewFromInt(55), ProcessedQty: decimal.NewFromInt(60),
		TransferredIn: decimal.NewFromInt(60), Consumed: decimal.NewFromInt(60),
	}
	welding.reconcile()
	if !welding.Reconciled || !welding.Balance.IsZero() {
		t.Fatalf("welding balance: %+v", welding)
	}
	// Good output that never reached the ledger must be reported, not hidden.
	broken := WIPOperationBalance{Sequence: 1, GoodQty: decimal.NewFromInt(90), ProcessedQty: decimal.NewFromInt(100), Received: decimal.NewFromInt(50)}
	broken.reconcile()
	if broken.Reconciled || !broken.Difference.Equal(decimal.NewFromInt(-40)) {
		t.Fatalf("unreconciled balance accepted: %+v", broken)
	}
}

func TestMovementLabelMarksReversals(t *testing.T) {
	if movementLabel(WIPMovement{Type: MovementTransfer}) != "TRANSFER" {
		t.Fatal("wrong movement label")
	}
	if movementLabel(WIPMovement{Type: MovementTransfer, ReversesID: uuid.New()}) != "REVERSAL" {
		t.Fatal("reversal not labelled")
	}
}
