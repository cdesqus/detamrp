package dashboard

import (
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Filter struct {
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	SupplierID uuid.UUID `json:"supplierId,omitempty"`
}

type Metrics struct {
	PendingApproval   int64 `json:"pendingApproval"`
	OpenPO            int64 `json:"openPO"`
	ReceivedKanban    int64 `json:"receivedKanban"`
	OutstandingKanban int64 `json:"outstandingKanban"`
	CurrentStock      int64 `json:"currentStock"`
}

type TrendPoint struct {
	Date     string `json:"date"`
	Ordered  int64  `json:"ordered"`
	Received int64  `json:"received"`
}

type StatusPoint struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type SupplierPoint struct {
	Supplier string `json:"supplier"`
	Kanban   int64  `json:"kanban"`
}

type Activity struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Label      string    `json:"label"`
	OccurredAt time.Time `json:"occurredAt"`
}

type Snapshot struct {
	Filter                Filter          `json:"filter"`
	Metrics               Metrics         `json:"metrics"`
	Trend                 []TrendPoint    `json:"trend"`
	POStatus              []StatusPoint   `json:"poStatus"`
	OutstandingBySupplier []SupplierPoint `json:"outstandingBySupplier"`
	Activities            []Activity      `json:"activities"`
}

func ParseFilter(values url.Values, now time.Time) (Filter, map[string]string) {
	defaultTo := now
	filter := Filter{
		From: defaultTo.AddDate(0, 0, -29),
		To:   defaultTo,
	}
	fields := map[string]string{}

	parseDate := func(key string, fallback time.Time) time.Time {
		value := strings.TrimSpace(values.Get(key))
		if value == "" {
			return fallback
		}
		parsed, err := time.Parse(time.DateOnly, value)
		if err != nil {
			fields[key] = "Use YYYY-MM-DD."
			return fallback
		}
		return parsed
	}
	filter.From = parseDate("from", filter.From)
	filter.To = parseDate("to", filter.To)

	if value := strings.TrimSpace(values.Get("supplierId")); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			fields["supplierId"] = "Select a valid supplier."
		} else {
			filter.SupplierID = parsed
		}
	}
	if _, badFrom := fields["from"]; !badFrom {
		if _, badTo := fields["to"]; !badTo && filter.From.After(filter.To) {
			fields["from"] = "From date must be on or before To date."
		}
	}
	return filter, fields
}
