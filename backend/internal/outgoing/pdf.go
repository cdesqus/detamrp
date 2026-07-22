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
	for _, x := range [][2]string{{"Date", d.TransactionDate.Format("02 Jan 2006")}, {"Destination", d.Destination}, {"Completed By", d.CreatedBy}, {"Kanban", fmt.Sprint(d.KanbanCount)}, {"Materials", fmt.Sprint(d.MaterialCount)}, {"Notes", d.Notes}} {
		p.SetFont("OS", "B", 9)
		p.CellFormat(38, 7, x[0], "", 0, "L", false, 0, "")
		p.SetFont("OS", "", 9)
		p.CellFormat(0, 7, x[1], "", 1, "L", false, 0, "")
	}
	var b bytes.Buffer
	if e := p.Output(&b); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}
