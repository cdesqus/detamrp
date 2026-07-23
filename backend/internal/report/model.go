package report

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Actor struct{ TenantID, UserID uuid.UUID }

type Filter struct {
	FromDate, ToDate *time.Time
	SupplierID       *uuid.UUID
	Search           string
}

type Row struct {
	ReceivingNumber     string          `json:"receivingNumber"`
	DeliveryNoteNumber  string          `json:"deliveryNoteNumber"`
	PONumber            string          `json:"poNumber"`
	SupplierName        string          `json:"supplierName"`
	ReceivingDate       time.Time       `json:"receivingDate"`
	RawMaterialCode     string          `json:"rawMaterialCode"`
	RawMaterialName     string          `json:"rawMaterialName"`
	BaseUnitCode        string          `json:"baseUnitCode"`
	KanbanReceived      int             `json:"kanbanReceived"`
	ReceivedQuantity    decimal.Decimal `json:"receivedQuantity"`
	OutstandingQuantity decimal.Decimal `json:"outstandingQuantity"`
	SageNumber          string          `json:"sageNumber"`
	CreatedBy           string          `json:"createdBy"`
}

type Totals struct {
	KanbanReceived   int             `json:"kanbanReceived"`
	ReceivedQuantity decimal.Decimal `json:"receivedQuantity"`
}

type Result struct {
	Items  []Row  `json:"items"`
	Totals Totals `json:"totals"`
}

func summarize(rows []Row) Totals {
	var totals Totals
	for _, row := range rows {
		totals.KanbanReceived += row.KanbanReceived
		totals.ReceivedQuantity = totals.ReceivedQuantity.Add(row.ReceivedQuantity)
	}
	return totals
}
