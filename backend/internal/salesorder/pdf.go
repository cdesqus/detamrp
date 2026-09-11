package salesorder

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"order-stock/backend/internal/bom"
)

func RenderRequirementsPDF(order Order, requirements []RequirementLine) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 14)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(273, 5, fmt.Sprintf("Generated %s - Page %d", time.Now().Format("02 Jan 2006 15:04"), pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(180, 9, "DETA MRP", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(93, 9, "MATERIAL REQUIREMENTS", "", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(273, 6, "Sales Order "+order.Number+" - "+order.CustomerName+" - "+order.Status, "", 1, "L", false, 0, "")
	pdf.Ln(4)
	pdf.SetFillColor(245, 245, 247)
	pdf.SetFont("Arial", "B", 9)
	for i, h := range []string{"Finished Good", "Ordered", "Unit", "Sales Price", "Total Price", "Currency"} {
		pdf.CellFormat([]float64{90, 25, 20, 35, 38, 25}[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	widths := []float64{90, 25, 20, 35, 38, 25}
	for _, line := range order.Lines {
		values := []string{line.ItemCode + " - " + line.Name, line.Quantity.String(), line.Unit, line.SalesPrice.String(), line.Quantity.Mul(line.SalesPrice).String(), line.Currency}
		for i, v := range values {
			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(widths[i], 7, v, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(273, 7, "BOM Breakdown", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(245, 245, 247)
	bw := []float64{55, 82, 48, 45, 20}
	for i, h := range []string{"Finished Good", "Component", "Usage / Output", "Required Qty", "Unit"} {
		pdf.CellFormat(bw[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	for _, req := range requirements {
		if len(req.Result.Nodes) == 0 {
			continue
		}
		root := req.Result.Nodes[0]
		for _, node := range req.Result.Nodes {
			if node.Depth != 1 {
				continue
			}
			values := []string{root.ItemCode, node.ItemCode + " - " + node.Name, node.Usage.String() + " " + node.Unit + " per 1 " + root.Unit, node.Quantity.String(), node.Unit}
			for i, value := range values {
				pdf.SetFont("Arial", "", 8)
				pdf.CellFormat(bw[i], 7, value, "1", 0, "L", false, 0, "")
			}
			pdf.Ln(-1)
		}
	}
	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(273, 7, "Material Requirements", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(245, 245, 247)
	mw := []float64{35, 78, 35, 20, 35, 35, 25}
	for i, h := range []string{"Code", "Material", "Required", "Unit", "Unit Price", "Value", "Currency"} {
		pdf.CellFormat(mw[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	aggregate := map[string]bom.MaterialRequirement{}
	for _, req := range requirements {
		for _, m := range req.Result.Materials {
			if current, ok := aggregate[m.ItemID]; ok {
				current.Quantity = current.Quantity.Add(m.Quantity)
				current.Value = current.Value.Add(m.Value)
				aggregate[m.ItemID] = current
			} else {
				aggregate[m.ItemID] = m
			}
		}
	}
	for _, m := range aggregate {
		values := []string{m.ItemCode, m.Name, m.Quantity.String(), m.Unit, m.UnitPrice.String(), m.Value.String(), m.Currency}
		for i, v := range values {
			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(mw[i], 7, v, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
