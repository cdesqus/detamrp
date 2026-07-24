package dashboard

import (
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseFilterDefaultsToLastThirtyDays(t *testing.T) {
	filter, fields := ParseFilter(url.Values{}, time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC))

	if len(fields) != 0 || filter.From.Format(time.DateOnly) != "2026-06-25" || filter.To.Format(time.DateOnly) != "2026-07-24" || filter.SupplierID != uuid.Nil {
		t.Fatalf("unexpected defaults: filter=%+v fields=%v", filter, fields)
	}
}

func TestParseFilterAcceptsExplicitRangeAndSupplier(t *testing.T) {
	supplierID := uuid.New()
	filter, fields := ParseFilter(url.Values{
		"from":       {"2026-07-01"},
		"to":         {"2026-07-20"},
		"supplierId": {supplierID.String()},
	}, time.Now())

	if len(fields) != 0 || filter.From.Format(time.DateOnly) != "2026-07-01" || filter.To.Format(time.DateOnly) != "2026-07-20" || filter.SupplierID != supplierID {
		t.Fatalf("unexpected explicit filter: filter=%+v fields=%v", filter, fields)
	}
}

func TestParseFilterRejectsMalformedAndReversedRange(t *testing.T) {
	_, fields := ParseFilter(url.Values{
		"from":       {"2026/07/24"},
		"to":         {"invalid"},
		"supplierId": {"invalid"},
	}, time.Now())
	want := map[string]string{"from": "Use YYYY-MM-DD.", "to": "Use YYYY-MM-DD.", "supplierId": "Select a valid supplier."}
	for key, message := range want {
		if fields[key] != message {
			t.Fatalf("field %s: got %q want %q", key, fields[key], message)
		}
	}

	_, fields = ParseFilter(url.Values{
		"from": {"2026-07-25"},
		"to":   {"2026-07-24"},
	}, time.Now())
	if fields["from"] != "From date must be on or before To date." {
		t.Fatalf("unexpected reversed range error: %v", fields)
	}
}
