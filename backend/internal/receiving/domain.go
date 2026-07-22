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

const SessionActive = "ACTIVE"
const SessionPaused = "PAUSED"

type Actor struct {
	TenantID, UserID uuid.UUID
	DisplayName      string
}
type Option struct {
	DeliveryNoteID                             uuid.UUID `json:"deliveryNoteId"`
	DeliveryNoteNumber, PONumber, SupplierName string
	Planned, Received, Outstanding             int
}
type Scan struct {
	KanbanLotID                                uuid.UUID `json:"kanbanLotId"`
	KanbanID, MaterialCode, MaterialName, Unit string
	Quantity                                   decimal.Decimal `json:"quantity"`
}
type Session struct {
	ID                                                                  uuid.UUID `json:"id"`
	DeliveryNoteID                                                      uuid.UUID `json:"deliveryNoteId"`
	ReceivingNumber, DeliveryNoteNumber, PONumber, SupplierName, Status string
	ReceivingDate                                                       time.Time `json:"receivingDate"`
	Scans                                                               []Scan    `json:"scans"`
	Planned, PreviouslyReceived, Outstanding                            int
	CreatedBy                                                           string `json:"createdBy"`
}
type Receiving struct {
	ID                                                                                                uuid.UUID `json:"id"`
	ReceivingNumber, DeliveryNoteNumber, PONumber, SupplierName, Status, SageReceiptNumber, CreatedBy string
	ReceivingDate                                                                                     time.Time `json:"receivingDate"`
	Planned, PreviouslyReceived, ReceivedNow, Outstanding                                             int
	Scans                                                                                             []Scan `json:"scans"`
}

func normalizeKanban(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", ErrValidation
	}
	return value, nil
}
