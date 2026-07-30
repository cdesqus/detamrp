package emailing

import (
	"bytes"
	"fmt"
	"html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	_ "golang.org/x/image/webp"
	"order-stock/backend/internal/purchaseorder"
)

func formatEmailNumber(value string) string {
	value = strings.TrimSpace(value)
	sign := ""
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		sign, value = value[:1], value[1:]
	}
	parts := strings.SplitN(value, ".", 2)
	integer := strings.TrimLeft(parts[0], "0")
	if integer == "" {
		integer = "0"
	}
	var grouped strings.Builder
	for index, digit := range integer {
		if index > 0 && (len(integer)-index)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(digit)
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = strings.TrimRight(parts[1], "0")
	}
	if fraction == "" {
		return sign + grouped.String()
	}
	return sign + grouped.String() + "," + fraction
}

func referenceName(code, name string) string {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	if code == "" {
		return name
	}
	if name == "" || code == name {
		return code
	}
	return code + " · " + name
}

func summaryRow(label, value string) string {
	return `<tr><td class="stack-cell" style="width:34%;padding:7px 10px;color:#71717a;border-bottom:1px solid #f4f4f5">` + html.EscapeString(label) + `</td><td class="stack-cell" style="padding:7px 10px;font-weight:600;border-bottom:1px solid #f4f4f5">` + html.EscapeString(value) + `</td></tr>`
}

func sectionTitle(value string) string {
	return `<div style="margin:24px 0 9px;font-size:11px;font-weight:700;letter-spacing:.1em;color:#52525b">` + html.EscapeString(strings.ToUpper(value)) + `</div>`
}

func displayEmailStatus(value string) string {
	value = strings.TrimSpace(value)
	if value == "SUBMITTED" {
		return "PENDING APPROVAL"
	}
	return strings.ReplaceAll(value, "_", " ")
}

func approvalEmail(data ApprovalMailData, branding CompanyBranding, approveURL, rejectURL, detailURL string) renderedEmail {
	var rows strings.Builder
	for _, line := range data.Lines {
		fmt.Fprintf(&rows, `<tr><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7"><b>%s</b><br><span style="color:#71717a">%s</span></td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7">%s<br><span style="color:#71717a">%s</span></td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7">%s %s</td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7;text-align:center">%d</td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7">%s %s</td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7;text-align:right">%s %s<br><span style="color:#71717a">%s %s</span></td></tr>`,
			html.EscapeString(line.Code), html.EscapeString(line.Name),
			html.EscapeString(referenceName(line.CategoryCode, line.CategoryName)), html.EscapeString(referenceName(line.PackingCode, line.PackingName)),
			html.EscapeString(formatEmailNumber(line.QtyPerKanban)), html.EscapeString(line.Unit), line.TotalKanban,
			html.EscapeString(formatEmailNumber(line.TotalQuantity)), html.EscapeString(line.Unit),
			html.EscapeString(data.Currency), html.EscapeString(formatEmailNumber(line.UnitPrice)),
			html.EscapeString(data.Currency), html.EscapeString(formatEmailNumber(line.LineTotal)))
	}
	plant := referenceName(data.PlantCode, data.PlantName)
	if strings.TrimSpace(data.PlantAddress) != "" {
		plant += " — " + data.PlantAddress
	}
	body := `<p style="margin:0 0 18px;color:#52525b;line-height:1.6">Hello ` + html.EscapeString(data.ApproverName) + `, please review the purchase order below.</p>` +
		`<div style="border:1px solid #e4e4e7;border-radius:10px;overflow:hidden"><div style="padding:16px 18px;background:#fafafa"><div style="font-size:22px;font-weight:700">` + html.EscapeString(data.PONumber) + `</div><div style="color:#71717a;margin-top:4px">` + html.EscapeString(data.SupplierName) + `</div></div><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;font-size:13px">` +
		summaryRow("Status", displayEmailStatus(data.Status)) + summaryRow("Destination Plant", plant) + summaryRow("Order Date", data.OrderDate.Format("02 Jan 2006")) +
		summaryRow("Expected Delivery", data.ExpectedDeliveryDate.Format("02 Jan 2006")) + summaryRow("Created By", data.CreatedByName) +
		summaryRow("Total Amount", data.Currency+" "+formatEmailNumber(data.TotalAmount)) + `</table></div>` +
		sectionTitle("Material Details") + `<div style="overflow-x:auto"><table width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;font-size:12px;min-width:620px"><thead><tr style="background:#f4f4f5"><th align="left" style="padding:9px 8px">Part</th><th align="left">Category / Packing</th><th align="left">Qty/Card</th><th>Cards</th><th align="left">Total Qty</th><th align="right" style="padding-right:8px">Price / Amount</th></tr></thead><tbody>` + rows.String() + `</tbody></table></div>`
	if strings.TrimSpace(data.Notes) != "" {
		body += sectionTitle("Notes") + `<div style="padding:13px 15px;background:#fafafa;border-left:3px solid #a1a1aa;line-height:1.55">` + html.EscapeString(data.Notes) + `</div>`
	}
	body += `<div style="text-align:center;margin:26px 0 18px"><a href="` + html.EscapeString(approveURL) + `" style="display:inline-block;background:#166534;color:white;padding:12px 24px;border-radius:7px;text-decoration:none;font-weight:700;margin:4px">APPROVE</a><a href="` + html.EscapeString(rejectURL) + `" style="display:inline-block;border:1px solid #dc2626;color:#991b1b;padding:11px 24px;border-radius:7px;text-decoration:none;font-weight:700;margin:4px">REJECT</a></div><div style="text-align:center"><a href="` + html.EscapeString(detailURL) + `" style="color:#3f3f46">View PO detail in DETA MRP</a></div>`
	return emailShellWithBranding(branding, "Purchase Order Approval", body)
}

