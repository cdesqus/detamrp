package database

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeTenantTx struct {
	configuredTenant string
	executedSQL      string
	executedArgs     []any
	committed        bool
	rolledBack       bool
}

func (f *fakeTenantTx) SetTenant(_ context.Context, tenantID string) error {
	f.configuredTenant = tenantID
	return nil
}
func (f *fakeTenantTx) Commit(_ context.Context) error   { f.committed = true; return nil }
func (f *fakeTenantTx) Rollback(_ context.Context) error { f.rolledBack = true; return nil }
func (f *fakeTenantTx) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	f.executedSQL = sql
	f.executedArgs = arguments
	return pgconn.CommandTag{}, nil
}
func (f *fakeTenantTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (f *fakeTenantTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }

type fakeTenantBeginner struct{ tx *fakeTenantTx }

func (f fakeTenantBeginner) BeginTenantTx(context.Context) (TenantTx, error) { return f.tx, nil }

func TestWithTenantSetsTransactionContextAndCommits(t *testing.T) {
	tx := &fakeTenantTx{}
	tenantID := uuid.New()
	userID := uuid.New()
	called := false

	err := WithTenant(context.Background(), fakeTenantBeginner{tx}, TenantContext{TenantID: tenantID, UserID: userID}, func(_ TenantTx) error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("WithTenant returned error: %v", err)
	}
	if tx.configuredTenant != tenantID.String() {
		t.Fatalf("expected tenant %s, got %s", tenantID, tx.configuredTenant)
	}
	if tx.executedSQL != "SELECT set_config('app.user_id', $1, true)" {
		t.Fatalf("actor context was not configured: %q", tx.executedSQL)
	}
	if len(tx.executedArgs) != 1 || tx.executedArgs[0] != userID.String() {
		t.Fatalf("unexpected actor context args: %#v", tx.executedArgs)
	}
	if !called || !tx.committed || tx.rolledBack {
		t.Fatalf("unexpected transaction state: called=%v committed=%v rolledBack=%v", called, tx.committed, tx.rolledBack)
	}
}

func TestWithTenantRollsBackCallbackFailure(t *testing.T) {
	tx := &fakeTenantTx{}
	expected := errors.New("business failure")

	err := WithTenant(context.Background(), fakeTenantBeginner{tx}, TenantContext{TenantID: uuid.New(), UserID: uuid.New()}, func(_ TenantTx) error {
		return expected
	})

	if !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got %v", err)
	}
	if !tx.rolledBack || tx.committed {
		t.Fatalf("expected rollback without commit")
	}
}
