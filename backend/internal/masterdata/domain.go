package masterdata

import (
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type FieldErrors map[string]string

type ListQuery struct {
	Search     string
	Active     *bool
	SupplierID uuid.UUID
	Limit      int
	Offset     int
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

type UnitInput struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	DecimalAllowed bool   `json:"decimalAllowed"`
	Active         *bool  `json:"active,omitempty"`
}

func (i *UnitInput) NormalizeAndValidate() FieldErrors {
	i.Code = normalizeCode(i.Code)
	i.Name = strings.TrimSpace(i.Name)
	errs := FieldErrors{}
	if i.Code == "" {
		errs["code"] = "Code is required"
	} else if i.Code == "KANBAN" {
		errs["code"] = "KANBAN is a physical lot, not a base unit"
	}
	if i.Name == "" {
		errs["name"] = "Name is required"
	}
	return errs
}

type SupplierInput struct {
	Code             string `json:"code"`
	SageSupplierCode string `json:"sageSupplierCode"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	Address          string `json:"address"`
	ContactPerson    string `json:"contactPerson"`
	Currency         string `json:"currency"`
	Active           *bool  `json:"active,omitempty"`
}

var supportedCurrencies = map[string]struct{}{"IDR": {}, "USD": {}, "EUR": {}, "JPY": {}, "SGD": {}}

func (i *SupplierInput) NormalizeAndValidate() FieldErrors {
	i.Code = normalizeCode(i.Code)
	i.SageSupplierCode = normalizeCode(i.SageSupplierCode)
	i.Name = strings.TrimSpace(i.Name)
	i.Email = strings.TrimSpace(i.Email)
	i.Phone = strings.TrimSpace(i.Phone)
	i.Address = strings.TrimSpace(i.Address)
	i.ContactPerson = strings.TrimSpace(i.ContactPerson)
	i.Currency = normalizeCode(i.Currency)
	errs := FieldErrors{}
	if i.Code == "" {
		errs["code"] = "Supplier ID is required"
	}
	if i.SageSupplierCode == "" {
		errs["sageSupplierCode"] = "Sage Supplier Code is required"
	}
	if i.Name == "" {
		errs["name"] = "Supplier Name is required"
	}
	if address, err := mail.ParseAddress(i.Email); err != nil || address.Address != i.Email {
		errs["email"] = "Valid email is required"
	}
	if _, ok := supportedCurrencies[i.Currency]; !ok {
		errs["currency"] = "Select a supported currency"
	}
	return errs
}

type RawMaterialInput struct {
	Code              string          `json:"code"`
	SageItemCode      string          `json:"sageItemCode"`
	Name              string          `json:"name"`
	SupplierID        uuid.UUID       `json:"supplierId"`
	BaseUnitID        uuid.UUID       `json:"baseUnitId"`
	QtyPerKanban      decimal.Decimal `json:"qtyPerKanban"`
	MinimumStock      decimal.Decimal `json:"minimumStock"`
	StandardUnitPrice decimal.Decimal `json:"standardUnitPrice"`
	Description       string          `json:"description"`
	Active            *bool           `json:"active,omitempty"`
}

func (i *RawMaterialInput) NormalizeAndValidate() FieldErrors {
	i.Code = normalizeCode(i.Code)
	i.SageItemCode = normalizeCode(i.SageItemCode)
	i.Name = strings.TrimSpace(i.Name)
	i.Description = strings.TrimSpace(i.Description)
	errs := FieldErrors{}
	if i.Code == "" {
		errs["code"] = "Item Code is required"
	}
	if i.SageItemCode == "" {
		errs["sageItemCode"] = "Sage Item Code is required"
	}
	if i.Name == "" {
		errs["name"] = "Raw Material Name is required"
	}
	if i.SupplierID == uuid.Nil {
		errs["supplierId"] = "Primary Supplier is required"
	}
	if i.BaseUnitID == uuid.Nil {
		errs["baseUnitId"] = "Base Unit is required"
	}
	if !i.QtyPerKanban.IsPositive() {
		errs["qtyPerKanban"] = "Qty per Kanban must be greater than zero"
	}
	if i.MinimumStock.IsNegative() {
		errs["minimumStock"] = "Minimum Stock cannot be negative"
	}
	if i.StandardUnitPrice.IsNegative() {
		errs["standardUnitPrice"] = "Standard Unit Price cannot be negative"
	}
	return errs
}

func normalizeCode(value string) string { return strings.ToUpper(strings.TrimSpace(value)) }