func approvalHTML(data ApprovalMailData, approveURL, rejectURL, detailURL string) string {
	return approvalEmail(data, CompanyBranding{CompanyName: data.CompanyName}, approveURL, rejectURL, detailURL).HTML
}

func supplierEmailHTML(order purchaseorder.Order, branding CompanyBranding) renderedEmail {
	var rows strings.Builder
	for _, line := range order.Lines {
		fmt.Fprintf(&rows, `<tr><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7"><b>%s</b><br><span style="color:#71717a">%s</span></td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7">%s<br><span style="color:#71717a">%s</span></td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7">%s %s</td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7;text-align:center">%s</td><td style="padding:10px 8px;border-bottom:1px solid #e4e4e7">%s %s</td></tr>`,
			html.EscapeString(line.RawMaterialCode), html.EscapeString(line.RawMaterialName),
			html.EscapeString(referenceName(line.CategoryCode, line.CategoryName)), html.EscapeString(referenceName(line.PackingCode, line.PackingName)),
			html.EscapeString(formatEmailNumber(line.QtyPerKanbanSnapshot.String())), html.EscapeString(line.BaseUnitCode),
			html.EscapeString(formatEmailNumber(line.TotalKanban.String())),
			html.EscapeString(formatEmailNumber(line.OrderedBaseQty.String())), html.EscapeString(line.BaseUnitCode))
	}
	plant := referenceName(order.PlantCode, order.PlantName)
	if strings.TrimSpace(order.PlantAddress) != "" {
		plant += " — " + order.PlantAddress
	}
	dn := ""
	if order.Documents != nil {
		dn = order.Documents.DeliveryNoteNumber
	}
	body := `<p style="color:#52525b;line-height:1.6">Dear ` + html.EscapeString(order.SupplierName) + `, please prepare and ship the materials according to the issued documents.</p><div style="border:1px solid #e4e4e7;border-radius:10px;overflow:hidden"><div style="padding:16px 18px;background:#fafafa"><div style="font-size:22px;font-weight:700">` + html.EscapeString(order.PONumber) + `</div><div style="color:#71717a;margin-top:4px">Delivery Note ` + html.EscapeString(dn) + `</div></div><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;font-size:13px">` +
		summaryRow("Destination Plant", plant) + summaryRow("Order Date", order.OrderDate.Format("02 Jan 2006")) + summaryRow("Expected Delivery", order.ExpectedDeliveryDate.Format("02 Jan 2006")) + summaryRow("Total Amount", order.Currency+" "+formatEmailNumber(order.TotalAmount.String())) + `</table></div>` +
		sectionTitle("Material Preparation") + `<div style="overflow-x:auto"><table width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;font-size:12px;min-width:560px"><thead><tr style="background:#f4f4f5"><th align="left" style="padding:9px 8px">Part</th><th align="left">Category / Packing</th><th align="left">Qty/Card</th><th>Cards</th><th align="left">Total Qty</th></tr></thead><tbody>` + rows.String() + `</tbody></table></div>` +
		sectionTitle("Printing & Shipping Instructions") + `<div style="background:#fffbeb;border:1px solid #fde68a;border-radius:9px;padding:16px"><ol style="margin:0;padding-left:20px;line-height:1.75"><li>Print the attached Delivery Note and include it with the shipment.</li><li>Print and attach one Kanban Label to each physical card/lot.</li><li>Send every material to the destination Plant shown above.</li><li>The document number on the shipment must match <b>` + html.EscapeString(dn) + `</b>.</li></ol></div><p style="margin:20px 0 0;color:#52525b"><b>Attachments:</b> Purchase Order PDF, Delivery Note PDF, and Kanban Labels PDF.</p>`
	return emailShellWithBranding(branding, "Purchase Order & Delivery Documents", body)
}

