package inventory

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrNotFound = errors.New("record not found")
var ErrValidation = errors.New("validation failed")

type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

type Filters struct {
	Search     string
	SupplierID *uuid.UUID
	Status     string
}

type Summary struct {
	TotalRawMaterials   int `json:"totalRawMaterials"`
	TotalInStockKanban  int `json:"totalInStockKanban"`
	LowStockMaterials   int `json:"lowStockMaterials"`
	OutOfStockMaterials int `json:"outOfStockMaterials"`
}

type StockItem struct {
	RawMaterialID   uuid.UUID       `json:"rawMaterialId"`
	ItemCode        string          `json:"itemCode"`
	RawMaterialName string          `json:"rawMaterialName"`
	SupplierID      uuid.UUID       `json:"supplierId"`
	SupplierName    string          `json:"supplierName"`
	AvailableKanban int             `json:"availableKanban"`
	StockQuantity   decimal.Decimal `json:"stockQuantity"`
	BaseUnitCode    string          `json:"baseUnitCode"`
	MinimumStock    decimal.Decimal `json:"minimumStock"`
	StockStatus     string          `json:"stockStatus"`
}

type StockResponse struct {
	Summary Summary     `json:"summary"`
	Items   []StockItem `json:"items"`
}

type KanbanItem struct {
	KanbanLotID        uuid.UUID       `json:"kanbanLotId"`
	KanbanID           string          `json:"kanbanId"`
	DeliveryNoteNumber string          `json:"deliveryNoteNumber"`
	PONumber           string          `json:"poNumber"`
	Quantity           decimal.Decimal `json:"quantity"`
	BaseUnitCode       string          `json:"baseUnitCode"`
	ReceivedDate       time.Time       `json:"receivedDate"`
}

type KanbanResponse struct {
	RawMaterialID   uuid.UUID    `json:"rawMaterialId"`
	ItemCode        string       `json:"itemCode"`
	RawMaterialName string       `json:"rawMaterialName"`
	Kanbans         []KanbanItem `json:"kanbans"`
}

func StockStatus(quantity, minimum decimal.Decimal) string {
	if quantity.IsZero() {
		return "OUT_OF_STOCK"
	}
	if quantity.LessThanOrEqual(minimum) {
		return "LOW_STOCK"
	}
	return "IN_STOCK"
}
