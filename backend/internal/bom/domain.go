package bom

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strconv"
	"strings"
)

const (
	OutputFG          = "FG"
	OutputRawMaterial = "RAW_MATERIAL"
	StatusDraft       = "DRAFT"
	StatusActive      = "ACTIVE"
)

type FieldErrors map[string]string
type OutputRef struct {
	Kind     string    `json:"kind"`
	ID       uuid.UUID `json:"id"`
	ItemCode string    `json:"itemCode"`
	Name     string    `json:"name"`
	Unit     string    `json:"unit"`
}
type ComponentInput struct {
	RawMaterialID uuid.UUID       `json:"rawMaterialId"`
	UsageQty      decimal.Decimal `json:"usageQty"`
}
type Component struct {
	RawMaterialID uuid.UUID       `json:"rawMaterialId"`
	ItemCode      string          `json:"itemCode"`
	Name          string          `json:"name"`
	Unit          string          `json:"unit"`
	UsageQty      decimal.Decimal `json:"usageQty"`
}
type Input struct {
	Output     OutputRef        `json:"output"`
	Notes      string           `json:"notes"`
	Components []ComponentInput `json:"components"`
}
type BOM struct {
	ID         uuid.UUID   `json:"id"`
	Output     OutputRef   `json:"output"`
	Revision   int         `json:"revision"`
	Status     string      `json:"status"`
	Notes      string      `json:"notes"`
	Components []Component `json:"components"`
}

func (i *Input) NormalizeAndValidate() FieldErrors {
	i.Output.Kind = strings.ToUpper(strings.TrimSpace(i.Output.Kind))
	i.Notes = strings.TrimSpace(i.Notes)
	fields := FieldErrors{}
	if i.Output.Kind != OutputFG && i.Output.Kind != OutputRawMaterial {
		fields["output.kind"] = "Select a Finished Good or Raw Material"
	}
	if i.Output.ID == uuid.Nil {
		fields["output.id"] = "Select the item this BOM produces"
	}
	if len(i.Components) == 0 {
		fields["components"] = "Add at least one component"
	}
	seen := map[uuid.UUID]bool{}
	for index, component := range i.Components {
		prefix := "components[" + strconv.Itoa(index) + "]."
		if component.RawMaterialID == uuid.Nil {
			fields[prefix+"rawMaterialId"] = "Select a Raw Material"
		} else if seen[component.RawMaterialID] {
			fields[prefix+"rawMaterialId"] = "A component can only be listed once"
		} else {
			seen[component.RawMaterialID] = true
		}
		if i.Output.Kind == OutputRawMaterial && component.RawMaterialID == i.Output.ID {
			fields[prefix+"rawMaterialId"] = "An item cannot be its own component"
		}
		if !component.UsageQty.IsPositive() || !component.UsageQty.Equal(component.UsageQty.Round(6)) {
			fields[prefix+"usageQty"] = "Usage must be a positive value with up to 6 decimals"
		}
	}
	return fields
}
