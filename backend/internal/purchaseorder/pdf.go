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
	CompanyName          string
	SupplierName         string
	PlantCode            string
	PlantName            string
	PlantAddress         string
	OrderDate            time.Time
	ExpectedDeliveryDate time.Time
	IssuedAt             time.Time
	Lines                []DeliveryNoteLine
}

type DeliveryNoteLine struct {
	RawMaterialCode string
	RawMaterialName string
	BaseUnitCode    string
	CategoryCode    string
	CategoryName    string
	PackingCode     string
	PackingName     string
	QtyPerKanban    decimal.Decimal
	TotalKanban     decimal.Decimal
	TotalQuantity   decimal.Decimal
}

type KanbanLabelDocument struct {
	DeliveryNoteID     uuid.UUID
	DeliveryNoteNumber string
	PurchaseOrderID    uuid.UUID
	PONumber           string
	CompanyName        string
	SupplierName       string
	PlantCode          string
	PlantName          string
	PlantAddress       string
	OrderDate          time.Time
	Labels             []KanbanLabel
}

type KanbanLabel struct {
	KanbanID        string
	RawMaterialCode string
	RawMaterialName string
	Quantity        decimal.Decimal
	BaseUnitCode    string
	CategoryCode    string
	CategoryName    string
	PackingCode     string
	PackingName     string
	CardNumber      int
	CardTotal       int
}

const maxKanbanLabelsPerPDF = 1_000

const pdfFontFamily = "OrderStockSans"
const deliveryNoteNumberFontSize = 17.0

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

func pdfCompanyName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Order Stock"
	}
	return value
}

func RenderPOPDF(order Order, includePrices bool) ([]byte, error) {
	pdf := newA4PDF(order.PONumber)
	pdf.AddPage()
	setDocumentInk(pdf)
	pdf.SetXY(12, 10)
	pdf.SetFont(pdfFontFamily, "B", 9)
	pdf.CellFormat(112, 5, pdfCompanyName(order.CompanyName), "", 0, "L", false, 0, "")
	pdf.SetXY(12, 18)
	pdf.SetFont(pdfFontFamily, "B", 17)
	pdf.CellFormat(110, 8, "PURCHASE ORDER", "", 0, "L", false, 0, "")
	pdf.SetXY(124, 10)
	pdf.SetFont(pdfFontFamily, "B", 13)
	pdf.CellFormat(74, 7, order.PONumber, "", 0, "R", false, 0, "")
	pdf.SetXY(124, 20)
	pdf.SetFont(pdfFontFamily, "", 7)
	pdf.CellFormat(72, 4, string(order.Status), "", 0, "R", false, 0, "")
	pdf.SetLineWidth(0.55)
	pdf.Line(12, 31, 198, 31)
	pdf.SetLineWidth(0.25)

	writePOPartyDetails(pdf, order)
	writePDFSectionTitle(pdf, "ORDER DETAILS")
	writePOOrderMeta(pdf, order)

	writePDFSectionTitle(pdf, "MATERIAL DETAILS")
	widths := []float64{22, 36, 25, 23, 12, 23, 18, 27}
	headings := []string{"Material", "Description", "Category", "Packing", "Unit", "Qty/Card", "Cards", "Total Qty"}
	if includePrices {
		widths = []float64{17, 28, 19, 18, 10, 18, 14, 18, 21, 23}
		headings = []string{"Material", "Description", "Category", "Packing", "Unit", "Qty/Card", "Cards", "Total", "Unit Price", "Amount"}
	}
	totalBaseQty := decimal.Zero
	rows := make([][]string, 0, len(order.Lines))
	for _, line := range order.Lines {
		values := []string{
			line.RawMaterialCode, line.RawMaterialName,
			pdfReferenceName(line.CategoryCode, line.CategoryName),
			pdfReferenceName(line.PackingCode, line.PackingName),
			line.BaseUnitCode, formatPDFDecimal(line.QtyPerKanbanSnapshot, 6),
			formatPDFDecimal(line.TotalKanban, 0), formatPDFDecimal(line.OrderedBaseQty, 6),
		}
		if includePrices {
			values = append(values, formatPDFMoney(line.UnitPriceSnapshot, order.Currency), formatPDFMoney(line.LineTotal, order.Currency))
		}
		rows = append(rows, values)
		totalBaseQty = totalBaseQty.Add(line.OrderedBaseQty)
	}
	writePDFTable(pdf, widths, headings, rows)
	pdf.Ln(4)
	writePDFSectionTitle(pdf, "ORDER SUMMARY")
	writePOSummary(pdf, totalBaseQty, order, includePrices)
	if strings.TrimSpace(order.Notes) != "" {
		pdf.Ln(2)
		writeKeyValue(pdf, "Notes", order.Notes, 8)
	}
	pdf.Ln(4)
	writePDFSectionTitle(pdf, "APPROVAL")
	approver := order.SubmittedApproverDisplayName
	if approver == "" {
		approver = "Pending approval"
	}
	writePOApproval(pdf, order.CreatedBy.DisplayName, approver)
	return outputPDF(pdf)
}

