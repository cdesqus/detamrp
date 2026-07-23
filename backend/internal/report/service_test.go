package report

import (
	"bytes"
	"net/url"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestParseFilterRejectsReversedDates(t *testing.T) {
	_, fields := ParseFilter(url.Values{"fromDate": {"2026-07-24"}, "toDate": {"2026-07-23"}})
	if fields["toDate"] == "" {
		t.Fatal("reversed range accepted")
	}
}

func TestRenderReceivingPDF(t *testing.T) {
	result := Result{Items: []Row{{
		ReceivingNumber: "RCV-1", ReceivingDate: time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC),
		DeliveryNoteNumber: "DN-1", PONumber: "PO-1", SupplierName: "Supplier A",
		RawMaterialCode: "RM-1", RawMaterialName: "Material A", BaseUnitCode: "PC",
		KanbanReceived: 2, ReceivedQuantity: decimal.NewFromInt(10),
	}}}
	result.Totals = summarize(result.Items)
	pdf, err := RenderReceivingPDF(result, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) || !bytes.Contains(pdf, []byte("Receiving Report")) {
		t.Fatal("receiving report PDF content missing")
	}
}
