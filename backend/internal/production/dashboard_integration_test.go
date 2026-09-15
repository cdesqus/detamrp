package production

import (
	"github.com/shopspring/decimal"
	"testing"
)

func TestDashboardCumulativeFinishedCostSQL(t *testing.T) {
	ctx, _, s, a, o, material := newWIPSQLFixture(t)
	_, err := s.CreateEntry(ctx, a, EntryInput{OrderID: o.ID, OperationID: o.Operations[0].ID, OperatorID: a.UserID, ProductionDate: "2026-09-01", Shift: "1", Processed: decimal.NewFromInt(20), Good: decimal.NewFromInt(20), Materials: []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(20)}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: o.Operations[0].ID, MovementDate: "2026-09-02", Quantity: decimal.NewFromInt(10)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateEntry(ctx, a, EntryInput{OrderID: o.ID, OperationID: o.Operations[1].ID, OperatorID: a.UserID, ProductionDate: "2026-09-03", Shift: "1", Processed: decimal.NewFromInt(10), Good: decimal.NewFromInt(10)})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Dashboard(ctx, a, DashboardFilter{From: "2026-09-03", To: "2026-09-03"})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Costs) != 1 {
		t.Fatalf("cost rows: %+v", d.Costs)
	}
	c := d.Costs[0]
	// Period books only welding (80), but finished output carries 150 of earlier material/stamping.
	if !c.TotalActual.Equal(decimal.NewFromInt(80)) || !c.FinishedCost.Equal(decimal.NewFromInt(230)) || !c.CostPerPiece.Equal(decimal.NewFromInt(23)) {
		t.Fatalf("period and cumulative cost confused: %+v", c)
	}
	if !d.Totals.WIPValue.Equal(decimal.NewFromInt(150)) || d.Totals.Currency != ReportingCurrency {
		t.Fatalf("live WIP: %+v", d.Totals)
	}
}
