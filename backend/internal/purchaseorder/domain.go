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
	StatusDraft           Status = "DRAFT"
	StatusPendingApproval Status = "PENDING_APPROVAL"
	StatusApproved        Status = "APPROVED"
	StatusRejected        Status = "REJECTED"
	StatusCancelled       Status = "CANCELLED"
)

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "PENDING"
	ApprovalApproved ApprovalStatus = "APPROVED"
	ApprovalRejected ApprovalStatus = "REJECTED"
)

type Actor struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	DisplayName string
	Email       string
}

type Order struct {
	ID                           uuid.UUID
	TenantID                     uuid.UUID
	PONumber                     string
	SupplierID                   uuid.UUID
	OrderDate                    time.Time
	ExpectedDeliveryDate         time.Time
	Currency                     string
	Notes                        string
	Status                       Status
	Version                      int
	TotalAmount                  decimal.Decimal
	SagePurchaseOrderNumber      string
	SubmittedApproverUserID      uuid.UUID
	SubmittedApproverDisplayName string
	SubmittedApproverEmail       string
	CreatedBy                    Actor
	CreatedAt                    time.Time
	UpdatedBy                    Actor
	UpdatedAt                    time.Time
	Lines                        []OrderLine
}

type OrderLine struct {
	ID                   uuid.UUID
	TenantID             uuid.UUID
	PurchaseOrderID      uuid.UUID
	RawMaterialID        uuid.UUID
	RawMaterialCode      string
	RawMaterialName      string
	BaseUnitID           uuid.UUID
	BaseUnitCode         string
	QtyPerKanbanSnapshot decimal.Decimal
	TotalKanban          decimal.Decimal
	OrderedBaseQty       decimal.Decimal
	UnitPriceSnapshot    decimal.Decimal
	LineTotal            decimal.Decimal
	SortPosition         int
	CreatedBy            Actor
	CreatedAt            time.Time
	UpdatedBy            Actor
	UpdatedAt            time.Time
}

func (l *OrderLine) Recalculate() {
	l.OrderedBaseQty = l.QtyPerKanbanSnapshot.Mul(l.TotalKanban)
	l.LineTotal = l.OrderedBaseQty.Mul(l.UnitPriceSnapshot)
}

func (o *Order) RecalculateTotals() {
	o.TotalAmount = decimal.Zero
	for index := range o.Lines {
		o.Lines[index].Recalculate()
		o.TotalAmount = o.TotalAmount.Add(o.Lines[index].LineTotal)
	}
}

type OrderInput struct {
	SupplierID           uuid.UUID   `json:"supplierId"`
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
	if i.OrderDate.IsZero() {
		errs["orderDate"] = "Order Date is required"
	}
	if i.ExpectedDeliveryDate.IsZero() {
		errs["expectedDeliveryDate"] = "Expected Delivery Date is required"
	} else if !i.OrderDate.IsZero() && i.ExpectedDeliveryDate.Before(i.OrderDate) {
		errs["expectedDeliveryDate"] = "Expected Delivery Date cannot precede Order Date"
	}
	if len(i.Currency) != 3 {
		errs["currency"] = "Currency must be a three-letter code"
	}
	if requireLines && len(i.Lines) == 0 {
		errs["lines"] = "At least one Raw Material is required before submission"
	}

	seenMaterials := make(map[uuid.UUID]struct{}, len(i.Lines))
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
		}
	}
	return errs
}

type Approval struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	PurchaseOrderID     uuid.UUID
	Version             int
	ApproverUserID      uuid.UUID
	ApproverDisplayName string
	ApproverEmail       string
	Status              ApprovalStatus
	DecisionReason      string
	DecidedAt           *time.Time
	DecidedByUserID     uuid.UUID
	CreatedBy           Actor
	CreatedAt           time.Time
	UpdatedBy           Actor
	UpdatedAt           time.Time
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
