package production

import (
	"github.com/shopspring/decimal"
	"testing"
)

func TestWIPBackdateCannotConsumeFutureLotSQL(t *testing.T) {
	ctx, _, s, a, o, material := newWIPSQLFixture(t)
	for _, date := range []string{"2026-09-01", "2026-09-10"} {
		_, err := s.CreateEntry(ctx, a, EntryInput{OrderID: o.ID, OperationID: o.Operations[0].ID, OperatorID: a.UserID, ProductionDate: date, Shift: "1", Processed: decimal.NewFromInt(10), Good: decimal.NewFromInt(10), Materials: []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(10)}}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: o.Operations[0].ID, MovementDate: date, Quantity: decimal.NewFromInt(10)}); err != nil {
			t.Fatal(err)
		}
	}
	input := EntryInput{OrderID: o.ID, OperationID: o.Operations[1].ID, OperatorID: a.UserID, ProductionDate: "2026-09-11", Shift: "1", Processed: decimal.NewFromInt(10), Good: decimal.NewFromInt(10)}
	if _, err := s.CreateEntry(ctx, a, input); err != nil {
		t.Fatal(err)
	}
	if _, err := s.OrderAction(ctx, a, o.ID, "start", ""); err != nil {
		t.Fatal(err)
	}
	input.ProductionDate = "2026-09-02"
	if _, err := s.CreateEntry(ctx, a, input); err == nil {
		t.Fatal("backdated entry consumed a future lot")
	}
	balance, err := s.GetWIP(ctx, a, o.ID)
	if err != nil || !balance.TotalQty.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("failed posting changed WIP: %+v %v", balance, err)
	}
}
