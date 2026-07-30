package purchaseorder

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Status string

const (
	StatusDraft             Status = "DRAFT"
	StatusPendingApproval   Status = "PENDING_APPROVAL"
	StatusApproved          Status = "APPROVED"
	StatusPartiallyReceived Status = "PARTIALLY_RECEIVED"
	StatusFullyReceived     Status = "FULLY_RECEIVED"
	StatusRejected          Status = "REJECTED"
	StatusCancelled         Status = "CANCELLED"
)

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "PENDING"
	ApprovalApproved ApprovalStatus = "APPROVED"
	ApprovalRejected ApprovalStatus = "REJECTED"
)

const databaseDecimalPlaces int32 = 6

var maxTotalKanban = decimal.RequireFromString("99999999999999")

type Actor struct {
	TenantID    uuid.UUID `json:"tenantId"`
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
	Email       string    `json:"email"`
}

type Order struct {
	ID                           uuid.UUID        `json:"id"`
	TenantID                     uuid.UUID        `json:"tenantId"`
	PONumber                     string           `json:"poNumber"`
	CompanyName                  string           `json:"companyName"`
	SupplierID                   uuid.UUID        `json:"supplierId"`
	SupplierName                 string           `json:"supplierName"`
	PlantID                      uuid.UUID        `json:"plantId"`
	PlantCode                    string           `json:"plantCode"`
	PlantName                    string           `json:"plantName"`
	PlantAddress                 string           `json:"plantAddress"`
	OrderDate                    time.Time        `json:"orderDate"`
	ExpectedDeliveryDate         time.Time        `json:"expectedDeliveryDate"`
	Currency                     string           `json:"currency"`
	Notes                        string           `json:"notes"`
	Status                       Status           `json:"status"`
	Version                      int              `json:"version"`
	TotalAmount                  decimal.Decimal  `json:"totalAmount"`
	SagePurchaseOrderNumber      string           `json:"sagePurchaseOrderNumber"`
	SubmittedApproverUserID      uuid.UUID        `json:"submittedApproverUserId"`
	SubmittedApproverDisplayName string           `json:"submittedApproverDisplayName"`
	SubmittedApproverEmail       string           `json:"submittedApproverEmail"`
	CreatedBy                    Actor            `json:"createdBy"`
	CreatedAt                    time.Time        `json:"createdAt"`
	UpdatedBy                    Actor            `json:"updatedBy"`
	UpdatedAt                    time.Time        `json:"updatedAt"`
	Lines                        []OrderLine      `json:"lines"`
	Documents                    *DocumentSummary `json:"documents"`
}

type DocumentSummary struct {
	DeliveryNoteID     uuid.UUID `json:"deliveryNoteId"`
	DeliveryNoteNumber string    `json:"deliveryNoteNumber"`
	KanbanCount        int64     `json:"kanbanCount"`
	IssuedAt           time.Time `json:"issuedAt"`
}

