package receiving

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("transaction conflict")
var ErrValidation = errors.New("validation failed")
var ErrDeliveryNoteInvalid = errors.New("Delivery Note is invalid.")
var ErrDeliveryNoteFullyReceived = errors.New("Delivery Note has already been fully received.")
var ErrDeliveryNoteInProgress = errors.New("Delivery Note is currently being received in another session.")

const SessionActive = "ACTIVE"
const SessionPaused = "PAUSED"

type Actor struct {
	TenantID, UserID uuid.UUID
	DisplayName      string
}
type Option struct {
	DeliveryNoteID     uuid.UUID `json:"deliveryNoteId"`
	DeliveryNoteNumber string    `json:"deliveryNoteNumber"`
	PONumber           string    `json:"poNumber"`
	SupplierName       string    `json:"supplierName"`
	Planned            int       `json:"planned"`
	Received           int       `json:"received"`
	Outstanding        int       `json:"outstanding"`
}
type Scan struct {
	KanbanLotID  uuid.UUID       `json:"kanbanLotId"`
	KanbanID     string          `json:"kanbanId"`
	MaterialCode string          `json:"materialCode"`
	MaterialName string          `json:"materialName"`
	Unit         string          `json:"unit"`
	Quantity     decimal.Decimal `json:"quantity"`
}
type Session struct {
	ID                 uuid.UUID `json:"id"`
	DeliveryNoteID     uuid.UUID `json:"deliveryNoteId"`
	ReceivingNumber    string    `json:"receivingNumber"`
	DeliveryNoteNumber string    `json:"deliveryNoteNumber"`
	PONumber           string    `json:"poNumber"`
	SupplierName       string    `json:"supplierName"`
	Status             string    `json:"status"`
	ReceivingDate      time.Time `json:"receivingDate"`
	Scans              []Scan    `json:"scans"`
	Planned            int       `json:"planned"`
	PreviouslyReceived int       `json:"previouslyReceived"`
	Outstanding        int       `json:"outstanding"`
	CreatedBy          string    `json:"createdBy"`
}
type Receiving struct {
	ID                 uuid.UUID `json:"id"`
	SupplierID         uuid.UUID `json:"supplierId"`
	ReceivingNumber    string    `json:"receivingNumber"`
	DeliveryNoteNumber string    `json:"deliveryNoteNumber"`
	PONumber           string    `json:"poNumber"`
	SupplierName       string    `json:"supplierName"`
	Status             string    `json:"status"`
	SageReceiptNumber  string    `json:"sageReceiptNumber"`
	CreatedBy          string    `json:"createdBy"`
	ReceivingDate      time.Time `json:"receivingDate"`
	Planned            int       `json:"planned"`
	PreviouslyReceived int       `json:"previouslyReceived"`
	ReceivedNow        int       `json:"receivedNow"`
	Outstanding        int       `json:"outstanding"`
	Scans              []Scan    `json:"scans"`
	CreatedAt          time.Time `json:"createdAt"`
}

type ListQuery struct {
	SupplierID         uuid.UUID
	CreatedFrom        time.Time
	CreatedToExclusive time.Time
}

func normalizeKanban(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", ErrValidation
	}
	return value, nil
}

func normalizeDeliveryNoteNumber(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", ErrValidation
	}
	return value, nil
}

func receivingErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrDeliveryNoteInvalid):
		return "DN_INVALID"
	case errors.Is(err, ErrDeliveryNoteFullyReceived):
		return "DN_FULLY_RECEIVED"
	case errors.Is(err, ErrDeliveryNoteInProgress):
		return "DN_IN_PROGRESS"
	default:
		return ""
	}
}
