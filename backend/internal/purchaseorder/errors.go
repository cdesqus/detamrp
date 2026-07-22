package purchaseorder

import (
	"fmt"

	"github.com/google/uuid"
)

type FieldErrors map[string]string

type ValidationError struct{ Fields FieldErrors }

func (e ValidationError) Error() string { return "validation failed" }

type NotFoundError struct{ Resource string }

func (e NotFoundError) Error() string { return fmt.Sprintf("%s not found", e.Resource) }

type ConflictError struct{ Fields FieldErrors }

func (e ConflictError) Error() string { return "record conflicts with existing data" }

type CapacityError struct {
	Field   string
	Message string
}

func (e CapacityError) Error() string { return "document numbering capacity exceeded" }

type ApprovalDocumentError struct {
	ApprovalID      uuid.UUID
	PurchaseOrderID uuid.UUID
	Err             error
}

func (e ApprovalDocumentError) Error() string { return "approval document generation failed" }
func (e ApprovalDocumentError) Unwrap() error { return e.Err }
