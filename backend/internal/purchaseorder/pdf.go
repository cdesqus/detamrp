package purchaseorder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/qr"
	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"order-stock/backend/internal/database"
)

type DeliveryNoteDocument struct {
	DeliveryNoteID       uuid.UUID
	DeliveryNoteNumber   string
	PurchaseOrderID      uuid.UUID
	PONumber             string
	SupplierName         string
	ExpectedDeliveryDate time.Time
	IssuedAt             time.Time
	Lines                []DeliveryNoteLine
}

type DeliveryNoteLine struct {
	RawMaterialCode string
	RawMaterialName string
	BaseUnitCode    string
	QtyPerKanban    decimal.Decimal
	TotalKanban     decimal.Decimal
	TotalQuantity   decimal.Decimal
}

type KanbanLabelDocument struct {
	DeliveryNoteID     uuid.UUID
	DeliveryNoteNumber string
	PurchaseOrderID    uuid.UUID
	PONumber           string
	SupplierName       string
	Labels             []KanbanLabel
}

type KanbanLabel struct {
	KanbanID        string
	RawMaterialCode string
	RawMaterialName string
	Quantity        decimal.Decimal
	BaseUnitCode    string
	LotNumber       int
}

const maxKanbanLabelsPerPDF = 1_000

const pdfFontFamily = "OrderStockSans"

func newA4PDF(title string) *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(false)
	pdf.SetTitle(title, true)
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddUTF8FontFromBytes(pdfFontFamily, "", goregular.TTF)
	pdf.AddUTF8FontFromBytes(pdfFontFamily, "B", gobold.TTF)
	return pdf
}

