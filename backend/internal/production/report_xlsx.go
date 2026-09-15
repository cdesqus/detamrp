package production

import (
	"bytes"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

type sheetWriter struct {
	file   *excelize.File
	header int
	bold   int
	err    error
}

func newSheetWriter(sheets ...string) (*sheetWriter, error) {
	file := excelize.NewFile()
	for _, name := range sheets {
		if _, err := file.NewSheet(name); err != nil {
			return nil, err
		}
	}
	file.DeleteSheet("Sheet1")
	header, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Border: []excelize.Border{{Type: "bottom", Style: 1, Color: "D4D4D8"}}})
	if err != nil {
		return nil, err
	}
	bold, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	return &sheetWriter{file: file, header: header, bold: bold}, nil
}

func cell(column int, row int) string {
	name, _ := excelize.ColumnNumberToName(column + 1)
	return fmt.Sprintf("%s%d", name, row)
}

// row writes one line of values and returns the next row number.
func (w *sheetWriter) row(sheet string, row int, style int, values ...any) int {
	if w.err != nil {
		return row + 1
	}
	for index, value := range values {
		reference := cell(index, row)
		if decimalValue, ok := value.(decimal.Decimal); ok {
			number, _ := decimalValue.Float64()
			value = number
		}
		if err := w.file.SetCellValue(sheet, reference, value); err != nil {
			w.err = err
			return row + 1
		}
		if style != 0 {
			if err := w.file.SetCellStyle(sheet, reference, reference, style); err != nil {
				w.err = err
				return row + 1
			}
		}
	}
	return row + 1
}

func (w *sheetWriter) widths(sheet string, widths ...float64) {
	for index, width := range widths {
		name, _ := excelize.ColumnNumberToName(index + 1)
		if err := w.file.SetColWidth(sheet, name, name, width); err != nil && w.err == nil {
			w.err = err
		}
	}
}

