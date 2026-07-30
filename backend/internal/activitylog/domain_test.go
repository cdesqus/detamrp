package activitylog

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func containsFold(value, fragment string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(fragment))
}

func TestParseQueryNormalizesFiltersAndPagination(t *testing.T) {
	values := url.Values{
		"from":     {"2026-07-01"},
		"to":       {"2026-07-30"},
		"module":   {" procurement "},
		"action":   {" approved "},
		"page":     {"2"},
		"pageSize": {"50"},
	}

	query, fields := ParseQuery(values, time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC))

	if len(fields) != 0 {
		t.Fatalf("unexpected validation fields: %#v", fields)
	}
	if query.Module != "PROCUREMENT" || query.Action != "APPROVED" {
		t.Fatalf("filters were not normalized: %#v", query)
	}
	if query.Page != 2 || query.PageSize != 50 || query.Offset() != 50 {
		t.Fatalf("pagination was not normalized: %#v", query)
	}
}

func TestParseQueryRejectsInvalidValuesAndBoundsPageSize(t *testing.T) {
	values := url.Values{
		"from":     {"2026-07-31"},
		"to":       {"2026-07-01"},
		"userId":   {"not-a-uuid"},
		"page":     {"0"},
		"pageSize": {"500"},
	}

	query, fields := ParseQuery(values, time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC))

	for _, field := range []string{"from", "userId", "page"} {
		if fields[field] == "" {
			t.Fatalf("expected validation error for %s: %#v", field, fields)
		}
	}
	if query.PageSize != 100 {
		t.Fatalf("page size = %d, want max 100", query.PageSize)
	}
}

func TestBuildListQueryAlwaysScopesTenantAndAppliesFilters(t *testing.T) {
	query, fields := ParseQuery(url.Values{
		"from":   {"2026-07-01"},
		"to":     {"2026-07-30"},
		"module": {"inventory"},
		"action": {"moved"},
		"page":   {"3"},
	}, time.Now())
	if len(fields) != 0 {
		t.Fatalf("unexpected fields: %#v", fields)
	}

	sql, args := buildListQuery(query)
	for _, fragment := range []string{
		"tenant_id=$1", "occurred_at >= $2", "occurred_at < $3",
		"module=$4", "action=$5", "limit $6 offset $7",
	} {
		if !containsFold(sql, fragment) {
			t.Fatalf("query missing %q:\n%s", fragment, sql)
		}
	}
	if len(args) != 7 || args[0] != query.TenantID || args[6] != query.Offset() {
		t.Fatalf("unexpected query args: %#v", args)
	}
}
