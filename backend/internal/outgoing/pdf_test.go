package outgoing

import (
	"bytes"
	"testing"
)

func TestRenderPDF(t *testing.T) {
	b, e := RenderPDF(Document{DocumentNumber: "OUT-1", Destination: "", KanbanCount: 2})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatal("not pdf")
	}
}
