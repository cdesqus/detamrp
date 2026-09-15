package production

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
	"github.com/shopspring/decimal"
)

// documentPDF sets up the ink-efficient layout the operational documents share:
// no heavy fills, thin rules and a printed generation stamp.
func documentPDF(title, subtitle, generated string) *fpdf.Fpdf {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetCompression(false)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 12)
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(277, 7, title, "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(277, 5, subtitle, "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(277, 4, "Generated "+generated, "", 1, "L", false, 0, "")
		pdf.Ln(2)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-9)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(277, 5, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	pdf.AddPage()
	return pdf
}

func sectionTitle(pdf *fpdf.Fpdf, title string) {
	pdf.Ln(3)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(277, 6, title, "", 1, "L", false, 0, "")
}

func tableRow(pdf *fpdf.Fpdf, widths []float64, values []string, bold bool, aligns string) {
	style := ""
	if bold {
		style = "B"
	}
	pdf.SetFont("Arial", style, 7)
	for index, value := range values {
		align := "L"
		if index < len(aligns) && aligns[index] == 'R' {
			align = "R"
		}
		pdf.CellFormat(widths[index], 5.5, value, "1", 0, align, false, 0, "")
	}
	pdf.Ln(-1)
}

func definitionRow(pdf *fpdf.Fpdf, label, value string) {
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(45, 5, label, "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(93, 5, value, "", 0, "L", false, 0, "")
}

func amount(value decimal.Decimal, currency string) string {
	return currency + " " + value.StringFixed(2)
}

// RenderOrderReportPDF prints the production order execution document.
func RenderOrderReportPDF(r OrderReport) ([]byte, error) {
	o, cost := r.Order, r.Cost
	pdf := documentPDF("Production Order Execution Report",
		fmt.Sprintf("%s  ·  %s %s  ·  %s", o.OrderNumber, o.PartNumber, o.PartName, o.PlantName),
		r.GeneratedAt.Format("02 Jan 2006 15:04"))

	definitionRow(pdf, "Planning", o.PlanNumber)
	definitionRow(pdf, "Status", string(o.Status))
	pdf.Ln(-1)
	definitionRow(pdf, "Order period", o.PeriodStart+" to "+o.DueDate)
	definitionRow(pdf, "BOM / routing revision", fmt.Sprintf("%d / %d", o.BOMRevision, o.RoutingRevision))
	pdf.Ln(-1)
	definitionRow(pdf, "Planned quantity", o.PlannedQty.String()+" "+o.UnitCode)
	definitionRow(pdf, "Actual good / reject", cost.GoodQty.String()+" / "+cost.RejectQty.String()+" "+o.UnitCode)
	pdf.Ln(-1)
	definitionRow(pdf, "Work in progress", r.WIP.TotalQty.String()+" "+o.UnitCode)
	definitionRow(pdf, "Cost per finished piece", amount(cost.ActualUnitCost, o.Currency))
	pdf.Ln(-1)

	sectionTitle(pdf, "Progress per operation")
	operationWidths := []float64{14, 26, 40, 26, 26, 24, 24, 26, 34, 37}
	tableRow(pdf, operationWidths, []string{"Seq", "Code", "Operation", "Planned", "Processed", "Good", "Reject", "WIP", "Actual cost", "Cost per good piece"}, true, "LLLRRRRRRR")
	for index, operation := range o.Operations {
		actual, perPiece := decimal.Zero, decimal.Zero
		if index < len(cost.Operations) {
			actual, perPiece = cost.Operations[index].ActualCost, cost.Operations[index].CostPerPiece
		}
		tableRow(pdf, operationWidths, []string{fmt.Sprint(operation.Sequence), operation.Code, operation.Name,
			operation.PlannedQty.String(), operation.ProcessedQty.String(), operation.GoodQty.String(),
			operation.RejectQty.String(), operation.WIPQty.String(), amount(actual, o.Currency), amount(perPiece, o.Currency)},
			false, "LLLRRRRRRR")
	}

	sectionTitle(pdf, "Material usage")
	materialWidths := []float64{34, 63, 20, 32, 32, 32, 32, 32}
	tableRow(pdf, materialWidths, []string{"Part number", "Part name", "Unit", "Standard", "Actual", "Variance", "Avg price", "Actual cost"}, true, "LLLRRRRR")
	for _, material := range cost.Materials {
		tableRow(pdf, materialWidths, []string{material.PartNumber, material.PartName, material.UnitCode,
			material.StandardQty.String(), material.UsedQty.String(), material.QtyVariance.String(),
			amount(material.AveragePrice, o.Currency), amount(material.ActualCost, o.Currency)}, false, "LLLRRRRR")
	}

	sectionTitle(pdf, "Work in progress balance")
	wipWidths := []float64{40, 40, 40, 40, 40, 77}
	tableRow(pdf, wipWidths, []string{"Operation", "Received", "Transferred out", "Waiting", "Staged", "Balance"}, true, "LRRRRR")
	for _, balance := range r.WIP.Operations {
		tableRow(pdf, wipWidths, []string{balance.Code, balance.Received.String(), balance.TransferredOut.String(),
			balance.OnHand.String(), balance.Staged.String(), balance.Balance.String()}, false, "LRRRRR")
	}

	sectionTitle(pdf, "Actual cost")
	costWidths := []float64{97, 60, 60, 60}
	tableRow(pdf, costWidths, []string{"Cost component", "Estimate at release", "Actual", "Difference"}, true, "LRRR")
	tableRow(pdf, costWidths, []string{"Material", amount(cost.MaterialEstimate, o.Currency), amount(cost.MaterialActual, o.Currency),
		amount(cost.MaterialActual.Sub(cost.MaterialEstimate), o.Currency)}, false, "LRRR")
	tableRow(pdf, costWidths, []string{"Process", amount(cost.ProcessEstimate, o.Currency), amount(cost.ProcessActual, o.Currency),
		amount(cost.ProcessActual.Sub(cost.ProcessEstimate), o.Currency)}, false, "LRRR")
	tableRow(pdf, costWidths, []string{"Less: cost held in work in progress", "-", amount(cost.WIPValue.Neg(), o.Currency), "-"}, false, "LRRR")
	tableRow(pdf, costWidths, []string{"Finished output cost", amount(cost.TotalEstimate, o.Currency), amount(cost.FinishedCost, o.Currency),
		amount(cost.FinishedCost.Sub(cost.TotalEstimate), o.Currency)}, true, "LRRR")

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// RenderDashboardPDF prints the production dashboard as a native document.
func RenderDashboardPDF(d ProductionDashboard) ([]byte, error) {
	pdf := documentPDF("Production Dashboard",
		fmt.Sprintf("Period %s to %s", d.Filter.From, d.Filter.To),
		d.GeneratedAt.Format("02 Jan 2006 15:04"))
	currency := d.Totals.Currency

	definitionRow(pdf, "Planned quantity", d.Totals.PlannedQty.String())
	definitionRow(pdf, "Good quantity", d.Totals.GoodQty.String())
	pdf.Ln(-1)
	definitionRow(pdf, "Plan achievement", d.Totals.Achievement.String()+" %")
	definitionRow(pdf, "Reject rate", d.Totals.RejectRate.String()+" %")
	pdf.Ln(-1)
	definitionRow(pdf, "Work in progress", d.Totals.WIPQty.String())
	pdf.Ln(-1)
	definitionRow(pdf, "Actual cost booked", amount(d.Totals.TotalActual, currency))
	definitionRow(pdf, "Live WIP value", amount(d.Totals.WIPValue, currency))
	pdf.Ln(-1)

	sectionTitle(pdf, "Plan vs actual per part number")
	partWidths := []float64{34, 63, 24, 40, 40, 36, 40}
	tableRow(pdf, partWidths, []string{"Part number", "Part name", "Orders", "Planned", "Good", "Reject", "Achievement %"}, true, "LLRRRRR")
	for _, part := range d.Parts {
		tableRow(pdf, partWidths, []string{part.PartNumber, part.PartName, fmt.Sprint(part.Orders),
			part.PlannedQty.String(), part.GoodQty.String(), part.RejectQty.String(), part.Achievement.String()}, false, "LLRRRRR")
	}

	sectionTitle(pdf, "Work in progress per process")
	processWidths := []float64{97, 90, 90}
	tableRow(pdf, processWidths, []string{"Operation", "Quantity", "Value"}, true, "LRR")
	for _, process := range d.Processes {
		tableRow(pdf, processWidths, []string{process.Code, process.Quantity.String(), amount(process.Value, process.Currency)}, false, "LRR")
	}

	sectionTitle(pdf, "Reject rate per operation")
	qualityWidths := []float64{97, 60, 60, 60}
	tableRow(pdf, qualityWidths, []string{"Operation", "Processed", "Reject", "Reject rate %"}, true, "LRRR")
	for _, quality := range d.Quality {
		tableRow(pdf, qualityWidths, []string{quality.Code, quality.ProcessedQty.String(), quality.RejectQty.String(), quality.RejectRate.String()}, false, "LRRR")
	}

	sectionTitle(pdf, "Cost per part number (per piece: cumulative booked cost less live WIP / cumulative good output)")
	costWidths := []float64{34, 63, 24, 40, 40, 40, 36}
	tableRow(pdf, costWidths, []string{"Part number", "Part name", "Orders", "Good qty", "Material", "Process", "Cumulative / piece"}, true, "LLRRRRR")
	for _, cost := range d.Costs {
		tableRow(pdf, costWidths, []string{cost.PartNumber, cost.PartName, fmt.Sprint(cost.Orders), cost.GoodQty.String(),
			amount(cost.MaterialActual, cost.Currency), amount(cost.ProcessActual, cost.Currency), amount(cost.CostPerPiece, cost.Currency)}, false, "LLRRRRR")
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
