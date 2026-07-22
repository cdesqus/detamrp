package receiving

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

func RenderPDF(r Receiving) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.AddUTF8FontFromBytes("OS", "", goregular.TTF)
	p.AddUTF8FontFromBytes("OS", "B", gobold.TTF)
	p.SetMargins(14, 14, 14)
	p.AddPage()
	p.SetFillColor(24, 24, 27)
	p.Rect(0, 0, 210, 30, "F")
	p.SetTextColor(255, 255, 255)
	p.SetXY(14, 9)
	p.SetFont("OS", "B", 17)
	p.CellFormat(120, 8, "GOODS RECEIVING", "", 0, "L", false, 0, "")
	p.SetFont("OS", "B", 10)
	p.CellFormat(60, 8, r.ReceivingNumber, "", 1, "R", false, 0, "")
	p.SetTextColor(24, 24, 27)
	p.SetY(38)
	fields := [][2]string{{"Delivery Note", r.DeliveryNoteNumber}, {"Purchase Order", r.PONumber}, {"Supplier", r.SupplierName}, {"Receiving Date", r.ReceivingDate.Format("02 Jan 2006")}, {"Completed By", r.CreatedBy}, {"Sage Receipt Number", emptyDash(r.SageReceiptNumber)}}
	p.SetFont("OS", "", 9)
	for _, f := range fields {
		p.SetFont("OS", "B", 9)
		p.CellFormat(42, 6, f[0], "", 0, "L", false, 0, "")
		p.SetFont("OS", "", 9)
		p.CellFormat(0, 6, f[1], "", 1, "L", false, 0, "")
	}
	p.Ln(4)
	p.SetFont("OS", "B", 9)
	p.SetFillColor(244, 244, 245)
	p.CellFormat(0, 7, "  RECEIVING SUMMARY", "", 1, "L", true, 0, "")
	p.SetFont("OS", "", 10)
	for _, f := range [][2]string{{"Planned", fmt.Sprint(r.Planned)}, {"Previously Received", fmt.Sprint(r.PreviouslyReceived)}, {"Received Now", fmt.Sprint(r.ReceivedNow)}, {"Outstanding", fmt.Sprint(r.Outstanding)}} {
		p.CellFormat(70, 7, f[0], "1", 0, "L", false, 0, "")
		p.CellFormat(30, 7, f[1], "1", 1, "R", false, 0, "")
	}
	if len(r.Scans) > 0 {
		p.Ln(5)
		p.SetFont("OS", "B", 9)
		p.CellFormat(45, 7, "KANBAN ID", "1", 0, "L", true, 0, "")
		p.CellFormat(72, 7, "RAW MATERIAL", "1", 0, "L", true, 0, "")
		p.CellFormat(35, 7, "QUANTITY", "1", 1, "R", true, 0, "")
		p.SetFont("OS", "", 8)
		for _, scan := range r.Scans {
			p.CellFormat(45, 7, scan.KanbanID, "1", 0, "L", false, 0, "")
			p.CellFormat(72, 7, scan.MaterialCode+" - "+scan.MaterialName, "1", 0, "L", false, 0, "")
			p.CellFormat(35, 7, scan.Quantity.String()+" "+scan.Unit, "1", 1, "R", false, 0, "")
		}
	}
	var b bytes.Buffer
	if err := p.Output(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
