package production

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestRoutingConcurrentRevisions(t *testing.T) {
	if os.Getenv("PRODUCTION_TEST_NATIVE") != "1" {
		t.Skip("requires native PostgreSQL")
	}
	ctx, admin, s, a, o, _ := newWIPSQLFixture(t)
	r, err := s.GetRouting(ctx, a, o.RoutingID)
	if err != nil {
		t.Fatal(err)
	}
	blocker, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err = blocker.Exec(ctx, `SELECT id FROM production_routings WHERE id=$1 FOR UPDATE`, r.ID); err != nil {
		t.Fatal(err)
	}
	const count = 3
	results := make(chan Routing, count)
	failures := make(chan error, count)
	workCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	for range count {
		go func() {
			value, e := s.CreateRouting(workCtx, a, RoutingInput{PartID: r.PartID, Kind: r.Kind, Name: r.Name, Currency: r.Currency, Steps: r.Steps})
			results <- value
			failures <- e
		}()
	}
	observer, err := pgx.Connect(ctx, os.Getenv("PLANNING_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close(ctx)
	deadline := time.Now().Add(5 * time.Second)
	waiting := 0
	for time.Now().Before(deadline) {
		err = observer.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND (query LIKE '%production_routings%' OR query LIKE '%finished_goods%')`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting >= count {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if waiting < count {
		t.Fatal("concurrent routing requests did not reach the lock barrier")
	}
	if err = blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	revisions := map[int]bool{}
	for range count {
		value := <-results
		if e := <-failures; e != nil {
			t.Fatal(e)
		}
		if revisions[value.Revision] {
			t.Fatalf("duplicate revision %d", value.Revision)
		}
		revisions[value.Revision] = true
	}
	for revision := 2; revision <= count+1; revision++ {
		if !revisions[revision] {
			t.Fatalf("missing revision %d: %v", revision, revisions)
		}
	}
}

func TestRoutingConcurrentActivations(t *testing.T) {
	if os.Getenv("PRODUCTION_TEST_NATIVE") != "1" {
		t.Skip("requires native PostgreSQL")
	}
	ctx, _, s, a, o, _ := newWIPSQLFixture(t)
	first, err := s.GetRouting(ctx, a, o.RoutingID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateRouting(ctx, a, RoutingInput{PartID: first.PartID, Kind: first.Kind, Name: first.Name, Currency: first.Currency, Steps: first.Steps})
	if err != nil {
		t.Fatal(err)
	}
	workCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 8)
	for n := 0; n < 8; n++ {
		id := first.ID
		if n%2 == 1 {
			id = second.ID
		}
		go func() { <-start; _, e := s.RoutingAction(workCtx, a, id, "activate"); results <- e }()
	}
	close(start)
	for range 8 {
		if e := <-results; e != nil {
			t.Error(e)
		}
	}
	all, err := s.ListRoutings(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	active := 0
	for _, r := range all {
		if r.Active {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("active routings = %d", active)
	}
}
