package activitylog

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

type Query struct {
	TenantID uuid.UUID
	From     time.Time
	To       time.Time
	UserID   uuid.UUID
	Module   string
	Action   string
	Page     int
	PageSize int
}

func (q Query) Offset() int { return (q.Page - 1) * q.PageSize }

type Item struct {
	ID          uuid.UUID      `json:"id"`
	OccurredAt  time.Time      `json:"occurredAt"`
	ActorUserID *uuid.UUID     `json:"actorUserId,omitempty"`
	ActorName   string         `json:"actorName"`
	Module      string         `json:"module"`
	Action      string         `json:"action"`
	TargetType  string         `json:"targetType"`
	TargetID    *uuid.UUID     `json:"targetId,omitempty"`
	TargetCode  string         `json:"targetCode"`
	Before      map[string]any `json:"before,omitempty"`
	After       map[string]any `json:"after,omitempty"`
}

type ActorOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type FilterOptions struct {
	Actors  []ActorOption `json:"actors"`
	Modules []string      `json:"modules"`
	Actions []string      `json:"actions"`
}

type Page struct {
	Items    []Item        `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
	Filters  FilterOptions `json:"filters"`
}

func ParseQuery(values url.Values, now time.Time) (Query, map[string]string) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	query := Query{
		From:     today.AddDate(0, 0, -29),
		To:       today,
		Module:   strings.ToUpper(strings.TrimSpace(values.Get("module"))),
		Action:   strings.ToUpper(strings.TrimSpace(values.Get("action"))),
		Page:     1,
		PageSize: defaultPageSize,
	}
	fields := map[string]string{}

	parseDate := func(key string, fallback time.Time) time.Time {
		value := strings.TrimSpace(values.Get(key))
		if value == "" {
			return fallback
		}
		parsed, err := time.ParseInLocation(time.DateOnly, value, now.Location())
		if err != nil {
			fields[key] = "Use YYYY-MM-DD."
			return fallback
		}
		return parsed
	}
	query.From = parseDate("from", query.From)
	query.To = parseDate("to", query.To)
	if !query.From.After(query.To) {
		// Valid range.
	} else if fields["from"] == "" && fields["to"] == "" {
		fields["from"] = "From date must be on or before To date."
	}

	if value := strings.TrimSpace(values.Get("userId")); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			fields["userId"] = "Select a valid user."
		} else {
			query.UserID = parsed
		}
	}
	if value := strings.TrimSpace(values.Get("page")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			fields["page"] = "Page must be at least 1."
		} else {
			query.Page = parsed
		}
	}
	if value := strings.TrimSpace(values.Get("pageSize")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			fields["pageSize"] = "Page size must be at least 1."
		} else if parsed > maxPageSize {
			query.PageSize = maxPageSize
		} else {
			query.PageSize = parsed
		}
	}
	return query, fields
}
