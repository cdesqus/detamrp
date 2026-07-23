package report

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestSummarizeReceivingRows(t *testing.T) {
	rows := []Row{
		{KanbanReceived: 2, ReceivedQuantity: decimal.NewFromInt(10)},
		{KanbanReceived: 3, ReceivedQuantity: decimal.RequireFromString("7.5")},
	}
	got := summarize(rows)
	if got.KanbanReceived != 5 {
		t.Fatalf("KanbanReceived = %d", got.KanbanReceived)
	}
	if !got.ReceivedQuantity.Equal(decimal.RequireFromString("17.5")) {
		t.Fatalf("ReceivedQuantity = %s", got.ReceivedQuantity)
	}
}
