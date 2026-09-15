package production

import (
	"context"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/database"
	"os"
	"testing"
	"time"
)

// These checks require independent PostgreSQL sessions, not the WASM harness.
func TestWIPConcurrentTransfersSQL(t *testing.T) {
	if os.Getenv("PRODUCTION_TEST_NATIVE") != "1" {
		t.Skip("set PRODUCTION_TEST_NATIVE=1 for native PostgreSQL concurrency checks")
	}
	ctx, _, s, a, o, material := newWIPSQLFixture(t)
	_, err := s.CreateEntry(ctx, a, EntryInput{OrderID: o.ID, OperationID: o.Operations[0].ID, OperatorID: a.UserID, ProductionDate: "2026-09-10", Shift: "1", Processed: decimal.NewFromInt(10), Good: decimal.NewFromInt(10), Materials: []MaterialUsage{{MaterialID: material, Quantity: decimal.NewFromInt(10)}}})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for n := 0; n < 2; n++ {
		go func() {
			<-start
			_, e := s.CreateTransfer(ctx, a, TransferInput{OrderID: o.ID, SourceOperationID: o.Operations[0].ID, Quantity: decimal.NewFromInt(8), MovementDate: "2026-09-10"})
			results <- e
		}()
	}
	close(start)
	successes := 0
	for n := 0; n < 2; n++ {
		if e := <-results; e == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("want exactly one transfer, got %d", successes)
	}
	balance, err := s.GetWIP(ctx, a, o.ID)
	if err != nil || !balance.Reconciled || !balance.Operations[0].OnHand.Equal(decimal.NewFromInt(2)) || !balance.Operations[1].Staged.Equal(decimal.NewFromInt(8)) {
		t.Fatalf("concurrent transfer overdrew: %+v %v", balance, err)
	}
}

func TestPeriodCloseWaitsForPostingSQL(t *testing.T) {
	if os.Getenv("PRODUCTION_TEST_NATIVE") != "1" {
		t.Skip("set PRODUCTION_TEST_NATIVE=1 for native PostgreSQL concurrency checks")
	}
	ctx, _, s, a, _, _ := newWIPSQLFixture(t)
	// Persist the period first so this exercises the row lock, not an insert race.
	if err := s.transaction(ctx, a, func(tx database.TenantTx) error { return lockEntryPeriod(ctx, tx, a, "2026-09-10") }); err != nil {
		t.Fatal(err)
	}
	err := s.transaction(ctx, a, func(tx database.TenantTx) error {
		if e := lockEntryPeriod(ctx, tx, a, "2026-09-10"); e != nil {
			return e
		}
		waiting, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancel()
		e := s.CloseProductionPeriod(waiting, a, "2026-09", "Close while posting")
		if e == nil || waiting.Err() == nil {
			t.Fatalf("close did not wait for posting lock: %v", e)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CloseProductionPeriod(ctx, a, "2026-09", "Posting complete"); err != nil {
		t.Fatal(err)
	}
	if err = s.transaction(ctx, a, func(tx database.TenantTx) error { return lockEntryPeriod(ctx, tx, a, "2026-09-10") }); err == nil {
		t.Fatal("posting accepted after period closed")
	}
}
