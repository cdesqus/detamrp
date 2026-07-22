package receiving

import (
	"bytes"
	"testing"
)

func TestRenderReceivingPDF(t *testing.T) {
	b, err := RenderPDF(Receiving{ReceivingNumber: "RCV-1", DeliveryNoteNumber: "DN-1", PONumber: "PO-1", SupplierName: "Pemasok", Planned: 10, PreviouslyReceived: 2, ReceivedNow: 3, Outstanding: 5})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatal("not pdf")
	}
}
