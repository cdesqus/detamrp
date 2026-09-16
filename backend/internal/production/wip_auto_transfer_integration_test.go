package production

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// A later operation pulls what it needs from the operation before it, so the
// floor only reports output. The movement is still a real ledger row.
func TestLaterOperationPullsUpstreamOutputSQL(t *testing.T) {
	ctx, _, s, a, o, material := newWIPSQLFixture(t)
	stamping, welding := o.Operations[0], o.Operations[1]

	first, e := s.CreateEntry(ctx, a, EntryInput{OrderID: o.ID, OperationID: stamping.ID, OperatorID: a.UserID,
		ProductionDate: "2026-09-10", Shift: "1", Processed: decimal.NewFromInt(20), Good: decimal.NewFromInt(20),
		Materials: []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(20)}}})
	if e != nil {
		t.Fatal("stamping:", e)
	}

	// No transfer was posted by hand; welding may still report its output.
	second, e := s.CreateEntry(ctx, a, EntryInput{OrderID: o.ID, OperationID: welding.ID, OperatorID: a.UserID,
		ProductionDate: "2026-09-11", Shift: "1", Processed: decimal.NewFromInt(12), Good: decimal.NewFromInt(11), Rejected: decimal.NewFromInt(1)})
	if e != nil {
		t.Fatal("welding without a manual transfer:", e)
	}

	balance, e := s.GetWIP(ctx, a, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !balance.Operations[0].OnHand.Equal(decimal.NewFromInt(8)) || !balance.Operations[0].TransferredOut.Equal(decimal.NewFromInt(12)) {
		t.Fatalf("stamping balance after the automatic move: %+v", balance.Operations[0])
	}
	if !balance.Operations[1].Consumed.Equal(decimal.NewFromInt(12)) || !balance.Operations[1].Staged.IsZero() {
		t.Fatalf("welding consumed the wrong quantity: %+v", balance.Operations[1])
	}
	if !balance.Reconciled {
		t.Fatalf("automatic movement broke reconciliation: %+v", balance.Operations)
	}
	automatic := 0
	for _, movement := range balance.Movements {
		if movement.Type == MovementTransfer && movement.EntryID == second.ID {
			automatic++
		}
	}
	if automatic == 0 {
		t.Fatal("the automatic move was not written to the ledger")
	}

	// Welding still cannot run ahead of stamping.
	if _, e = s.CreateEntry(ctx, a, EntryInput{OrderID: o.ID, OperationID: welding.ID, OperatorID: a.UserID,
		ProductionDate: "2026-09-12", Shift: "1", Processed: decimal.NewFromInt(9), Good: decimal.NewFromInt(9)}); e == nil {
		t.Fatal("welding processed more than stamping produced")
	}

	// Correcting welding takes its automatic movement back as well.
	if _, e = s.VoidEntry(ctx, a, second.ID, second.Version, "Wrong shift"); e != nil {
		t.Fatal("void:", e)
	}
	balance, e = s.GetWIP(ctx, a, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !balance.Operations[0].OnHand.Equal(decimal.NewFromInt(20)) || !balance.Operations[1].Staged.IsZero() {
		t.Fatalf("void did not return the stock to stamping: %+v", balance.Operations)
	}
	released, e := s.GetEntry(ctx, a, first.ID)
	if e != nil {
		t.Fatal(e)
	}
	if !released.CanCorrect {
		t.Fatal("stamping stayed locked after its output came back")
	}
	if first.ID == uuid.Nil {
		t.Fatal("fixture lost its first entry")
	}
}