type OrderLine struct {
	ID                   uuid.UUID       `json:"id"`
	TenantID             uuid.UUID       `json:"tenantId"`
	PurchaseOrderID      uuid.UUID       `json:"purchaseOrderId"`
	RawMaterialID        uuid.UUID       `json:"rawMaterialId"`
	RawMaterialCode      string          `json:"rawMaterialCode"`
	RawMaterialName      string          `json:"rawMaterialName"`
	BaseUnitID           uuid.UUID       `json:"baseUnitId"`
	BaseUnitCode         string          `json:"baseUnitCode"`
	CategoryCode         string          `json:"categoryCode"`
	CategoryName         string          `json:"categoryName"`
	PackingCode          string          `json:"packingCode"`
	PackingName          string          `json:"packingName"`
	QtyPerKanbanSnapshot decimal.Decimal `json:"qtyPerKanbanSnapshot"`
	TotalKanban          decimal.Decimal `json:"totalKanban"`
	OrderedBaseQty       decimal.Decimal `json:"orderedBaseQty"`
	UnitPriceSnapshot    decimal.Decimal `json:"unitPriceSnapshot"`
	LineTotal            decimal.Decimal `json:"lineTotal"`
	SortPosition         int             `json:"sortPosition"`
	CreatedBy            Actor           `json:"createdBy"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedBy            Actor           `json:"updatedBy"`
	UpdatedAt            time.Time       `json:"updatedAt"`
}

func (l *OrderLine) Recalculate() {
	l.OrderedBaseQty = roundDatabaseDecimal(l.QtyPerKanbanSnapshot.Mul(l.TotalKanban))
	l.LineTotal = roundDatabaseDecimal(l.OrderedBaseQty.Mul(l.UnitPriceSnapshot))
}

func (o *Order) RecalculateTotals() {
	o.TotalAmount = decimal.Zero
	for index := range o.Lines {
		o.Lines[index].Recalculate()
		o.TotalAmount = o.TotalAmount.Add(o.Lines[index].LineTotal)
	}
	o.TotalAmount = roundDatabaseDecimal(o.TotalAmount)
}

// roundDatabaseDecimal applies PostgreSQL numeric(20,6) precision using half-away-from-zero rounding.
func roundDatabaseDecimal(value decimal.Decimal) decimal.Decimal {
	return value.Round(databaseDecimalPlaces)
}

type OrderInput struct {
	SupplierID           uuid.UUID   `json:"supplierId"`
	PlantID              uuid.UUID   `json:"plantId"`
	OrderDate            time.Time   `json:"orderDate"`
	ExpectedDeliveryDate time.Time   `json:"expectedDeliveryDate"`
	Currency             string      `json:"currency"`
	Notes                string      `json:"notes"`
	Lines                []LineInput `json:"lines"`
}

type LineInput struct {
	RawMaterialID uuid.UUID       `json:"rawMaterialId"`
	TotalKanban   decimal.Decimal `json:"totalKanban"`
}

func (i *OrderInput) NormalizeAndValidate(requireLines bool) FieldErrors {
	i.Currency = strings.ToUpper(strings.TrimSpace(i.Currency))
	i.Notes = strings.TrimSpace(i.Notes)
	errs := FieldErrors{}
	if i.SupplierID == uuid.Nil {
		errs["supplierId"] = "Supplier is required"
	}
	if i.PlantID == uuid.Nil {
		errs["plantId"] = "Plant is required"
	}
	if i.OrderDate.IsZero() {
		errs["orderDate"] = "Order Date is required"
	}
	if i.ExpectedDeliveryDate.IsZero() {
		errs["expectedDeliveryDate"] = "Expected Delivery Date is required"
	} else if !i.OrderDate.IsZero() && i.ExpectedDeliveryDate.Before(i.OrderDate) {
		errs["expectedDeliveryDate"] = "Expected Delivery Date cannot precede Order Date"
	}
	if !isASCIICurrency(i.Currency) {
		errs["currency"] = "Currency must be a three-letter code"
	}
	if requireLines && len(i.Lines) == 0 {
		errs["lines"] = "At least one Raw Material is required before submission"
	}

	seenMaterials := make(map[uuid.UUID]struct{}, len(i.Lines))
	totalKanbans := decimal.Zero
	for index := range i.Lines {
		line := &i.Lines[index]
		prefix := "lines[" + strconv.Itoa(index) + "]."
		if line.RawMaterialID == uuid.Nil {
			errs[prefix+"rawMaterialId"] = "Raw Material is required"
		} else if _, seen := seenMaterials[line.RawMaterialID]; seen {
			errs[prefix+"rawMaterialId"] = "Raw Material can only be added once"
		} else {
			seenMaterials[line.RawMaterialID] = struct{}{}
		}
		if !line.TotalKanban.IsPositive() || !line.TotalKanban.Equal(line.TotalKanban.Truncate(0)) {
			errs[prefix+"totalKanban"] = "Total Kanban must be a positive integer"
		} else if line.TotalKanban.GreaterThan(maxTotalKanban) {
			errs[prefix+"totalKanban"] = "Total Kanban cannot exceed 99999999999999"
		} else {
			totalKanbans = totalKanbans.Add(line.TotalKanban)
		}
	}
	if requireLines && totalKanbans.GreaterThan(decimal.NewFromInt(maxKanbanSequence)) {
		errs["lines"] = "A purchase order cannot exceed 999999 Kanban labels"
	}
	return errs
}

func isASCIICurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

type Approval struct {
	ID                  uuid.UUID      `json:"id"`
	TenantID            uuid.UUID      `json:"tenantId"`
	PurchaseOrderID     uuid.UUID      `json:"purchaseOrderId"`
	PONumber            string         `json:"poNumber"`
	SupplierID          uuid.UUID      `json:"supplierId"`
	SupplierName        string         `json:"supplierName"`
	Version             int            `json:"version"`
	ApproverUserID      uuid.UUID      `json:"approverUserId"`
	ApproverDisplayName string         `json:"approverDisplayName"`
	ApproverEmail       string         `json:"approverEmail"`
	Status              ApprovalStatus `json:"status"`
	DecisionReason      string         `json:"decisionReason"`
	DecidedAt           *time.Time     `json:"decidedAt"`
	DecidedByUserID     uuid.UUID      `json:"decidedByUserId"`
	CreatedBy           Actor          `json:"createdBy"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedBy           Actor          `json:"updatedBy"`
	UpdatedAt           time.Time      `json:"updatedAt"`
}

type DecisionInput struct {
	Reason string `json:"reason"`
}

func (i *DecisionInput) NormalizeAndValidate(rejected bool) FieldErrors {
	i.Reason = strings.TrimSpace(i.Reason)
	errs := FieldErrors{}
	if rejected && i.Reason == "" {
		errs["reason"] = "Rejection reason is required"
	}
	return errs
}

type ListQuery struct {
	SupplierID uuid.UUID
	Status     Status
	Search     string
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
