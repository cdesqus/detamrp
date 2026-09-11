package salesorder

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

func RenderRequirementsXLSX(order Order, requirements []RequirementLine) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	for _, name := range []string{"Summary", "Finished Goods", "BOM Breakdown", "Material Requirements"} {
		if name != "Sheet1" {
			if _, err := f.NewSheet(name); err != nil {
				return nil, err
			}
		}
	}
	f.DeleteSheet("Sheet1")
	header, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"1F2937"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center"}})
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	f.SetCellValue("Summary", "A1", "DETA MRP - Sales Order Requirements")
	f.SetCellStyle("Summary", "A1", "A1", bold)
	for i, row := range [][]string{{"Sales Order", order.Number}, {"Customer", order.CustomerName}, {"Status", order.Status}} {
		f.SetCellValue("Summary", fmt.Sprintf("A%d", i+3), row[0])
		f.SetCellValue("Summary", fmt.Sprintf("B%d", i+3), row[1])
	}
	fg := "Finished Goods"
	headers := []string{"Code", "Name", "Ordered", "Unit", "Sales Price", "Total Price", "Currency"}
	for i, h := range headers {
		f.SetCellValue(fg, fmt.Sprintf("%c1", 'A'+i), h)
		f.SetCellStyle(fg, fmt.Sprintf("%c1", 'A'+i), fmt.Sprintf("%c1", 'A'+i), header)
	}
	for r, line := range order.Lines {
		row := r + 2
		vals := []any{line.ItemCode, line.Name, line.Quantity.String(), line.Unit, line.SalesPrice.String(), line.Quantity.Mul(line.SalesPrice).String(), line.Currency}
		for i, v := range vals {
			f.SetCellValue(fg, fmt.Sprintf("%c%d", 'A'+i, row), v)
		}
	}
	for i, h := range []string{"Finished Good", "Component", "Required Qty", "Unit"} {
		f.SetCellValue("BOM Breakdown", fmt.Sprintf("%c1", 'A'+i), h)
		f.SetCellStyle("BOM Breakdown", fmt.Sprintf("%c1", 'A'+i), fmt.Sprintf("%c1", 'A'+i), header)
	}
	row := 2
	for _, req := range requirements {
		for _, n := range req.Result.Nodes {
			if n.Depth == 1 {
				f.SetCellValue("BOM Breakdown", fmt.Sprintf("A%d", row), req.LineID)
				f.SetCellValue("BOM Breakdown", fmt.Sprintf("B%d", row), n.ItemCode+" - "+n.Name)
				f.SetCellValue("BOM Breakdown", fmt.Sprintf("C%d", row), n.Quantity.String())
				f.SetCellValue("BOM Breakdown", fmt.Sprintf("D%d", row), n.Unit)
				row++
			}
		}
	}
	for i, h := range []string{"Code", "Material", "Required Qty", "Unit", "Unit Price", "Material Value", "Currency"} {
		f.SetCellValue("Material Requirements", fmt.Sprintf("%c1", 'A'+i), h)
		f.SetCellStyle("Material Requirements", fmt.Sprintf("%c1", 'A'+i), fmt.Sprintf("%c1", 'A'+i), header)
	}
	aggregate := map[string]struct{ code, name, unit, price, value, currency, qty string }{}
	for _, req := range requirements {
		for _, m := range req.Result.Materials {
			current := aggregate[m.ItemID]
			if current.code == "" {
				current = struct{ code, name, unit, price, value, currency, qty string }{m.ItemCode, m.Name, m.Unit, m.UnitPrice.String(), m.Value.String(), m.Currency, m.Quantity.String()}
			} else {
				current.qty = decimal.RequireFromString(current.qty).Add(m.Quantity).String()
				current.value = decimal.RequireFromString(current.value).Add(m.Value).String()
			}
			aggregate[m.ItemID] = current
		}
	}
	row = 2
	for _, m := range aggregate {
		for i, v := range []string{m.code, m.name, m.qty, m.unit, m.price, m.value, m.currency} {
			f.SetCellValue("Material Requirements", fmt.Sprintf("%c%d", 'A'+i, row), v)
		}
		row++
	}
	for _, sheet := range []string{"Summary", fg, "BOM Breakdown", "Material Requirements"} {
		f.SetColWidth(sheet, "A", "A", 22)
		f.SetColWidth(sheet, "B", "B", 34)
		f.SetRowHeight(sheet, 1, 22)
		f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1})
	}
	var out []byte
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	out = buffer.Bytes()
	return out, nil
}