func decisionEmail(data DecisionMailData, branding CompanyBranding) renderedEmail {
	plant := referenceName(data.PlantCode, data.PlantName)
	if strings.TrimSpace(data.PlantAddress) != "" {
		plant += " — " + data.PlantAddress
	}
	accent, panel, title := "#166534", "#f0fdf4", "Purchase Order Approved"
	if data.Status == "REJECTED" {
		accent, panel, title = "#991b1b", "#fef2f2", "Purchase Order Rejected"
	}
	body := `<p style="margin:0 0 18px;color:#52525b;line-height:1.6">Hello ` + html.EscapeString(data.RecipientName) + `, the approval decision below has been recorded.</p><div style="border:1px solid #e4e4e7;border-radius:10px;overflow:hidden"><div style="padding:16px 18px;background:` + panel + `;border-left:4px solid ` + accent + `"><div style="font-size:22px;font-weight:700">` + html.EscapeString(data.PONumber) + `</div><div style="color:` + accent + `;font-weight:700;margin-top:5px">` + html.EscapeString(data.Status) + `</div></div><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;font-size:13px">` +
		summaryRow("Supplier", data.SupplierName) + summaryRow("Destination Plant", plant) + summaryRow("Decision By", data.DecisionActor) + summaryRow("Decision Time", data.DecisionAt.UTC().Format("02 Jan 2006 15:04 MST")) + `</table></div>`
	if strings.TrimSpace(data.Reason) != "" {
		body += sectionTitle("Rejection Reason") + `<div style="padding:14px 16px;background:#fafafa;border-left:3px solid #dc2626;line-height:1.55">` + html.EscapeString(data.Reason) + `</div>`
	}
	return emailShellWithBranding(branding, title, body)
}

func emailShellWithBranding(branding CompanyBranding, title, body string) renderedEmail {
	company := strings.TrimSpace(branding.CompanyName)
	if company == "" {
		company = "DETA MRP"
	}
	identity := `<div style="font-size:20px;font-weight:700;line-height:1.25">` + html.EscapeString(company) + `</div>`
	var inline []Attachment
	if validEmailLogo(branding.Logo, branding.LogoMIME) {
		identity = `<img src="cid:company-logo" alt="` + html.EscapeString(company) + `" style="display:block;max-width:180px;max-height:54px;width:auto;height:auto;margin:0 0 10px">` + identity
		extension := strings.TrimPrefix(branding.LogoMIME, "image/")
		if extension == "jpeg" {
			extension = "jpg"
		}
		inline = []Attachment{{Filename: "company-logo." + extension, ContentType: branding.LogoMIME, Content: branding.Logo, Inline: true, ContentID: "company-logo"}}
	}
	return renderedEmail{HTML: fmt.Sprintf(`<!doctype html><html><head><meta name="viewport" content="width=device-width"><style>@media only screen and (max-width:620px){.email-wrap{padding:12px!important}.email-card{padding:22px 16px!important}.stack-cell{display:block!important;width:100%%!important;padding:5px 0!important}}</style></head><body style="margin:0;background:#f4f4f5;font-family:Arial,sans-serif;color:#18181b"><div class="email-wrap" style="max-width:680px;margin:0 auto;padding:24px"><table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:separate"><tr><td style="background:#fff;border:1px solid #e4e4e7;border-bottom:0;border-radius:12px 12px 0 0;padding:22px 24px">%s<div style="font-size:11px;color:#71717a;letter-spacing:.12em;margin-top:5px">DETA MRP · LOGISTICS &amp; PRODUCTION CONTROL</div></td></tr><tr><td class="email-card" style="background:#fff;border:1px solid #e4e4e7;padding:30px 24px"><h1 style="font-size:23px;line-height:1.3;margin:0 0 22px">%s</h1>%s</td></tr><tr><td style="padding:16px;text-align:center;color:#71717a;font-size:11px">This email was generated automatically by DETA MRP. Please do not reply.</td></tr></table></div></body></html>`, identity, html.EscapeString(title), body), Inline: inline}
}

func validEmailLogo(content []byte, mime string) bool {
	if len(content) == 0 {
		return false
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return false
	}
	return format == "png" && mime == "image/png" || format == "jpeg" && mime == "image/jpeg" || format == "webp" && mime == "image/webp"
}
