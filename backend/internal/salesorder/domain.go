package salesorder

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strconv"
	"strings"
)

const (
	StatusDraft     = "DRAFT"
	StatusSubmitted = "SUBMITTED"
)

type FieldErrors map[string]string
type LineInput struct {
	FinishedGoodID uuid.UUID       `json:"finishedGoodId"`
	Quantity       decimal.Decimal `json:"quantity"`
}
type Input struct {
	CustomerID          uuid.UUID   `json:"customerId"`
	OrderDate           string      `json:"orderDate"`
	DeliveryDate        string      `json:"deliveryDate"`
	CustomerPOReference string      `json:"customerPoReference"`
	Notes               string      `json:"notes"`
	Lines               []LineInput `json:"lines"`
}
type DeliveryLineInput struct {
	SalesOrderLineID uuid.UUID       `json:"salesOrderLineId"`
	Quantity         decimal.Decimal `json:"quantity"`
}
type DeliveryInput struct {
	DeliveryDate string              `json:"deliveryDate"`
	Notes        string              `json:"notes"`
	Lines        []DeliveryLineInput `json:"lines"`
}
type Delivery struct {
	ID           uuid.UUID           `json:"id"`
	Number       string              `json:"number"`
	SalesOrderID uuid.UUID           `json:"salesOrderId"`
	DeliveryDate string              `json:"deliveryDate"`
	Lines        []DeliveryLineInput `json:"lines"`
}
type DeliverySummary struct {
	ID            uuid.UUID       `json:"id"`
	Number        string          `json:"number"`
	DeliveryDate  string          `json:"deliveryDate"`
	TotalQuantity decimal.Decimal `json:"totalQuantity"`
}

func (i *DeliveryInput) NormalizeAndValidate() FieldErrors {
	fields := FieldErrors{}
	if len(i.Lines) == 0 {
		fields["lines"] = "Add at least one line"
	}
	seen := map[uuid.UUID]bool{}
	for n, line := range i.Lines {
		prefix := "lines[" + strconv.Itoa(n) + "]."
		if line.SalesOrderLineID == uuid.Nil {
			fields[prefix+"salesOrderLineId"] = "Select an order line"
		} else if seen[line.SalesOrderLineID] {
			fields[prefix+"salesOrderLineId"] = "A line can only be delivered once"
		} else {
			seen[line.SalesOrderLineID] = true
		}
		if !line.Quantity.IsPositive() || !line.Quantity.Equal(line.Quantity.Round(6)) {
			fields[prefix+"quantity"] = "Quantity must be positive with up to 6 decimals"
		}
	}
	return fields
}

func (i *Input) NormalizeAndValidate() FieldErrors {
	i.CustomerPOReference = strings.TrimSpace(i.CustomerPOReference)
	i.Notes = strings.TrimSpace(i.Notes)
	fields := FieldErrors{}
	if i.CustomerID == uuid.Nil {
		fields["customerId"] = "Select a customer"
	}
	if len(i.Lines) == 0 {
		fields["lines"] = "Add at least one Finished Good"
	}
	seen := map[uuid.UUID]bool{}
	for n, line := range i.Lines {
		prefix := "lines[" + strconv.Itoa(n) + "]."
		if line.FinishedGoodID == uuid.Nil {
			fields[prefix+"finishedGoodId"] = "Select a Finished Good"
		} else if seen[line.FinishedGoodID] {
			fields[prefix+"finishedGoodId"] = "A Finished Good can only be listed once"
		} else {
			seen[line.FinishedGoodID] = true
		}
		if !line.Quantity.IsPositive() || !line.Quantity.Equal(line.Quantity.Round(6)) {
			fields[prefix+"quantity"] = "Quantity must be positive with up to 6 decimals"
		}
	}
	return fields
}
