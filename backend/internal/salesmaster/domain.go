package salesmaster

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Actor struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	Permissions []string
}

type Audit struct {
	CreatedBy     uuid.UUID `json:"createdBy"`
	CreatedByName string    `json:"createdByName"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedBy     uuid.UUID `json:"updatedBy"`
	UpdatedByName string    `json:"updatedByName"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Customer struct {
	ID      uuid.UUID `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Address string    `json:"address"`
	Contact string    `json:"contact"`
	Email   string    `json:"email"`
	Phone   string    `json:"phone"`
	Active  bool      `json:"active"`
	Audit
}

type FinishedGood struct {
	ID           uuid.UUID       `json:"id"`
	ItemCode     string          `json:"itemCode"`
	Name         string          `json:"name"`
	BaseUnitID   uuid.UUID       `json:"baseUnitId"`
	BaseUnitCode string          `json:"baseUnitCode"`
	BaseUnitName string          `json:"baseUnitName"`
	SalesPrice   decimal.Decimal `json:"salesPrice"`
	Currency     string          `json:"currency"`
	PriceVersion int             `json:"priceVersion"`
	Active       bool            `json:"active"`
	Audit
}

type CustomerInput struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Contact string `json:"contact"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Active  *bool  `json:"active,omitempty"`
}

func (i *CustomerInput) NormalizeAndValidate() FieldErrors {
	i.Code = normalizeCode(i.Code)
	i.Name = strings.TrimSpace(i.Name)
	i.Address = strings.TrimSpace(i.Address)
	i.Contact = strings.TrimSpace(i.Contact)
	i.Email = strings.TrimSpace(i.Email)
	i.Phone = strings.TrimSpace(i.Phone)
	fields := FieldErrors{}
	if i.Code == "" {
		fields["code"] = "Customer Code is required"
	}
	if i.Name == "" {
		fields["name"] = "Customer Name is required"
	}
	if i.Email != "" {
		address, err := mail.ParseAddress(i.Email)
		if err != nil || address.Address != i.Email {
			fields["email"] = "Enter a valid email"
		}
	}
	return fields
}

type FinishedGoodInput struct {
	ItemCode   string          `json:"itemCode"`
	Name       string          `json:"name"`
	BaseUnitID uuid.UUID       `json:"baseUnitId"`
	SalesPrice decimal.Decimal `json:"salesPrice"`
	Currency   string          `json:"currency"`
	Active     *bool           `json:"active,omitempty"`
}

var supportedCurrencies = map[string]struct{}{"IDR": {}, "USD": {}, "EUR": {}, "JPY": {}, "SGD": {}}

func (i *FinishedGoodInput) NormalizeAndValidate() FieldErrors {
	i.ItemCode = normalizeCode(i.ItemCode)
	i.Name = strings.TrimSpace(i.Name)
	i.Currency = normalizeCode(i.Currency)
	fields := FieldErrors{}
	if i.ItemCode == "" {
		fields["itemCode"] = "Item Code is required"
	}
	if i.Name == "" {
		fields["name"] = "Finished Good Name is required"
	}
	if i.BaseUnitID == uuid.Nil {
		fields["baseUnitId"] = "Base Unit is required"
	}
	if !i.SalesPrice.IsPositive() {
		fields["salesPrice"] = "Sales Price must be greater than zero"
	}
	if _, ok := supportedCurrencies[i.Currency]; !ok {
		fields["currency"] = "Select a supported currency"
	}
	return fields
}

type ListQuery struct {
	Search string
	Active *bool
	Limit  int
	Offset int
}

func (q *ListQuery) Normalize() {
	q.Search = strings.TrimSpace(q.Search)
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
}

type FieldErrors map[string]string

type ValidationError struct{ Fields FieldErrors }

func (ValidationError) Error() string { return "validation failed" }

type ConflictError struct{ Fields FieldErrors }

func (ConflictError) Error() string { return "record conflicts with existing data" }

type NotFoundError struct{ Resource string }

func (e NotFoundError) Error() string { return fmt.Sprintf("%s not found", e.Resource) }

type ForbiddenError struct{ Message string }

func (e ForbiddenError) Error() string {
	if e.Message == "" {
		return "permission denied"
	}
	return e.Message
}

func normalizeCode(value string) string { return strings.ToUpper(strings.TrimSpace(value)) }

func activeValue(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}
