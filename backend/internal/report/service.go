package report

import (
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FieldErrors map[string]string

func ParseFilter(values url.Values) (Filter, FieldErrors) {
	var filter Filter
	fields := FieldErrors{}
	parseDate := func(key string) *time.Time {
		value := strings.TrimSpace(values.Get(key))
		if value == "" {
			return nil
		}
		date, err := time.Parse("2006-01-02", value)
		if err != nil {
			fields[key] = "Use YYYY-MM-DD"
			return nil
		}
		return &date
	}
	filter.FromDate, filter.ToDate = parseDate("fromDate"), parseDate("toDate")
	if strings.TrimSpace(values.Get("fromDate")) == "" {
		fields["fromDate"] = "From Date is required"
	}
	if strings.TrimSpace(values.Get("toDate")) == "" {
		fields["toDate"] = "To Date is required"
	}
	if filter.FromDate != nil && filter.ToDate != nil && filter.FromDate.After(*filter.ToDate) {
		fields["toDate"] = "To Date must be on or after From Date"
	}
	if value := strings.TrimSpace(values.Get("supplierId")); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			fields["supplierId"] = "Supplier is invalid"
		} else {
			filter.SupplierID = &id
		}
	}
	filter.Search = strings.TrimSpace(values.Get("search"))
	return filter, fields
}
