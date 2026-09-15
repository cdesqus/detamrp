package production

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"order-stock/backend/internal/database"
	"testing"
)

type guardDB struct{ tx *guardTx }

func (d guardDB) BeginTenantTx(context.Context) (database.TenantTx, error) { return d.tx, nil }

type guardTx struct {
	database.TenantTx
	status     PlanStatus
	linked     int
	rolledBack bool
	committed  bool
	writes     int
}

func (t *guardTx) SetTenant(context.Context, string) error { return nil }
func (t *guardTx) Commit(context.Context) error            { t.committed = true; return nil }
func (t *guardTx) Rollback(context.Context) error          { t.rolledBack = true; return nil }
func (t *guardTx) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	if q != "SELECT set_config('app.user_id', $1, true)" {
		t.writes++
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (t *guardTx) QueryRow(context.Context, string, ...any) pgx.Row { return guardRow{t} }

type guardRow struct{ t *guardTx }

func (r guardRow) Scan(dest ...any) error {
	if len(dest) != 1 {
		return errors.New("unexpected scan")
	}
	switch value := dest[0].(type) {
	case *PlanStatus:
		*value = r.t.status
	case *int:
		*value = r.t.linked
	default:
		return errors.New("unexpected scan type")
	}
	return nil
}
func TestStoreRejectsProtectedMutations(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status PlanStatus
		linked int
		action string
	}{
		{"approved edit", PlanApproved, 0, "update"}, {"closed edit", PlanClosed, 0, "update"}, {"linked draft edit", PlanDraft, 1, "update"},
		{"approved delete", PlanApproved, 0, "delete"}, {"linked draft delete", PlanDraft, 1, "delete"},
		{"closed approve", PlanClosed, 0, "approve"}, {"draft close", PlanDraft, 0, "close"}, {"draft order", PlanDraft, 0, "orders"}, {"duplicate orders", PlanApproved, 1, "orders"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &guardTx{status: tc.status, linked: tc.linked}
			s := NewStore(guardDB{tx})
			a := Actor{uuid.New(), uuid.New()}
			id := uuid.New()
			var err error
			switch tc.action {
			case "update":
				_, err = s.Update(context.Background(), a, id, validPlan())
			case "delete":
				err = s.Delete(context.Background(), a, id)
			case "approve":
				_, err = s.Approve(context.Background(), a, id)
			case "close":
				_, err = s.Close(context.Background(), a, id)
			case "orders":
				_, err = s.CreateOrders(context.Background(), a, id)
			}
			if !errors.Is(err, ErrConflict) {
				t.Fatalf("expected conflict, got %v", err)
			}
			if !tx.rolledBack || tx.committed || tx.writes != 0 {
				t.Fatalf("protected mutation wrote or committed: %+v", tx)
			}
		})
	}
}
