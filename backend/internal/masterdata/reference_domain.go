package masterdata

import (
	"strings"

	"github.com/google/uuid"
)

type Category struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	Audit
}

type Packing struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	Audit
}

type Plant struct {
	ID      uuid.UUID `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Address string    `json:"address"`
	Active  bool      `json:"active"`
	Audit
}

type CategoryInput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      *bool  `json:"active,omitempty"`
}

func (i *CategoryInput) NormalizeAndValidate() FieldErrors {
	i.Code = normalizeCode(i.Code)
	i.Name = strings.TrimSpace(i.Name)
	i.Description = strings.TrimSpace(i.Description)
	return validateReferenceIdentity(i.Code, i.Name)
}

type PackingInput struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      *bool  `json:"active,omitempty"`
}

func (i *PackingInput) NormalizeAndValidate() FieldErrors {
	i.Code = normalizeCode(i.Code)
	i.Name = strings.TrimSpace(i.Name)
	i.Description = strings.TrimSpace(i.Description)
	return validateReferenceIdentity(i.Code, i.Name)
}

type PlantInput struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Active  *bool  `json:"active,omitempty"`
}

func (i *PlantInput) NormalizeAndValidate() FieldErrors {
	i.Code = normalizeCode(i.Code)
	i.Name = strings.TrimSpace(i.Name)
	i.Address = strings.TrimSpace(i.Address)
	return validateReferenceIdentity(i.Code, i.Name)
}

func validateReferenceIdentity(code, name string) FieldErrors {
	fields := FieldErrors{}
	if code == "" {
		fields["code"] = "Code is required"
	}
	if name == "" {
		fields["name"] = "Name is required"
	}
	return fields
}
