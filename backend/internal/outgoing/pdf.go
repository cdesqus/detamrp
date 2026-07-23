package outgoing

import (
	"bytes"
	"fmt"
	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

func RenderPDF(d Document) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.AddUTF8FontFromBytes("OS", "", goregular.TTF)
	p.AddUTF8FontFromBytes("OS", "B", gobold.TTF)
	p.AddPage()
	p.SetFillColor(24, 24, 27)
	p.Rect(0, 0, 210, 30, "F")
	p.SetTextColor(255, 255, 255)
	p.SetXY(14, 9)
	p.SetFont("OS", "B", 17)
	p.CellFormat(120, 8, "OUTGOING MATERIAL", "", 0, "L", false, 0, "")
	p.SetFont("OS", "B", 10)
	p.CellFormat(60, 8, d.DocumentNumber, "", 1, "R", false, 0, "")
	p.SetTextColor(24, 24, 27)
	p.SetY(39)
	for _, x := range [][2]string{{"Date", d.TransactionDate.Format("02 Jan 2006")}, {"Destination", outgoingDisplayValue(d.Destination)}, {"Completed By", d.CreatedBy}, {"Kanban", fmt.Sprint(d.KanbanCount)}, {"Materials", fmt.Sprint(d.MaterialCount)}, {"Notes", outgoingDisplayValue(d.Notes)}} {
		p.SetFont("OS", "B", 9)
		p.CellFormat(38, 7, x[0], "", 0, "L", false, 0, "")
		p.SetFont("OS", "", 9)
		p.CellFormat(0, 7, x[1], "", 1, "L", false, 0, "")
	}
	if len(d.Scans) > 0 {
		p.Ln(5)
		p.SetFillColor(244, 244, 245)
		p.SetFont("OS", "B", 9)
		p.CellFormat(45, 7, "KANBAN ID", "1", 0, "L", true, 0, "")
		p.CellFormat(72, 7, "RAW MATERIAL", "1", 0, "L", true, 0, "")
		p.CellFormat(35, 7, "QUANTITY", "1", 1, "R", true, 0, "")
		p.SetFont("OS", "", 8)
		for _, scan := range d.Scans {
			p.CellFormat(45, 7, scan.KanbanID, "1", 0, "L", false, 0, "")
			p.CellFormat(72, 7, scan.MaterialCode+" - "+scan.MaterialName, "1", 0, "L", false, 0, "")
			p.CellFormat(35, 7, scan.Quantity.String()+" "+scan.Unit, "1", 1, "R", false, 0, "")
		}
	}
	var b bytes.Buffer
	if e := p.Output(&b); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}

func outgoingDisplayValue(value string) string {
	if value == "" {
		return "—"
	}
	return value
}
