package outgoing

import (
	"errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("transaction conflict")
var ErrValidation = errors.New("validation failed")

type Actor struct {
	TenantID, UserID uuid.UUID
	DisplayName      string
}
type Scan struct {
	KanbanLotID  uuid.UUID       `json:"kanbanLotId"`
	KanbanID     string          `json:"kanbanId"`
	MaterialCode string          `json:"materialCode"`
	MaterialName string          `json:"materialName"`
	Unit         string          `json:"unit"`
	Warehouse    string          `json:"warehouse"`
	Location     string          `json:"location"`
	Quantity     decimal.Decimal `json:"quantity"`
}
type Session struct {
	ID              uuid.UUID `json:"id"`
	DocumentNumber  string    `json:"documentNumber"`
	Destination     string    `json:"destination"`
	Notes           string    `json:"notes"`
	Status          string    `json:"status"`
	CreatedBy       string    `json:"createdBy"`
	TransactionDate time.Time `json:"transactionDate"`
	Scans           []Scan    `json:"scans"`
}
type Document struct {
	ID              uuid.UUID `json:"id"`
	DocumentNumber  string    `json:"documentNumber"`
	Destination     string    `json:"destination"`
	Notes           string    `json:"notes"`
	Status          string    `json:"status"`
	CreatedBy       string    `json:"createdBy"`
	TransactionDate time.Time `json:"transactionDate"`
	KanbanCount     int       `json:"kanbanCount"`
	MaterialCount   int       `json:"materialCount"`
	Scans           []Scan    `json:"scans"`
}

func normalizeDestination(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 120 {
		return "", ErrValidation
	}
	return v, nil
}
func normalizeKanban(v string) (string, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	if v == "" {
		return "", ErrValidation
	}
	return v, nil
}
