package masterdata

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMeasurementRejectsKanbanAsBaseUnit(t *testing.T) {
	measurement := Measurement{Code: "KANBAN", Name: "Kanban"}
	if err := measurement.Validate(); err == nil {
		t.Fatal("KANBAN must not be accepted as a measurement")
	}
}

func TestSupplierRequiresValidEmail(t *testing.T) {
	supplier := Supplier{Code: "SUP-001", Name: "PT Baja", Email: "invalid"}
	if err := supplier.Validate(); err == nil {
		t.Fatal("invalid supplier email accepted")
	}
}

func TestRawMaterialRequiresSupplierUnitAndPositiveKanbanQuantity(t *testing.T) {
	material := RawMaterial{Code: "MAT-001", Name: "Bolt M8", SupplierID: uuid.New(), BaseUnitID: uuid.New(), QtyPerKanban: decimal.Zero}
	if err := material.Validate(); err == nil {
		t.Fatal("zero Kanban quantity accepted")
	}
	material.QtyPerKanban = decimal.NewFromInt(500)
	if err := material.Validate(); err != nil {
		t.Fatalf("valid raw material rejected: %v", err)
	}
}

func TestLocationRequiresWarehouse(t *testing.T) {
	location := Location{Code: "A-01", Name: "Rack A-01"}
	if err := location.Validate(); err == nil {
		t.Fatal("location without warehouse accepted")
	}
}
