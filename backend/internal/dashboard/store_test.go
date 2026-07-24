package dashboard

import (
	"testing"
	"time"
)

func TestCompleteTrendFillsMissingDays(t *testing.T) {
	from, _ := time.Parse(time.DateOnly, "2026-07-01")
	to, _ := time.Parse(time.DateOnly, "2026-07-03")
	got := completeTrend(from, to, map[string]TrendPoint{
		"2026-07-02": {Date: "2026-07-02", Ordered: 3, Received: 2},
	})

	if len(got) != 3 {
		t.Fatalf("got %d points, want 3", len(got))
	}
	if got[0].Date != "2026-07-01" || got[0].Ordered != 0 || got[1].Ordered != 3 || got[2].Date != "2026-07-03" {
		t.Fatalf("unexpected completed trend: %+v", got)
	}
}
