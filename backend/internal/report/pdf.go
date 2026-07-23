package report

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

func RenderReceivingPDF(result Result, filter Filter) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetCompression(false)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 12)
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Arial", "B", 15)
		pdf.CellFormat(277, 8, "Receiving Report", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(277, 5, "Generated "+time.Now().Format("02 Jan 2006 15:04"), "", 1, "L", false, 0, "")
		writeReportHeader(pdf)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-9)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(277, 5, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	pdf.AddPage()
	if len(result.Items) == 0 {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(277, 20, "No receiving transactions found.", "", 1, "C", false, 0, "")
	} else {
		for _, row := range result.Items {
			values := []string{row.ReceivingNumber, row.ReceivingDate.Format("2006-01-02"), row.DeliveryNoteNumber, row.PONumber, row.SupplierName, row.RawMaterialCode + " - " + row.RawMaterialName, fmt.Sprint(row.KanbanReceived), row.ReceivedQuantity.String() + " " + row.BaseUnitCode, row.OutstandingQuantity.String() + " " + row.BaseUnitCode, row.SageNumber, row.CreatedBy}
			widths := reportColumnWidths()
			pdf.SetFont("Arial", "", 6.5)
			for i, value := range values {
				pdf.CellFormat(widths[i], 6, value, "1", 0, "L", false, 0, "")
			}
			pdf.Ln(-1)
		}
	}
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(175, 7, "TOTAL", "1", 0, "R", false, 0, "")
	pdf.CellFormat(14, 7, fmt.Sprint(result.Totals.KanbanReceived), "1", 0, "R", false, 0, "")
	pdf.CellFormat(88, 7, result.Totals.ReceivedQuantity.String(), "1", 1, "R", false, 0, "")
	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func reportColumnWidths() []float64 { return []float64{25, 18, 23, 23, 30, 42, 14, 24, 25, 22, 25} }

func writeReportHeader(pdf *fpdf.Fpdf) {
	headers := []string{"Receiving", "Date", "DN", "PO", "Supplier", "Raw Material", "KBN", "Received Qty", "Outstanding", "Sage No.", "Created By"}
	for i, header := range headers {
		pdf.SetFont("Arial", "B", 6.5)
		pdf.CellFormat(reportColumnWidths()[i], 7, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
}