func setDocumentInk(pdf *fpdf.Fpdf) {
	pdf.SetDrawColor(39, 39, 42)
	pdf.SetTextColor(24, 24, 27)
	pdf.SetLineWidth(0.25)
}

func pdfReferenceName(code, name string) string {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	switch {
	case code == "":
		return name
	case name == "":
		return code
	default:
		return code + " — " + name
	}
}

func writePOPartyDetails(pdf *fpdf.Fpdf, order Order) {
	const top, columnWidth = 38.0, 90.0
	writePOPartyColumn(pdf, 12, top, columnWidth, "SUPPLIER DETAILS", order.SupplierName, "")
	writePOPartyColumn(pdf, 108, top, columnWidth, "DESTINATION PLANT",
		pdfReferenceName(order.PlantCode, order.PlantName), order.PlantAddress)
	pdf.SetY(62)
}

func writePOPartyColumn(pdf *fpdf.Fpdf, x, y, width float64, heading, primary, secondary string) {
	pdf.SetXY(x, y)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(width, 5, heading, "", 0, "L", false, 0, "")
	pdf.Line(x, y+6, x+width, y+6)
	pdf.SetFont(pdfFontFamily, "B", 9)
	writePDFTextLines(pdf, x, y+9, width-2, 5, fitPDFTextLines(pdf, primary, width-2, 2), "L")
	if strings.TrimSpace(secondary) != "" {
		pdf.SetFont(pdfFontFamily, "", 7)
		writePDFTextLines(pdf, x, y+18, width-2, 4, fitPDFTextLines(pdf, secondary, width-2, 1), "L")
	}
}

func writePOOrderMeta(pdf *fpdf.Fpdf, order Order) {
	const y, width = 72.0, 58.0
	values := [][2]string{
		{"Order Date", formatPDFDate(order.OrderDate)},
		{"Expected Delivery", formatPDFDate(order.ExpectedDeliveryDate)},
		{"Currency", order.Currency},
	}
	for index, value := range values {
		x := 12 + float64(index)*64
		pdf.SetXY(x, y)
		pdf.SetFont(pdfFontFamily, "", 7)
		pdf.CellFormat(width, 4, value[0], "", 0, "L", false, 0, "")
		pdf.SetXY(x, y+5)
		pdf.SetFont(pdfFontFamily, "B", 9)
		pdf.CellFormat(width, 5, value[1], "", 0, "L", false, 0, "")
		pdf.Line(x, y+11, x+width, y+11)
	}
	pdf.SetY(y + 16)
}

func writePOSummary(pdf *fpdf.Fpdf, totalBaseQty decimal.Decimal, order Order, includePrices bool) {
	left, _, _, _ := pdf.GetMargins()
	rows := [][2]string{{"Total Base Quantity", formatPDFDecimal(totalBaseQty, 6)}}
	if includePrices {
		rows = append(rows, [2]string{"Total Amount", formatPDFMoney(order.TotalAmount, order.Currency)})
	}
	for _, row := range rows {
		pdf.SetX(left + 92)
		pdf.SetFont(pdfFontFamily, "", 8)
		pdf.CellFormat(44, 7, row[0], "", 0, "L", false, 0, "")
		pdf.SetFont(pdfFontFamily, "B", 9)
		pdf.CellFormat(50, 7, row[1], "", 1, "R", false, 0, "")
		pdf.Line(left+92, pdf.GetY(), left+186, pdf.GetY())
	}
}

