package masterdata

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Audit struct {
	CreatedBy     uuid.UUID `json:"createdBy"`
	CreatedByName string    `json:"createdByName"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedBy     uuid.UUID `json:"updatedBy"`
	UpdatedByName string    `json:"updatedByName"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Unit struct {
	ID             uuid.UUID `json:"id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	DecimalAllowed bool      `json:"decimalAllowed"`
	Active         bool      `json:"active"`
	Audit
}

func (m Unit) Validate() error {
	if strings.TrimSpace(m.Code) == "" || strings.TrimSpace(m.Name) == "" {
		return errors.New("unit code and name are required")
	}
	if strings.EqualFold(strings.TrimSpace(m.Code), "KANBAN") {
		return errors.New("KANBAN is a physical lot, not a unit")
	}
	return nil
}

type Supplier struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	SageSupplierCode string    `json:"sageSupplierCode"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	Phone            string    `json:"phone"`
	Address          string    `json:"address"`
	ContactPerson    string    `json:"contactPerson"`
	Currency         string    `json:"currency"`
	Active           bool      `json:"active"`
	Audit
}

func (s Supplier) Validate() error {
	if strings.TrimSpace(s.Code) == "" || strings.TrimSpace(s.Name) == "" {
		return errors.New("supplier code and name are required")
	}
	address, err := mail.ParseAddress(strings.TrimSpace(s.Email))
	if err != nil || address.Address != strings.TrimSpace(s.Email) {
		return errors.New("supplier email is invalid")
	}
	if len(strings.TrimSpace(s.Currency)) != 3 {
		return errors.New("supplier currency must be a three-letter code")
	}
	return nil
}

type RawMaterial struct {
	ID                uuid.UUID       `json:"id"`
	Code              string          `json:"code"`
	SageItemCode      string          `json:"sageItemCode"`
	Name              string          `json:"name"`
	SupplierID        uuid.UUID       `json:"supplierId"`
	SupplierName      string          `json:"supplierName"`
	BaseUnitID        uuid.UUID       `json:"baseUnitId"`
	BaseUnitCode      string          `json:"baseUnitCode"`
	QtyPerKanban      decimal.Decimal `json:"qtyPerKanban"`
	MinimumStock      decimal.Decimal `json:"minimumStock"`
	StandardUnitPrice decimal.Decimal `json:"standardUnitPrice"`
	Currency          string          `json:"currency"`
	Description       string          `json:"description"`
	Active            bool            `json:"active"`
	Audit
}

func (r RawMaterial) Validate() error {
	if strings.TrimSpace(r.Code) == "" || strings.TrimSpace(r.Name) == "" {
		return errors.New("material code and name are required")
	}
	if r.SupplierID == uuid.Nil || r.BaseUnitID == uuid.Nil {
		return errors.New("supplier and base unit are required")
	}
	if !r.QtyPerKanban.IsPositive() {
		return errors.New("quantity per Kanban must be positive")
	}
	if r.MinimumStock.IsNegative() {
		return errors.New("minimum stock cannot be negative")
	}
	return nil
}

type Warehouse struct {
	ID                uuid.UUID `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	SageWarehouseCode string    `json:"sageWarehouseCode"`
	Address           string    `json:"address"`
	WarehouseType     string    `json:"warehouseType"`
	Active            bool      `json:"active"`
	Audit
}

func (w Warehouse) Validate() error {
	if strings.TrimSpace(w.Code) == "" || strings.TrimSpace(w.Name) == "" {
		return errors.New("warehouse code and name are required")
	}
	return nil
}

type Location struct {
	ID          uuid.UUID `json:"id"`
	WarehouseID uuid.UUID `json:"warehouseId"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Zone        string    `json:"zone"`
	Rack        string    `json:"rack"`
	Bin         string    `json:"bin"`
	Active      bool      `json:"active"`
	Audit
}

func (l Location) Validate() error {
	if l.WarehouseID == uuid.Nil {
		return errors.New("warehouse is required")
	}
	if strings.TrimSpace(l.Code) == "" || strings.TrimSpace(l.Name) == "" {
		return errors.New("location code and name are required")
	}
	return nil
}
