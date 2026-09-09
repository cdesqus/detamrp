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
type SalesOrderRow struct { Number string `json:"number"`; Customer string `json:"customer"`; OrderDate time.Time `json:"orderDate"`; Status string `json:"status"`; ItemCode string `json:"itemCode"`; ItemName string `json:"itemName"`; Ordered decimal.Decimal `json:"ordered"`; Delivered decimal.Decimal `json:"delivered"`; Remaining decimal.Decimal `json:"remaining"`; Unit string `json:"unit"` }

func summarize(rows []Row) Totals {
	var totals Totals
	for _, row := range rows {
		totals.KanbanReceived += row.KanbanReceived
		totals.ReceivedQuantity = totals.ReceivedQuantity.Add(row.ReceivedQuantity)
	}
	return totals
}