func writePOApproval(pdf *fpdf.Fpdf, createdBy, approver string) {
	left, _, _, _ := pdf.GetMargins()
	y := pdf.GetY() + 2
	for index, value := range [][2]string{{"Created By", createdBy}, {"Approver", approver}} {
		x := left + float64(index)*96
		pdf.SetXY(x, y)
		pdf.SetFont(pdfFontFamily, "", 7)
		pdf.CellFormat(90, 5, value[0], "", 0, "C", false, 0, "")
		pdf.Line(x+8, y+22, x+82, y+22)
		pdf.SetXY(x, y+23)
		pdf.SetFont(pdfFontFamily, "B", 8)
		pdf.CellFormat(90, 5, value[1], "", 0, "C", false, 0, "")
	}
	pdf.SetY(y + 30)
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
	pdf.SetDrawColor(24, 24, 27)
	pdf.SetTextColor(39, 39, 42)
	pdf.SetFont(pdfFontFamily, "B", 8)
	x, y := pdf.GetX(), pdf.GetY()
	pdf.CellFormat(0, 6, title, "", 1, "L", false, 0, "")
	pdf.Line(x, y+6, 198, y+6)
	pdf.Ln(2)
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
	widths := []float64{8, 21, 38, 24, 22, 11, 22, 16, 24}
	headings := []string{"No.", "Material", "Description", "Category", "Packing", "Unit", "Qty/Card", "Cards", "Total Qty"}
	rows := make([][]string, 0, len(document.Lines))
	for index, line := range document.Lines {
		rows = append(rows, []string{
			strconv.Itoa(index + 1),
			line.RawMaterialCode,
			line.RawMaterialName,
			pdfReferenceName(line.CategoryCode, line.CategoryName),
			pdfReferenceName(line.PackingCode, line.PackingName),
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
	setDocumentInk(pdf)
	pdf.SetXY(12, 9)
	pdf.SetFont(pdfFontFamily, "B", 9)
	companyLines := fitPDFTextLines(pdf, pdfCompanyName(document.CompanyName), 52, 2)
	writePDFTextLines(pdf, 12, 9, 52, 5, companyLines, "L")
	pdf.SetXY(65, 10)
	pdf.SetFont(pdfFontFamily, "B", 13)
	pdf.CellFormat(68, 7, "DELIVERY NOTE", "", 0, "C", false, 0, "")
	pdf.SetXY(65, 19)
	pdf.SetFont(pdfFontFamily, "", 7)
	pdf.CellFormat(68, 5, "SUPPLIER SHIPPING DOCUMENT", "", 0, "C", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", deliveryNoteNumberFontSize)
	numberLines := fitPDFTextLines(pdf, document.DeliveryNoteNumber, 62, 2)
	writePDFTextLines(pdf, 136, 11, 62, 7, numberLines, "R")
	pdf.Line(12, 32, 198, 32)

	pdf.SetY(38)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(128, 6, "DELIVERY INFORMATION", "", 0, "L", false, 0, "")
	pdf.SetX(151)
	pdf.CellFormat(47, 6, "SCAN FOR RECEIVING", "", 1, "C", false, 0, "")
	pdf.Line(12, 44, 140, 44)
	writeDeliveryNoteParty(pdf, 12, 48, 60, "SUPPLIER", document.SupplierName, "")
	writeDeliveryNoteParty(pdf, 78, 48, 62, "DESTINATION PLANT",
		pdfReferenceName(document.PlantCode, document.PlantName), document.PlantAddress)
	writeDeliveryNoteDates(pdf, document)

	imageName := "delivery-note-qr"
	pdf.RegisterImageOptionsReader(imageName, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, bytes.NewReader(qrPNG))
	pdf.ImageOptions(imageName, 158, 47, 32, 32, false, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, 0, "")
	pdf.SetXY(151, 80)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(47, 4, document.DeliveryNoteNumber, "", 0, "C", false, 0, "")
	pdf.SetY(96)
}

func writeDeliveryNoteParty(pdf *fpdf.Fpdf, x, y, width float64, label, primary, secondary string) {
	pdf.SetXY(x, y)
	pdf.SetFont(pdfFontFamily, "", 6.5)
	pdf.CellFormat(width, 4, label, "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", 8)
	writePDFTextLines(pdf, x, y+5, width-2, 4, fitPDFTextLines(pdf, primary, width-2, 2), "L")
	if strings.TrimSpace(secondary) != "" {
		pdf.SetFont(pdfFontFamily, "", 6.5)
		writePDFTextLines(pdf, x, y+13, width-2, 3.5, fitPDFTextLines(pdf, secondary, width-2, 1), "L")
	}
}

func writeDeliveryNoteDates(pdf *fpdf.Fpdf, document DeliveryNoteDocument) {
	const y, width = 74.0, 30.0
	values := [][2]string{
		{"PO Number", document.PONumber},
		{"Order Date", formatPDFDate(document.OrderDate)},
		{"Expected Delivery", formatPDFDate(document.ExpectedDeliveryDate)},
		{"Issued Date", formatPDFDate(document.IssuedAt)},
	}
	for index, value := range values {
		x := 12 + float64(index)*32
		cellWidth := width
		pdf.SetXY(x, y)
		pdf.SetFont(pdfFontFamily, "", 6.5)
		pdf.CellFormat(cellWidth, 4, value[0], "", 0, "L", false, 0, "")
		pdf.SetXY(x, y+5)
		pdf.SetFont(pdfFontFamily, "B", 7.5)
		pdf.CellFormat(cellWidth, 5, value[1], "", 0, "L", false, 0, "")
		pdf.Line(x, y+11, x+cellWidth-2, y+11)
	}
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
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(136, 7, "Total Kanban", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(50, 7, formatPDFDecimal(totalKanban, 0), "", 1, "R", false, 0, "")
	pdf.Line(left, pdf.GetY(), left+186, pdf.GetY())
	pdf.SetX(left)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(136, 7, "Total Quantity", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(50, 7, strings.Join(deliveryNoteUnitTotals(lines), "  |  "), "", 1, "R", false, 0, "")
	pdf.Line(left, pdf.GetY(), left+186, pdf.GetY())

	pdf.Ln(3)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(186, 6, "REMARKS", "", 1, "L", false, 0, "")
	remarksY := pdf.GetY()
	pdf.Line(left, remarksY+12, left+186, remarksY+12)
	pdf.SetY(remarksY + 14)
	pdf.Ln(4)
	writeDeliveryNoteSignatureBoxes(pdf)
}

func writeDeliveryNoteTable(pdf *fpdf.Fpdf, widths []float64, headings []string, rows [][]string) {
	writeHeader := func() {
		left, _, _, _ := pdf.GetMargins()
		y := pdf.GetY()
		pdf.SetLineWidth(0.35)
		pdf.Line(left, y, left+186, y)
		pdf.SetFont(pdfFontFamily, "B", 6.2)
		for index, heading := range headings {
			pdf.CellFormat(widths[index], 8, heading, "", 0, "L", false, 0, "")
		}
		pdf.Ln(8)
		pdf.Line(left, pdf.GetY(), left+186, pdf.GetY())
		pdf.SetLineWidth(0.2)
	}
	writeHeader()
	for _, values := range rows {
		pdf.SetFont(pdfFontFamily, "", 6.7)
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
			align := "L"
			if index == 0 || index >= len(values)-4 {
				align = "R"
			}
			writePDFTextLines(pdf, x+1, y+1, widths[index]-2, 4, lines, align)
			x += widths[index]
		}
		left, _, _, _ := pdf.GetMargins()
		pdf.Line(left, y+rowHeight, left+186, y+rowHeight)
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
		pdf.SetFont(pdfFontFamily, "B", 8)
		pdf.CellFormat(90, 6, headings[index], "", 1, "C", false, 0, "")
		pdf.SetXY(x, y+6)
		pdf.SetFont(pdfFontFamily, "", 7)
		pdf.CellFormat(90, 5, subheadings[index], "", 1, "C", false, 0, "")
		pdf.Line(x+5, y+28, x+85, y+28)
		pdf.SetXY(x+2, y+28)
		pdf.CellFormat(50, 5, "Name:", "", 0, "L", false, 0, "")
		pdf.CellFormat(38, 5, "Date:", "", 0, "L", false, 0, "")
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
	pdf.SetDrawColor(24, 24, 27)
	pdf.SetTextColor(24, 24, 27)
	pdf.SetLineWidth(0.35)
	pdf.Rect(x, y, width, height, "D")

	const headerHeight = 15.0
	pdf.Line(x, y+headerHeight, x+width, y+headerHeight)
	pdf.SetXY(x+4, y+2)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(92, 4, pdfCompanyName(document.CompanyName), "", 0, "L", false, 0, "")
	pdf.SetXY(x+4, y+6)
	pdf.SetFont(pdfFontFamily, "B", 13)
	pdf.CellFormat(92, 7, "KANBAN CARD", "", 0, "L", false, 0, "")
	pdf.SetXY(x+98, y+2)
	pdf.SetFont(pdfFontFamily, "", 7)
	pdf.CellFormat(84, 4, "KANBAN ID", "", 0, "R", false, 0, "")
	pdf.SetXY(x+98, y+6)
	pdf.SetFont(pdfFontFamily, "B", 12)
	pdf.CellFormat(84, 7, label.KanbanID, "", 0, "R", false, 0, "")

	qrX, qrY, qrSize := x+4, y+19, 38.0
	pdf.Rect(qrX-1, qrY-1, qrSize+2, qrSize+2, "D")
	imageName := "kanban-qr-" + strconv.Itoa(index)
	pdf.RegisterImageOptionsReader(imageName, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, bytes.NewReader(qrPNG))
	pdf.ImageOptions(imageName, qrX, qrY, qrSize, qrSize, false, fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, 0, "")
	pdf.SetXY(qrX-1, qrY+qrSize+2)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(qrSize+2, 5, label.KanbanID, "", 0, "C", false, 0, "")

	detailX := x + 47
	detailWidth := width - 51
	partNumberWidth := 43.0
	partY := y + 18
	pdf.Rect(detailX, partY, partNumberWidth, 18, "D")
	pdf.Rect(detailX+partNumberWidth, partY, detailWidth-partNumberWidth, 18, "D")
	pdf.SetXY(detailX+2, partY+2)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(partNumberWidth-4, 4, "PART NUMBER", "", 0, "L", false, 0, "")
	pdf.SetXY(detailX+2, partY+7)
	pdf.SetFont(pdfFontFamily, "B", 11)
	pdf.CellFormat(partNumberWidth-4, 7, label.RawMaterialCode, "", 0, "L", false, 0, "")
	pdf.SetXY(detailX+partNumberWidth+2, partY+2)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(detailWidth-partNumberWidth-4, 4, "PART NAME", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", 11)
	partName := fitPDFTextLines(pdf, label.RawMaterialName, detailWidth-partNumberWidth-4, 2)
	writePDFTextLines(pdf, detailX+partNumberWidth+2, partY+7, detailWidth-partNumberWidth-4, 4.5, partName, "L")

	metaY := y + 36
	supplierWidth := 80.0
	pdf.Rect(detailX, metaY, supplierWidth, 14, "D")
	pdf.Rect(detailX+supplierWidth, metaY, detailWidth-supplierWidth, 14, "D")
	pdf.SetXY(detailX+2, metaY+1)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(supplierWidth-4, 4, "SUPPLIER", "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	supplier := fitPDFTextLines(pdf, document.SupplierName, supplierWidth-4, 1)
	writePDFTextLines(pdf, detailX+2, metaY+6, supplierWidth-4, 4, supplier, "L")
	pdf.SetXY(detailX+supplierWidth+2, metaY+1)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(detailWidth-supplierWidth-4, 4, "ORDER DATE", "", 0, "L", false, 0, "")
	pdf.SetXY(detailX+supplierWidth+2, metaY+6)
	pdf.SetFont(pdfFontFamily, "", 8)
	pdf.CellFormat(detailWidth-supplierWidth-4, 5, formatPDFDate(document.OrderDate), "", 0, "L", false, 0, "")

	infoY := y + 50
	columnWidth := detailWidth / 2
	pdf.Rect(detailX, infoY, columnWidth, 14, "D")
	pdf.Rect(detailX+columnWidth, infoY, columnWidth, 14, "D")
	pdf.SetXY(detailX+2, infoY+1)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(columnWidth-4, 4, "QUANTITY", "", 0, "L", false, 0, "")
	pdf.SetXY(detailX+2, infoY+5)
	pdf.SetFont(pdfFontFamily, "B", 12)
	pdf.CellFormat(columnWidth-4, 7, formatPDFDecimal(label.Quantity, 6)+" "+label.BaseUnitCode, "", 0, "L", false, 0, "")
	pdf.SetXY(detailX+columnWidth+2, infoY+1)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(columnWidth-4, 4, "CARD", "", 0, "L", false, 0, "")
	pdf.SetXY(detailX+columnWidth+2, infoY+5)
	pdf.SetFont(pdfFontFamily, "B", 12)
	pdf.CellFormat(columnWidth-4, 7, formatCardPosition(label), "", 0, "L", false, 0, "")

	referenceY := y + 64
	referenceWidth := detailWidth / 2
	pdf.Rect(detailX, referenceY, referenceWidth, 16, "D")
	pdf.Rect(detailX+referenceWidth, referenceY, referenceWidth, 16, "D")
	writeKanbanReference(pdf, detailX, referenceY, referenceWidth, "DELIVERY NOTE", document.DeliveryNoteNumber)
	writeKanbanReference(pdf, detailX+referenceWidth, referenceY, referenceWidth, "PURCHASE ORDER", document.PONumber)
	return pdf.Error()
}

func formatCardPosition(label KanbanLabel) string {
	return strconv.Itoa(label.CardNumber) + "/" + strconv.Itoa(label.CardTotal)
}

func writeKanbanReference(pdf *fpdf.Fpdf, x, y, width float64, label, value string) {
	pdf.SetXY(x+2, y+2)
	pdf.SetFont(pdfFontFamily, "B", 7)
	pdf.CellFormat(width-4, 4, label, "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	lines := fitPDFTextLines(pdf, value, width-4, 1)
	writePDFTextLines(pdf, x+2, y+7, width-4, 4, lines, "L")
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
	pdf.SetXY(x+2, y)
	pdf.CellFormat(keyWidth-4, lineHeight, key, "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 9)
	writePDFTextLines(pdf, x+keyWidth+2, y, valueWidth-4, lineHeight, lines, "L")
	pdf.SetXY(left, y+rowHeight)
}

func writePDFTable(pdf *fpdf.Fpdf, widths []float64, headings []string, rows [][]string) {
	writeHeader := func() {
		left, _, _, _ := pdf.GetMargins()
		y := pdf.GetY()
		pdf.SetLineWidth(0.35)
		pdf.Line(left, y, left+186, y)
		pdf.SetFont(pdfFontFamily, "B", 6.5)
		for index, heading := range headings {
			pdf.CellFormat(widths[index], 8, heading, "", 0, "L", false, 0, "")
		}
		pdf.Ln(8)
		pdf.Line(left, pdf.GetY(), left+186, pdf.GetY())
		pdf.SetLineWidth(0.2)
	}
	writeHeader()
	for _, values := range rows {
		pdf.SetFont(pdfFontFamily, "", 7)
		linesByCell := make([][]string, len(values))
		rowHeight := 8.0
		for index, value := range values {
			linesByCell[index] = fitPDFTextLines(pdf, value, widths[index]-2, 3)
			rowHeight = max(rowHeight, 3+float64(len(linesByCell[index]))*4)
		}
		if !pdfPageHasRoom(pdf, rowHeight) {
			pdf.AddPage()
			pdf.SetFont(pdfFontFamily, "B", 9)
			pdf.CellFormat(0, 7, "PURCHASE ORDER / MATERIALS CONTINUED", "", 1, "L", false, 0, "")
			pdf.Ln(2)
			writeHeader()
			pdf.SetFont(pdfFontFamily, "", 7)
		}
		x, y := pdf.GetX(), pdf.GetY()
		for index, lines := range linesByCell {
			align := "L"
			if index >= len(values)-3 {
				align = "R"
			}
			writePDFTextLines(pdf, x+1, y+2, widths[index]-2, 4, lines, align)
			x += widths[index]
		}
		left, _, _, _ := pdf.GetMargins()
		pdf.Line(left, y+rowHeight, left+186, y+rowHeight)
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
	CompanyName          string
	PONumber             string
	SupplierName         string
	PlantCode            string
	PlantName            string
	PlantAddress         string
	OrderDate            time.Time
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
		PurchaseOrderID: purchaseOrderID, PONumber: header.PONumber, CompanyName: header.CompanyName, SupplierName: header.SupplierName,
		PlantCode: header.PlantCode, PlantName: header.PlantName, PlantAddress: header.PlantAddress,
		OrderDate: header.OrderDate, ExpectedDeliveryDate: header.ExpectedDeliveryDate, IssuedAt: header.IssuedAt,
	}
	rows, err := query.Query(ctx, `SELECT pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,pol.base_unit_code_snapshot,
 pol.category_code_snapshot,pol.category_name_snapshot,pol.packing_code_snapshot,pol.packing_name_snapshot,
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
		if err := rows.Scan(&line.RawMaterialCode, &line.RawMaterialName, &line.BaseUnitCode,
			&line.CategoryCode, &line.CategoryName, &line.PackingCode, &line.PackingName,
			&line.QtyPerKanban, &line.TotalKanban, &line.TotalQuantity); err != nil {
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
		PurchaseOrderID: purchaseOrderID, PONumber: header.PONumber, CompanyName: header.CompanyName,
		SupplierName: header.SupplierName, PlantCode: header.PlantCode, PlantName: header.PlantName,
		PlantAddress: header.PlantAddress, OrderDate: header.OrderDate,
	}
	rows, err := query.Query(ctx, `SELECT kl.kanban_id,pol.raw_material_code_snapshot,pol.raw_material_name_snapshot,
 kl.quantity,pol.base_unit_code_snapshot,pol.category_code_snapshot,pol.category_name_snapshot,
 pol.packing_code_snapshot,pol.packing_name_snapshot,kl.lot_number,pol.total_kanban::integer
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
		if err := rows.Scan(&label.KanbanID, &label.RawMaterialCode, &label.RawMaterialName, &label.Quantity, &label.BaseUnitCode,
			&label.CategoryCode, &label.CategoryName, &label.PackingCode, &label.PackingName,
			&label.CardNumber, &label.CardTotal); err != nil {
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
	err := query.QueryRow(ctx, `SELECT COALESCE(NULLIF(BTRIM(ts.company_name),''),'Order Stock'),p.po_number,s.name,
 p.plant_code_snapshot,p.plant_name_snapshot,p.plant_address_snapshot,p.order_date,p.expected_delivery_date,p.status
 FROM purchase_orders p
 JOIN tenant_settings ts ON ts.tenant_id=p.tenant_id
 JOIN suppliers s ON s.tenant_id=p.tenant_id AND s.id=p.supplier_id
 WHERE p.tenant_id=$1 AND p.id=$2`, tenantID, purchaseOrderID).
		Scan(&header.CompanyName, &header.PONumber, &header.SupplierName, &header.PlantCode, &header.PlantName, &header.PlantAddress,
			&header.OrderDate, &header.ExpectedDeliveryDate, &header.Status)
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
