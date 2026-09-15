package production

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

func TestOrderInputValidation(t *testing.T) {
	good := OrderInput{PlanLineID: uuid.New(), PlannedQty: decimal.NewFromInt(100), DueDate: "2026-09-30"}
	if e := good.Validate(true); e != nil {
		t.Fatal(e)
	}
	for _, change := range []func(*OrderInput){func(v *OrderInput) { v.PlanLineID = uuid.Nil }, func(v *OrderInput) { v.PlannedQty = decimal.Zero }, func(v *OrderInput) { v.DueDate = "bad" }} {
		v := good
		change(&v)
		if v.Validate(true) == nil {
			t.Fatal("invalid order accepted")
		}
	}
}
func TestOrderMutationGuards(t *testing.T) {
	for _, s := range []OrderStatus{OrderCompleted, OrderCancelled, OrderPartial} {
		if canEditOrder(s, 0) {
			t.Fatalf("editable %s", s)
		}
	}
	if canEditOrder(OrderReleased, 1) {
		t.Fatal("entry order editable")
	}
	if !canEditOrder(OrderReleased, 0) {
		t.Fatal("new order not editable")
	}
}
