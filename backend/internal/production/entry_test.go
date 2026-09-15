package production

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

func validEntryInput() EntryInput {
	return EntryInput{OrderID: uuid.New(), OperationID: uuid.New(), OperatorID: uuid.New(), ProductionDate: "2026-09-14", Shift: "1", Processed: decimal.NewFromInt(10), Good: decimal.NewFromInt(9), Rejected: decimal.NewFromInt(1)}
}
func TestEntryInputValidation(t *testing.T) {
	p := validEntryInput()
	if e := p.Validate(false); e != nil {
		t.Fatal(e)
	}
	for name, change := range map[string]func(*EntryInput){"no order": func(p *EntryInput) { p.OrderID = uuid.Nil }, "no operation": func(p *EntryInput) { p.OperationID = uuid.Nil }, "no operator": func(p *EntryInput) { p.OperatorID = uuid.Nil }, "invalid date": func(p *EntryInput) { p.ProductionDate = "2026-02-30" }, "zero": func(p *EntryInput) { p.Processed = decimal.Zero }, "negative good": func(p *EntryInput) { p.Good = decimal.NewFromInt(-1) }, "excess output": func(p *EntryInput) { p.Good = decimal.NewFromInt(11) }, "precision": func(p *EntryInput) { p.Processed = decimal.RequireFromString("10.0000001") }, "no shift": func(p *EntryInput) { p.Shift = " " }} {
		t.Run(name, func(t *testing.T) {
			p := validEntryInput()
			change(&p)
			if p.Validate(false) == nil {
				t.Fatal("invalid entry accepted")
			}
		})
	}
}
func TestEntryRequiresVersionForEdit(t *testing.T) {
	p := validEntryInput()
	if p.Validate(true) == nil {
		t.Fatal("unversioned edit accepted")
	}
}

func TestEntryTimelineAllowsValidBackdating(t *testing.T) {
	days := []OperationDay{{Date: "2026-09-01", Supplied: decimal.NewFromInt(50)}, {Date: "2026-09-02", Supplied: decimal.NewFromInt(50), Processed: decimal.NewFromInt(50)}}
	if !inputTimelineAvailable(days, "2026-09-01", decimal.NewFromInt(50)) {
		t.Fatal("valid backdated input rejected")
	}
	if inputTimelineAvailable(days, "2026-08-31", decimal.NewFromInt(1)) {
		t.Fatal("future output used early")
	}
	days = []OperationDay{{Date: "2026-09-01", Supplied: decimal.NewFromInt(50), Processed: decimal.NewFromInt(50)}}
	if inputTimelineAvailable(days, "2026-09-01", decimal.NewFromInt(1)) {
		t.Fatal("double consumed input")
	}
}
func TestEntryAvailabilityFollowsStagedWIP(t *testing.T) {
	// 55 good pieces exist at stamping but only 15 were transferred to welding.
	o := Order{PlannedQty: decimal.NewFromInt(100), Operations: []Operation{
		{ID: uuid.New(), ProcessedQty: decimal.NewFromInt(60), GoodQty: decimal.NewFromInt(55), OnHandQty: decimal.NewFromInt(40)},
		{ID: uuid.New(), ProcessedQty: decimal.NewFromInt(20), StagedQty: decimal.NewFromInt(15)},
	}}
	if !operationRemaining(o, 0).Equal(decimal.NewFromInt(40)) {
		t.Fatal("wrong first operation remainder")
	}
	if !operationRemaining(o, 1).Equal(decimal.NewFromInt(15)) {
		t.Fatal("downstream availability must follow transferred WIP, not upstream good output")
	}
	o.Operations[1].StagedQty = decimal.Zero
	if !operationRemaining(o, 1).IsZero() {
		t.Fatal("welding may not run without staged WIP")
	}
}
func TestEntryStatusRecalculation(t *testing.T) {
	o := Order{PlannedQty: decimal.NewFromInt(100), Operations: []Operation{{ProcessedQty: decimal.NewFromInt(100), GoodQty: decimal.NewFromInt(90)}, {ProcessedQty: decimal.NewFromInt(90), GoodQty: decimal.NewFromInt(85)}}}
	if derivedOrderStatus(o) != OrderCompleted {
		t.Fatal("exhausted routing should complete including rejects")
	}
	o.Operations[1].ProcessedQty = decimal.NewFromInt(20)
	if derivedOrderStatus(o) != OrderPartial {
		t.Fatal("partial final output should be partial")
	}
	o.Operations[1].GoodQty = decimal.Zero
	if derivedOrderStatus(o) != OrderInProgress {
		t.Fatal("unfinished upstream routing should remain in progress")
	}
}
