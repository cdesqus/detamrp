package emailing

import (
	"fmt"
	"html"
	"strings"
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

func approvalHTML(data ApprovalMailData, approveURL, rejectURL, detailURL string) string {
	var rows strings.Builder
	for _, line := range data.Lines {
		fmt.Fprintf(&rows, `<tr><td><b>%s</b><br><span>%s</span></td><td>%s %s</td><td>%d</td><td>%s %s</td></tr>`, html.EscapeString(line.Code), html.EscapeString(line.Name), html.EscapeString(formatEmailNumber(line.QtyPerKanban)), html.EscapeString(line.Unit), line.TotalKanban, html.EscapeString(formatEmailNumber(line.TotalQuantity)), html.EscapeString(line.Unit))
	}
	return emailShell("Purchase Order Approval", fmt.Sprintf(`<p style="margin:0 0 18px;color:#52525b">Hello %s, a purchase order is waiting for your decision.</p>
<div style="border:1px solid #e4e4e7;border-radius:10px;padding:18px;margin-bottom:18px"><div style="font-size:22px;font-weight:700">%s</div><div style="color:#71717a;margin-top:4px">%s</div><table style="width:100%%;margin-top:16px;font-size:13px"><tr><td>Order Date</td><td><b>%s</b></td><td>Expected Delivery</td><td><b>%s</b></td></tr><tr><td>Created By</td><td><b>%s</b></td><td>Total</td><td><b>%s %s</b></td></tr></table></div>
<table style="width:100%%;border-collapse:collapse;font-size:12px;margin-bottom:20px"><thead><tr><th align="left">Material</th><th align="left">Qty/Kanban</th><th align="left">Kanban</th><th align="left">Total Qty</th></tr></thead><tbody>%s</tbody></table>
<div style="text-align:center;margin:24px 0"><a href="%s" style="display:inline-block;background:#18181b;color:white;padding:12px 24px;border-radius:7px;text-decoration:none;font-weight:700">APPROVE</a> <a href="%s" style="display:inline-block;border:1px solid #d4d4d8;color:#991b1b;padding:11px 24px;border-radius:7px;text-decoration:none;font-weight:700">REJECT</a></div>
<div style="text-align:center"><a href="%s" style="color:#3f3f46">View PO Detail in DETA MRP</a></div>`,
		html.EscapeString(data.ApproverName), html.EscapeString(data.PONumber), html.EscapeString(data.SupplierName), data.OrderDate.Format("02 Jan 2006"), data.ExpectedDeliveryDate.Format("02 Jan 2006"), html.EscapeString(data.CreatedByName), html.EscapeString(data.Currency), html.EscapeString(formatEmailNumber(data.TotalAmount)), rows.String(), approveURL, rejectURL, detailURL))
}
func supplierHTML(poNumber, supplier, dnNumber, expected string, totalMaterials int, totalKanban int64) string {
	return emailShell("New Purchase Order", fmt.Sprintf(`<p style="color:#52525b">Dear %s,</p><p>Please prepare the materials according to the attached Purchase Order and Delivery Note.</p>
<div style="border:1px solid #e4e4e7;border-radius:10px;padding:18px;margin:18px 0"><div style="font-size:22px;font-weight:700">%s</div><table style="width:100%%;margin-top:14px;font-size:13px"><tr><td>Delivery Note</td><td><b>%s</b></td></tr><tr><td>Expected Delivery</td><td><b>%s</b></td></tr><tr><td>Total Materials</td><td><b>%d</b></td></tr><tr><td>Total Kanban</td><td><b>%d</b></td></tr></table></div>
<div style="background:#fffbeb;border:1px solid #fde68a;border-radius:9px;padding:16px"><b>IMPORTANT SHIPPING INSTRUCTION</b><ol style="padding-left:20px;line-height:1.7"><li>Print the attached Delivery Note.</li><li>Print and attach one Kanban Label to each physical lot.</li><li>The DN attached to the shipment must exactly match <b>%s</b>.</li><li>Shipments without the issued DN and Kanban labels may be rejected.</li></ol></div>
<p style="margin-top:20px"><b>Attached:</b> Purchase Order.pdf, Delivery Note.pdf, Kanban Labels.pdf</p>`, html.EscapeString(supplier), html.EscapeString(poNumber), html.EscapeString(dnNumber), html.EscapeString(expected), totalMaterials, totalKanban, html.EscapeString(dnNumber)))
}
func emailShell(title, body string) string {
	return fmt.Sprintf(`<!doctype html><html><body style="margin:0;background:#f4f4f5;font-family:Arial,sans-serif;color:#18181b"><div style="max-width:680px;margin:0 auto;padding:24px"><div style="background:#18181b;color:white;border-radius:12px 12px 0 0;padding:20px 24px"><b style="font-size:18px">DETA MRP</b><div style="font-size:12px;color:#d4d4d8;margin-top:4px">Logistics &amp; Production Control</div></div><div style="background:white;border:1px solid #e4e4e7;border-top:0;padding:28px 24px"><h1 style="font-size:22px;margin:0 0 22px">%s</h1>%s</div><div style="padding:14px;text-align:center;color:#71717a;font-size:11px">This email was generated automatically by DETA MRP.</div></div></body></html>`, title, body)
}