func outputPDF(pdf *fpdf.Fpdf) ([]byte, error) {
	var result bytes.Buffer
	if err := pdf.Output(&result); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func RenderPOPDF(order Order, includePrices bool) ([]byte, error) {
	pdf := newA4PDF(order.PONumber)
	pdf.AddPage()
	pdf.SetFillColor(24, 24, 27)
	pdf.Rect(0, 0, 210, 31, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(12, 8)
	pdf.SetFont(pdfFontFamily, "B", 18)
	pdf.CellFormat(115, 8, "PURCHASE ORDER", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", 11)
	pdf.CellFormat(71, 8, order.PONumber, "", 1, "R", false, 0, "")
	pdf.SetX(12)
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(115, 5, "ORDER STOCK  /  PROCUREMENT", "", 0, "L", false, 0, "")
	pdf.CellFormat(71, 5, string(order.Status), "", 1, "R", false, 0, "")
	pdf.SetTextColor(24, 24, 27)
	pdf.SetY(37)
	writePDFSectionTitle(pdf, "SUPPLIER DETAILS")
	writeKeyValue(pdf, "Supplier", order.SupplierName, 3)
	writePDFSectionTitle(pdf, "ORDER DETAILS")
	writeKeyValue(pdf, "Order Date", formatPDFDate(order.OrderDate), 2)
	writeKeyValue(pdf, "Expected Delivery", formatPDFDate(order.ExpectedDeliveryDate), 2)
	writeKeyValue(pdf, "Currency", order.Currency, 2)
	pdf.Ln(2)

	widths := []float64{23, 47, 15, 24, 20, 27}
	headings := []string{"Material", "Description", "Unit", "Qty/Kanban", "Kanbans", "Total Qty"}
	if includePrices {
		widths = []float64{20, 33, 12, 20, 16, 22, 28, 29}
		headings = []string{"Material", "Description", "Unit", "Qty/Kanban", "Kanbans", "Total Qty", "Unit Price", "Line Total"}
	}
	totalBaseQty := decimal.Zero
	rows := make([][]string, 0, len(order.Lines))
	for _, line := range order.Lines {
		values := []string{line.RawMaterialCode, line.RawMaterialName, line.BaseUnitCode, formatPDFDecimal(line.QtyPerKanbanSnapshot, 6), formatPDFDecimal(line.TotalKanban, 0), formatPDFDecimal(line.OrderedBaseQty, 6)}
		if includePrices {
			values = append(values, formatPDFMoney(line.UnitPriceSnapshot, order.Currency), formatPDFMoney(line.LineTotal, order.Currency))
		}
		rows = append(rows, values)
		totalBaseQty = totalBaseQty.Add(line.OrderedBaseQty)
	}
	writePDFTable(pdf, widths, headings, rows)
	pdf.Ln(3)
	writePDFSectionTitle(pdf, "ORDER SUMMARY")
	writeKeyValue(pdf, "Total Base Quantity", formatPDFDecimal(totalBaseQty, 6), 3)
	if includePrices {
		writeKeyValue(pdf, "Total Amount", formatPDFMoney(order.TotalAmount, order.Currency), 3)
	}
	if strings.TrimSpace(order.Notes) != "" {
		writeKeyValue(pdf, "Notes", order.Notes, 8)
	}
	writePDFSectionTitle(pdf, "APPROVAL")
	writeKeyValue(pdf, "Created By", order.CreatedBy.DisplayName, 3)
	approver := order.SubmittedApproverDisplayName
	if approver == "" {
		approver = "Pending approval"
	}
	writeKeyValue(pdf, "Approver", approver, 3)
	return outputPDF(pdf)
}

func formatPDFDecimal(value decimal.Decimal, maximumFractionDigits int) string {
	raw := value.StringFixed(int32(maximumFractionDigits))
	parts := strings.SplitN(raw, ".", 2)
	negative := strings.HasPrefix(parts[0], "-")
	whole := strings.TrimPrefix(parts[0], "-")
	for index := len(whole) - 3; index > 0; index -= 3 {
		whole = whole[:index] + "." + whole[index:]
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = strings.TrimRight(parts[1], "0")
	}
	result := whole
	if fraction != "" {
		result += "," + fraction
	}
	if negative && !value.IsZero() {
		result = "-" + result
	}
	return result
}

func formatPDFMoney(value decimal.Decimal, currency string) string {
	maximumFractionDigits := 2
	if strings.EqualFold(strings.TrimSpace(currency), "IDR") {
		maximumFractionDigits = 0
	}
	formatted := formatPDFDecimal(value, maximumFractionDigits)
	if strings.TrimSpace(currency) == "" {
		return formatted
	}
	return strings.TrimSpace(currency) + " " + formatted
}

func writePDFSectionTitle(pdf *fpdf.Fpdf, title string) {
	ensurePDFPageRoom(pdf, 9)
	pdf.SetFillColor(244, 244, 245)
	pdf.SetTextColor(39, 39, 42)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(0, 7, "  "+title, "", 1, "L", true, 0, "")
	pdf.Ln(1)
}

func RenderDeliveryNotePDF(document DeliveryNoteDocument) ([]byte, error) {
	pdf := newA4PDF(document.DeliveryNoteNumber)
	pdf.AddPage()
	qrPNG, err := deliveryNoteQRPNG(document.DeliveryNoteNumber)
	if err != nil {
		return nil, err
	}
	writeDeliveryNoteHeader(pdf, document, qrPNG)
	writePDFSectionTitle(pdf, "MATERIAL DETAILS")
	widths := []float64{10, 29, 55, 16, 27, 22, 27}
	headings := []string{"No.", "Material Code", "Description", "Unit", "Qty / Kanban", "Kanbans", "Total Qty"}
	rows := make([][]string, 0, len(document.Lines))
	for index, line := range document.Lines {
		rows = append(rows, []string{
			strconv.Itoa(index + 1),
			line.RawMaterialCode,
			line.RawMaterialName,
			line.BaseUnitCode,
			formatPDFDecimal(line.QtyPerKanban, 6),
			formatPDFDecimal(line.TotalKanban, 0),
			formatPDFDecimal(line.TotalQuantity, 6),
		})
	}
	writeDeliveryNoteTable(pdf, widths, headings, rows)
	writeDeliveryNoteFooter(pdf, document.Lines)
	return outputPDF(pdf)
}

func writeDeliveryNoteHeader(pdf *fpdf.Fpdf, document DeliveryNoteDocument, qrPNG []byte) {
	pdf.SetFillColor(24, 24, 27)
	pdf.Rect(0, 0, 210, 32, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(12, 7)
	pdf.SetFont(pdfFontFamily, "B", 12)
	pdf.CellFormat(55, 6, "Order Stock", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", 17)
	pdf.CellFormat(76, 7, "DELIVERY NOTE", "", 0, "C", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", 9)
	pdf.CellFormat(55, 6, document.DeliveryNoteNumber, "", 1, "R", false, 0, "")
	pdf.SetX(67)
	pdf.SetFont(pdfFontFamily, "", 7)
	pdf.CellFormat(76, 5, "SUPPLIER SHIPPING DOCUMENT", "", 0, "C", false, 0, "")
	pdf.SetTextColor(24, 24, 27)

	pdf.SetY(38)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(92, 6, "DELIVERY INFORMATION", "", 0, "L", false, 0, "")
	pdf.SetX(151)
	pdf.CellFormat(47, 6, "SCAN FOR RECEIVING", "", 1, "C", false, 0, "")
	pdf.SetY(45)
	writeDeliveryNoteMeta(pdf, "Supplier", document.SupplierName)
	writeDeliveryNoteMeta(pdf, "PO Number", document.PONumber)
	writeDeliveryNoteMeta(pdf, "Expected Delivery", formatPDFDate(document.ExpectedDeliveryDate))
	writeDeliveryNoteMeta(pdf, "Issued Date", formatPDFDate(document.IssuedAt))

	imageName := "delivery-note-qr"
	pdf.RegisterImageOptionsReader(imageName, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, bytes.NewReader(qrPNG))
	pdf.ImageOptions(imageName, 158, 47, 32, 32, false, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, 0, "")
	pdf.SetXY(151, 80)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(47, 4, document.DeliveryNoteNumber, "", 0, "C", false, 0, "")
	pdf.SetY(88)
}

func writeDeliveryNoteMeta(pdf *fpdf.Fpdf, label, value string) {
	y := pdf.GetY()
	pdf.SetX(12)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(34, 6, label, "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(94, 6, value, "", 1, "L", false, 0, "")
	pdf.SetY(y + 7)
}

func deliveryNoteUnitTotals(lines []DeliveryNoteLine) []string {
	totals := make(map[string]decimal.Decimal)
	for _, line := range lines {
		unit := strings.ToUpper(strings.TrimSpace(line.BaseUnitCode))
		totals[unit] = totals[unit].Add(line.TotalQuantity)
	}
	units := make([]string, 0, len(totals))
	for unit := range totals {
		units = append(units, unit)
	}
	sort.Strings(units)
	result := make([]string, 0, len(units))
	for _, unit := range units {
		result = append(result, strings.TrimSpace(unit+" "+formatPDFDecimal(totals[unit], 6)))
	}
	return result
}

func writeDeliveryNoteFooter(pdf *fpdf.Fpdf, lines []DeliveryNoteLine) {
	const footerHeight = 75.0
	ensurePDFPageRoom(pdf, footerHeight)
	pdf.Ln(2)

	totalKanban := decimal.Zero
	for _, line := range lines {
		totalKanban = totalKanban.Add(line.TotalKanban)
	}
	left, _, _, _ := pdf.GetMargins()
	pdf.SetX(left)
	pdf.SetFillColor(244, 244, 245)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(93, 7, "Total Kanban", "1", 0, "L", true, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(93, 7, formatPDFDecimal(totalKanban, 0), "1", 1, "R", false, 0, "")
	pdf.SetX(left)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(93, 7, "Total Quantity", "1", 0, "L", true, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(93, 7, strings.Join(deliveryNoteUnitTotals(lines), "  |  "), "1", 1, "R", false, 0, "")

	pdf.Ln(3)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(186, 6, "REMARKS", "1", 1, "L", true, 0, "")
	pdf.CellFormat(186, 12, "", "1", 1, "L", false, 0, "")
	pdf.Ln(4)
	writeDeliveryNoteSignatureBoxes(pdf)
}

func writeDeliveryNoteTable(pdf *fpdf.Fpdf, widths []float64, headings []string, rows [][]string) {
	writeHeader := func() {
		pdf.SetFont(pdfFontFamily, "B", 7)
		pdf.SetFillColor(244, 244, 245)
		for index, heading := range headings {
			pdf.CellFormat(widths[index], 7, heading, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
	}
	writeHeader()
	for _, values := range rows {
		pdf.SetFont(pdfFontFamily, "", 7)
		linesByCell := make([][]string, len(values))
		rowHeight := 7.0
		for index, value := range values {
			linesByCell[index] = fitPDFTextLines(pdf, value, widths[index]-2, 3)
			rowHeight = max(rowHeight, 2+float64(len(linesByCell[index]))*4)
		}
		if !pdfPageHasRoom(pdf, rowHeight) {
			pdf.AddPage()
			pdf.SetFont(pdfFontFamily, "B", 9)
			pdf.CellFormat(0, 7, "DELIVERY NOTE / CONTINUED", "", 1, "L", false, 0, "")
			pdf.SetFont(pdfFontFamily, "", 7)
			pdf.CellFormat(0, 5, "Material details continued", "", 1, "L", false, 0, "")
			pdf.Ln(2)
			writeHeader()
			pdf.SetFont(pdfFontFamily, "", 7)
		}
		x, y := pdf.GetX(), pdf.GetY()
		for index, lines := range linesByCell {
			pdf.Rect(x, y, widths[index], rowHeight, "D")
			align := "L"
			if index == 0 || index >= 3 {
				align = "C"
			}
			writePDFTextLines(pdf, x+1, y+1, widths[index]-2, 4, lines, align)
			x += widths[index]
		}
		left, _, _, _ := pdf.GetMargins()
		pdf.SetXY(left, y+rowHeight)
	}
}

func writeDeliveryNoteSignatureBoxes(pdf *fpdf.Fpdf) {
	left, _, _, _ := pdf.GetMargins()
	baseY := pdf.GetY()
	xPositions := []float64{left, left + 96}
	headings := []string{"SUPPLIER", "RECEIVER"}
	subheadings := []string{"Prepared By", "Received By"}
	for index, x := range xPositions {
		y := baseY
		pdf.SetXY(x, y)
		pdf.SetFillColor(244, 244, 245)
		pdf.SetFont(pdfFontFamily, "B", 8)
		pdf.CellFormat(90, 6, headings[index], "1", 1, "C", true, 0, "")
		pdf.SetXY(x, y+6)
		pdf.SetFont(pdfFontFamily, "", 7)
		pdf.CellFormat(90, 5, subheadings[index], "1", 1, "C", false, 0, "")
		pdf.Rect(x, y+11, 90, 17, "D")
		pdf.SetXY(x+2, y+28)
		pdf.CellFormat(43, 5, "Name:", "1", 0, "L", false, 0, "")
		pdf.CellFormat(45, 5, "Date:", "1", 0, "L", false, 0, "")
	}
	pdf.SetXY(left, baseY+34)
}

func encodeDeliveryNoteQR(value string) (barcode.Barcode, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("delivery note number is required")
	}
	encoded, err := qr.Encode(value, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("encode delivery note QR %q: %w", value, err)
	}
	scaled, err := barcode.Scale(encoded, 220, 220)
	if err != nil {
		return nil, fmt.Errorf("scale delivery note QR %q: %w", value, err)
	}
	return scaled, nil
}

func deliveryNoteQRPNG(value string) ([]byte, error) {
	encoded, err := encodeDeliveryNoteQR(value)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := png.Encode(&output, encoded); err != nil {
		return nil, fmt.Errorf("render delivery note QR %q: %w", value, err)
	}
	return output.Bytes(), nil
}

func encodeKanbanQR(value string) (barcode.Barcode, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("Kanban ID is required")
	}
	encoded, err := qr.Encode(value, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("encode Kanban QR %q: %w", value, err)
	}
	scaled, err := barcode.Scale(encoded, 260, 260)
	if err != nil {
		return nil, fmt.Errorf("scale Kanban QR %q: %w", value, err)
	}
	return scaled, nil
}

func kanbanQRPNG(value string) ([]byte, error) {
	encoded, err := encodeKanbanQR(value)
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := png.Encode(&output, encoded); err != nil {
		return nil, fmt.Errorf("render Kanban QR %q: %w", value, err)
	}
	return output.Bytes(), nil
}

var encodeKanbanBarcode = func(value string) (image.Image, error) {
	encoded, err := code128.Encode(value)
	if err != nil {
		return nil, err
	}
	return barcode.Scale(encoded, 420, 72)
}

var encodeKanbanQRImage = func(value string) (image.Image, error) {
	return encodeKanbanQR(value)
}

func RenderKanbanLabelsPDF(document KanbanLabelDocument) ([]byte, error) {
	return renderKanbanLabelsPDF(context.Background(), document)
}

func renderKanbanLabelsPDF(ctx context.Context, document KanbanLabelDocument) ([]byte, error) {
	if err := validateKanbanLabelExportSize(document.Labels); err != nil {
		return nil, err
	}
	pdf := newA4PDF("KANBAN-" + document.DeliveryNoteNumber)
	pdf.SetAutoPageBreak(false, 0)
	const cardsPerPage = 3
	const cardX, cardWidth, cardHeight = 12.0, 186.0, 84.0
	const cardGap, firstCardY = 5.0, 12.0

	for index, label := range document.Labels {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		slot := index % cardsPerPage
		if slot == 0 {
			pdf.AddPage()
		}
		qrImage, err := encodeKanbanQRImage(label.KanbanID)
		if err != nil {
			return nil, fmt.Errorf("encode Kanban QR %q: %w", label.KanbanID, err)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var qrPNG bytes.Buffer
		if err := png.Encode(&qrPNG, qrImage); err != nil {
			return nil, fmt.Errorf("render Kanban QR %q: %w", label.KanbanID, err)
		}
		y := firstCardY + float64(slot)*(cardHeight+cardGap)
		if err := writeKanbanCard(pdf, document, label, index, cardX, y, cardWidth, cardHeight, qrPNG.Bytes()); err != nil {
			return nil, err
		}
		if slot < cardsPerPage-1 && index+1 < len(document.Labels) {
			writeKanbanCutLine(pdf, y+cardHeight+cardGap/2)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return outputPDF(pdf)
}

func writeKanbanCard(pdf *fpdf.Fpdf, document KanbanLabelDocument, label KanbanLabel, index int, x, y, width, height float64, qrPNG []byte) error {
	pdf.SetDrawColor(39, 39, 42)
	pdf.SetLineWidth(0.35)
	pdf.Rect(x, y, width, height, "D")

	const headerHeight = 15.0
	pdf.SetFillColor(24, 24, 27)
	pdf.Rect(x, y, width, headerHeight, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(x+4, y+3)
	pdf.SetFont(pdfFontFamily, "B", 15)
	pdf.CellFormat(82, 8, "KANBAN CARD", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 7)
	pdf.CellFormat(32, 8, "KANBAN ID", "", 0, "R", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", 12)
	pdf.CellFormat(64, 8, label.KanbanID, "", 0, "R", false, 0, "")
	pdf.SetTextColor(24, 24, 27)

	qrX, qrY, qrSize := x+5, y+20, 43.0
	pdf.Rect(qrX-1, qrY-1, qrSize+2, qrSize+2, "D")
	imageName := "kanban-qr-" + strconv.Itoa(index)
	pdf.RegisterImageOptionsReader(imageName, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, bytes.NewReader(qrPNG))
	pdf.ImageOptions(imageName, qrX, qrY, qrSize, qrSize, false, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, 0, "")
	pdf.SetXY(qrX-1, qrY+qrSize+2)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(qrSize+2, 5, label.KanbanID, "", 0, "C", false, 0, "")

	detailX := x + 53
	detailWidth := width - 58
	pdf.SetXY(detailX, y+20)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(detailWidth, 5, "RAW MATERIAL", "", 1, "L", false, 0, "")
	pdf.SetXY(detailX, y+25)
	pdf.SetFont(pdfFontFamily, "B", 13)
	pdf.CellFormat(detailWidth, 7, label.RawMaterialCode, "", 1, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	description := fitPDFTextLines(pdf, label.RawMaterialName, detailWidth, 2)
	writePDFTextLines(pdf, detailX, y+33, detailWidth, 5, description, "L")

	infoY := y + 47
	columnWidth := detailWidth / 2
	pdf.Rect(detailX, infoY, columnWidth, 17, "D")
	pdf.Rect(detailX+columnWidth, infoY, columnWidth, 17, "D")
	pdf.SetXY(detailX+2, infoY+2)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(columnWidth-4, 4, "QUANTITY", "", 1, "L", false, 0, "")
	pdf.SetXY(detailX+2, infoY+7)
	pdf.SetFont(pdfFontFamily, "B", 13)
	pdf.CellFormat(columnWidth-4, 7, formatPDFDecimal(label.Quantity, 6)+" "+label.BaseUnitCode, "", 0, "L", false, 0, "")
	pdf.SetXY(detailX+columnWidth+2, infoY+2)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(columnWidth-4, 4, "LOT", "", 1, "L", false, 0, "")
	pdf.SetXY(detailX+columnWidth+2, infoY+7)
	pdf.SetFont(pdfFontFamily, "B", 13)
	pdf.CellFormat(columnWidth-4, 7, strconv.Itoa(label.LotNumber), "", 0, "L", false, 0, "")

	referenceY := y + 67
	pdf.SetXY(detailX, referenceY)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(25, 5, "DELIVERY NOTE", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(39, 5, document.DeliveryNoteNumber, "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(27, 5, "PURCHASE ORDER", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(detailWidth-91, 5, document.PONumber, "", 1, "L", false, 0, "")
	return pdf.Error()
}

func writeKanbanCutLine(pdf *fpdf.Fpdf, y float64) {
	pdf.SetDrawColor(113, 113, 122)
	pdf.SetDashPattern([]float64{2, 2}, 0)
	pdf.Line(12, y, 198, y)
	pdf.SetDashPattern([]float64{}, 0)
	pdf.SetTextColor(113, 113, 122)
	pdf.SetXY(84, y-2.5)
	pdf.SetFont(pdfFontFamily, "", 6)
	pdf.CellFormat(42, 5, "CUT HERE", "", 0, "C", false, 0, "")
	pdf.SetTextColor(24, 24, 27)
}

func validateKanbanLabelExportSize(labels []KanbanLabel) error {
	if len(labels) > maxKanbanLabelsPerPDF {
		return DocumentExportLimitError{DocumentType: "Kanban label", Limit: maxKanbanLabelsPerPDF}
	}
	return nil
}

func writeKeyValue(pdf *fpdf.Fpdf, key, value string, maxLines int) {
	const keyWidth, lineHeight = 38.0, 5.0
	pageWidth, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	valueWidth := pageWidth - left - right - keyWidth
	pdf.SetFont(pdfFontFamily, "", 9)
	lines := fitPDFTextLines(pdf, value, valueWidth, maxLines)
	rowHeight := max(lineHeight, float64(len(lines))*lineHeight)
	ensurePDFPageRoom(pdf, rowHeight)
	x, y := pdf.GetX(), pdf.GetY()
	pdf.SetFont(pdfFontFamily, "B", 9)
	pdf.SetXY(x, y)
	pdf.CellFormat(keyWidth, lineHeight, key, "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 9)
	writePDFTextLines(pdf, x+keyWidth, y, valueWidth, lineHeight, lines, "L")
	pdf.SetXY(left, y+rowHeight)
}

func writePDFTable(pdf *fpdf.Fpdf, widths []float64, headings []string, rows [][]string) {
	writeHeader := func() {
		pdf.SetFont(pdfFontFamily, "B", 8)
		for index, heading := range headings {
			pdf.CellFormat(widths[index], 7, heading, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)
	}
	writeHeader()
	for _, values := range rows {
		pdf.SetFont(pdfFontFamily, "", 8)
		linesByCell := make([][]string, len(values))
		rowHeight := 7.0
		for index, value := range values {
			linesByCell[index] = fitPDFTextLines(pdf, value, widths[index], 3)
			rowHeight = max(rowHeight, 2+float64(len(linesByCell[index]))*4)
		}
		if !pdfPageHasRoom(pdf, rowHeight) {
			pdf.AddPage()
			writeHeader()
			pdf.SetFont(pdfFontFamily, "", 8)
		}
		x, y := pdf.GetX(), pdf.GetY()
		for index, lines := range linesByCell {
			pdf.Rect(x, y, widths[index], rowHeight, "D")
			writePDFTextLines(pdf, x, y+1, widths[index], 4, lines, "L")
			x += widths[index]
		}
		left, _, _, _ := pdf.GetMargins()
		pdf.SetXY(left, y+rowHeight)
	}
}

func writePDFTextLines(pdf *fpdf.Fpdf, x, y, width, lineHeight float64, lines []string, align string) {
	for index, line := range lines {
		pdf.SetXY(x, y+float64(index)*lineHeight)
		pdf.CellFormat(width, lineHeight, line, "", 0, align, false, 0, "")
	}
}

func fitPDFTextLines(pdf *fpdf.Fpdf, value string, width float64, maxLines int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{""}
	}
	lines := pdf.SplitText(value, width)
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	lines = append([]string(nil), lines[:maxLines]...)
	last := strings.TrimSpace(lines[maxLines-1])
	for {
		candidate := strings.TrimSpace(last) + "…"
		if pdf.GetStringWidth(candidate) <= width-2 || last == "" {
			lines[maxLines-1] = candidate
			return lines
		}
		characters := []rune(last)
		last = string(characters[:len(characters)-1])
	}
}

func ensurePDFPageRoom(pdf *fpdf.Fpdf, height float64) {
	if !pdfPageHasRoom(pdf, height) {
		pdf.AddPage()
	}
}

func pdfPageHasRoom(pdf *fpdf.Fpdf, height float64) bool {
	_, pageHeight := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	return pdf.GetY()+height <= pageHeight-bottom
}

func formatPDFDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("02 Jan 2006")
}

type documentQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type documentHeader struct {
	PONumber             string
	SupplierName         string
	ExpectedDeliveryDate time.Time
	Status               Status
	DeliveryNoteID       uuid.UUID
	DeliveryNoteNumber   string
	IssuedAt             time.Time
}

func (s *SQLStore) LoadDeliveryNoteDocument(ctx context.Context, actor Actor, purchaseOrderID uuid.UUID) (document DeliveryNoteDocument, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		document, err = loadDeliveryNoteDocument(ctx, tx, actor, purchaseOrderID)
		return err
	})
	return
}

func (s *SQLStore) LoadKanbanLabelDocument(ctx context.Context, actor Actor, purchaseOrderID uuid.UUID) (document KanbanLabelDocument, err error) {
	err = database.WithTenant(ctx, s.db, tenantContext(actor), func(tx database.TenantTx) error {
		document, err = loadKanbanLabelDocument(ctx, tx, actor, purchaseOrderID)
		return err
	})
	return
}

func loadDeliveryNoteDocument(ctx context.Context, query documentQuerier, actor Actor, purchaseOrderID uuid.UUID) (DeliveryNoteDocument, error) {
	header, err := loadOperationalDocumentHeader(ctx, query, actor.TenantID, purchaseOrderID, "deliveryNote")
	if err != nil {
		return DeliveryNoteDocument{}, err
	}
	document := DeliveryNoteDocument{
		DeliveryNoteID: header.DeliveryNoteID, DeliveryNoteNumber: header.DeliveryNoteNumber,
		PurchaseOrderID: purchaseOrderID, PONumber: header.PONumber, SupplierName: header.SupplierName,
		ExpectedDeliveryDate: header.ExpectedDeliveryDate, IssuedAt: header.IssuedAt,
	}
	rows, err := query.Query(ctx, `SELECT pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,pol.base_unit_code_snapshot,
 pol.qty_per_kanban_snapshot,pol.total_kanban,pol.ordered_base_qty
 FROM delivery_note_lines dnl
 JOIN purchase_order_lines pol ON pol.tenant_id=dnl.tenant_id AND pol.purchase_order_id=dnl.purchase_order_id AND pol.id=dnl.purchase_order_line_id
 WHERE dnl.tenant_id=$1 AND dnl.purchase_order_id=$2
 ORDER BY pol.sort_position`, actor.TenantID, purchaseOrderID)
	if err != nil {
		return DeliveryNoteDocument{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var line DeliveryNoteLine
		if err := rows.Scan(&line.RawMaterialCode, &line.RawMaterialName, &line.BaseUnitCode, &line.QtyPerKanban, &line.TotalKanban, &line.TotalQuantity); err != nil {
			return DeliveryNoteDocument{}, err
		}
		document.Lines = append(document.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return DeliveryNoteDocument{}, err
	}
	if len(document.Lines) == 0 {
		return DeliveryNoteDocument{}, documentConflict("deliveryNote", "Delivery Note lines are unavailable")
	}
	return document, nil
}

func loadKanbanLabelDocument(ctx context.Context, query documentQuerier, actor Actor, purchaseOrderID uuid.UUID) (KanbanLabelDocument, error) {
	header, err := loadOperationalDocumentHeader(ctx, query, actor.TenantID, purchaseOrderID, "kanbanLabels")
	if err != nil {
		return KanbanLabelDocument{}, err
	}
	document := KanbanLabelDocument{
		DeliveryNoteID: header.DeliveryNoteID, DeliveryNoteNumber: header.DeliveryNoteNumber,
		PurchaseOrderID: purchaseOrderID, PONumber: header.PONumber, SupplierName: header.SupplierName,
	}
	rows, err := query.Query(ctx, `SELECT kl.kanban_id,pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,
 kl.quantity,pol.base_unit_code_snapshot,kl.lot_number
 FROM delivery_note_lines dnl
 JOIN purchase_order_lines pol ON pol.tenant_id=dnl.tenant_id AND pol.purchase_order_id=dnl.purchase_order_id AND pol.id=dnl.purchase_order_line_id
 JOIN kanban_lots kl ON kl.tenant_id=dnl.tenant_id AND kl.delivery_note_line_id=dnl.id AND kl.purchase_order_line_id=pol.id
 WHERE dnl.tenant_id=$1 AND dnl.purchase_order_id=$2
 ORDER BY pol.sort_position,kl.lot_number
 LIMIT $3`, actor.TenantID, purchaseOrderID, maxKanbanLabelsPerPDF+1)
	if err != nil {
		return KanbanLabelDocument{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var label KanbanLabel
		if err := rows.Scan(&label.KanbanID, &label.RawMaterialCode, &label.RawMaterialName, &label.Quantity, &label.BaseUnitCode, &label.LotNumber); err != nil {
			return KanbanLabelDocument{}, err
		}
		document.Labels = append(document.Labels, label)
	}
	if err := rows.Err(); err != nil {
		return KanbanLabelDocument{}, err
	}
	if err := validateKanbanLabelExportSize(document.Labels); err != nil {
		return KanbanLabelDocument{}, err
	}
	if len(document.Labels) == 0 {
		return KanbanLabelDocument{}, documentConflict("kanbanLabels", "Kanban labels are unavailable")
	}
	return document, nil
}

func loadOperationalDocumentHeader(ctx context.Context, query documentQuerier, tenantID, purchaseOrderID uuid.UUID, field string) (documentHeader, error) {
	var header documentHeader
	err := query.QueryRow(ctx, `SELECT p.po_number,s.name,p.expected_delivery_date,p.status
 FROM purchase_orders p
 JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id
 WHERE p.tenant_id=$1 AND p.id=$2`, tenantID, purchaseOrderID).
		Scan(&header.PONumber, &header.SupplierName, &header.ExpectedDeliveryDate, &header.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return documentHeader{}, NotFoundError{Resource: "purchase order"}
	}
	if err != nil {
		return documentHeader{}, err
	}
	if header.Status != StatusApproved && header.Status != StatusPartiallyReceived && header.Status != StatusFullyReceived {
		return documentHeader{}, documentConflict(field, "Purchase order operational documents are unavailable")
	}
	err = query.QueryRow(ctx, `SELECT dn.id,dn.delivery_note_number,dn.issued_at
 FROM delivery_notes dn WHERE dn.tenant_id=$1 AND dn.purchase_order_id=$2`, tenantID, purchaseOrderID).
		Scan(&header.DeliveryNoteID, &header.DeliveryNoteNumber, &header.IssuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return documentHeader{}, documentConflict(field, "Operational document is unavailable")
	}
	return header, err
}

func documentConflict(field, message string) ConflictError {
	return ConflictError{Fields: FieldErrors{field: message}}
}