func (w *sheetWriter) bytes() ([]byte, error) {
	defer w.file.Close()
	if w.err != nil {
		return nil, w.err
	}
	var output bytes.Buffer
	if err := w.file.Write(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// RenderOrderReportXLSX writes the native workbook behind a production order:
// summary, operations, material usage, WIP ledger and cost build-up.
func RenderOrderReportXLSX(r OrderReport) ([]byte, error) {
	w, err := newSheetWriter("Summary", "Operations", "Material Usage", "WIP Ledger")
	if err != nil {
		return nil, err
	}
	o, cost := r.Order, r.Cost
	row := w.row("Summary", 1, w.bold, "Production Order Execution Report")
	row = w.row("Summary", row, 0, "Generated", r.GeneratedAt.Format("2006-01-02 15:04"))
	row++
	for _, line := range [][]any{
		{"Production order", o.OrderNumber}, {"Planning", o.PlanNumber}, {"Part number", o.PartNumber},
		{"Part name", o.PartName}, {"Plant", o.PlantName}, {"Status", string(o.Status)},
		{"Order period", o.PeriodStart + " to " + o.DueDate}, {"BOM revision", o.BOMRevision},
		{"Routing revision", o.RoutingRevision}, {"Currency", o.Currency},
	} {
		row = w.row("Summary", row, 0, line...)
	}
	row++
	row = w.row("Summary", row, w.header, "Measure", "Quantity", "Unit")
	for _, line := range [][]any{
		{"Planned quantity", o.PlannedQty, o.UnitCode},
		{"Actual good", cost.GoodQty, o.UnitCode},
		{"Rejected", cost.RejectQty, o.UnitCode},
		{"Work in progress", r.WIP.TotalQty, o.UnitCode},
	} {
		row = w.row("Summary", row, 0, line...)
	}
	row++
	row = w.row("Summary", row, w.header, "Cost component", "Estimate at release", "Actual", "Currency")
	row = w.row("Summary", row, 0, "Material", cost.MaterialEstimate, cost.MaterialActual, o.Currency)
	row = w.row("Summary", row, 0, "Process", cost.ProcessEstimate, cost.ProcessActual, o.Currency)
	row = w.row("Summary", row, 0, "Cost held in WIP", "", cost.WIPValue, o.Currency)
	row = w.row("Summary", row, w.bold, "Finished output cost", cost.TotalEstimate, cost.FinishedCost, o.Currency)
	row = w.row("Summary", row, 0, "Cost per piece", cost.PlannedUnitCost, cost.ActualUnitCost, o.Currency)
	w.row("Summary", row, 0, "Variance per piece", "", cost.Variance, o.Currency)
	w.widths("Summary", 26, 22, 18, 12)

	row = w.row("Operations", 1, w.header, "Sequence", "Code", "Operation", "Planned", "Processed", "Good", "Reject", "WIP balance", "Rate", "Actual cost", "Cost per good piece")
	for index, operation := range o.Operations {
		actual, perPiece := decimal.Zero, decimal.Zero
		if index < len(cost.Operations) {
			actual, perPiece = cost.Operations[index].ActualCost, cost.Operations[index].CostPerPiece
		}
		row = w.row("Operations", row, 0, operation.Sequence, operation.Code, operation.Name, operation.PlannedQty,
			operation.ProcessedQty, operation.GoodQty, operation.RejectQty, operation.WIPQty, operation.Rate, actual, perPiece)
	}
	w.widths("Operations", 10, 16, 24, 14, 14, 12, 12, 14, 12, 14, 18)

	row = w.row("Material Usage", 1, w.header, "Part number", "Part name", "Unit", "Standard usage", "Actual usage", "Usage variance", "Average price", "Actual cost")
	for _, material := range cost.Materials {
		row = w.row("Material Usage", row, 0, material.PartNumber, material.PartName, material.UnitCode,
			material.StandardQty, material.UsedQty, material.QtyVariance, material.AveragePrice, material.ActualCost)
	}
	w.row("Material Usage", row, w.bold, "Total", "", "", "", "", "", "", cost.MaterialActual)
	w.widths("Material Usage", 18, 26, 10, 16, 16, 16, 16, 16)

	row = w.row("WIP Ledger", 1, w.header, "Date", "Movement", "From", "To", "Quantity", "Unit cost", "Total cost", "Entry", "Recorded by")
	for _, movement := range r.WIP.Movements {
		row = w.row("WIP Ledger", row, 0, movement.MovementDate, movementLabel(movement), movement.SourceCode, movement.DestinationCode,
			movement.Quantity, movement.UnitCost, movement.TotalCost, movement.EntryNumber, movement.CreatedBy)
	}
	w.widths("WIP Ledger", 14, 14, 14, 14, 14, 14, 14, 16, 20)
	return w.bytes()
}

// RenderDashboardXLSX writes the dashboard as a native workbook.
func RenderDashboardXLSX(d ProductionDashboard) ([]byte, error) {
	w, err := newSheetWriter("Summary", "Plan vs Actual", "WIP per Process", "Quality", "Cost per Part")
	if err != nil {
		return nil, err
	}
	row := w.row("Summary", 1, w.bold, "Production Dashboard")
	row = w.row("Summary", row, 0, "Period", d.Filter.From+" to "+d.Filter.To)
	row = w.row("Summary", row, 0, "Generated", d.GeneratedAt.Format("2006-01-02 15:04"))
	row++
	row = w.row("Summary", row, w.header, "Measure", "Value")
	for _, line := range [][]any{
		{"Planned quantity", d.Totals.PlannedQty}, {"Processed quantity", d.Totals.ProcessedQty},
		{"Good quantity", d.Totals.GoodQty}, {"Rejected quantity", d.Totals.RejectQty},
		{"Reject rate %", d.Totals.RejectRate}, {"Plan achievement %", d.Totals.Achievement},
		{"WIP quantity", d.Totals.WIPQty},

		{"Daily entries", d.Totals.Entries},
		{"Active orders", d.Totals.ActiveOrders}, {"Open orders", d.Totals.OpenOrders},
		{"Completed orders", d.Totals.CompletedOrder},
	} {
		row = w.row("Summary", row, 0, line...)
	}
	row = w.row("Summary", row, 0, "Actual cost booked", d.Totals.TotalActual, d.Totals.Currency)
	row = w.row("Summary", row, 0, "Live WIP value", d.Totals.WIPValue, d.Totals.Currency)
	w.widths("Summary", 26, 22, 12)

	row = w.row("Plan vs Actual", 1, w.header, "Part number", "Part name", "Unit", "Orders", "Planned", "Good", "Reject", "Achievement %")
	for _, part := range d.Parts {
		row = w.row("Plan vs Actual", row, 0, part.PartNumber, part.PartName, part.UnitCode, part.Orders,
			part.PlannedQty, part.GoodQty, part.RejectQty, part.Achievement)
	}
	w.widths("Plan vs Actual", 18, 26, 10, 10, 14, 14, 12, 16)

	row = w.row("WIP per Process", 1, w.header, "Operation", "Quantity", "Value ("+d.Totals.Currency+")")
	for _, process := range d.Processes {
		row = w.row("WIP per Process", row, 0, process.Code, process.Quantity, process.Value)
	}
	w.widths("WIP per Process", 18, 14, 16)

	row = w.row("Quality", 1, w.header, "Operation", "Processed", "Good", "Reject", "Reject rate %")
	for _, quality := range d.Quality {
		row = w.row("Quality", row, 0, quality.Code, quality.ProcessedQty, quality.GoodQty, quality.RejectQty, quality.RejectRate)
	}
	w.widths("Quality", 18, 14, 14, 12, 16)

	row = w.row("Cost per Part", 1, w.header, "Part number", "Part name", "Orders", "Good quantity", "Material", "Process", "Total actual", "Cumulative cost per piece", "Currency", "Cumulative finished cost", "Cumulative good quantity")
	for _, cost := range d.Costs {
		row = w.row("Cost per Part", row, 0, cost.PartNumber, cost.PartName, cost.Orders, cost.GoodQty,
			cost.MaterialActual, cost.ProcessActual, cost.TotalActual, cost.CostPerPiece, cost.Currency, cost.FinishedCost, cost.LifetimeGoodQty)
	}
	w.widths("Cost per Part", 18, 26, 10, 16, 14, 14, 16, 16, 10)
	return w.bytes()
}
