package inventory

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestFilterStockItemsKeepsZeroStockAndAppliesFilters(t *testing.T) {
	supplierA := uuid.New()
	supplierB := uuid.New()
	items := []StockItem{
		{ItemCode: "RM-001", RawMaterialName: "Bolt", SupplierID: supplierA, StockQuantity: decimal.Zero, StockStatus: "OUT_OF_STOCK"},
		{ItemCode: "RM-002", RawMaterialName: "Belt", SupplierID: supplierA, StockQuantity: decimal.NewFromInt(5), StockStatus: "LOW_STOCK"},
		{ItemCode: "RM-003", RawMaterialName: "Coil", SupplierID: supplierB, StockQuantity: decimal.NewFromInt(20), StockStatus: "IN_STOCK"},
	}

	all := filterStockItems(items, Filters{})
	if len(all) != 3 || !all[0].StockQuantity.IsZero() {
		t.Fatalf("unfiltered items = %#v, want zero-stock material retained", all)
	}

	filtered := filterStockItems(items, Filters{Search: "belt", SupplierID: &supplierA, Status: "LOW_STOCK"})
	if len(filtered) != 1 || filtered[0].ItemCode != "RM-002" {
		t.Fatalf("filtered items = %#v, want RM-002", filtered)
	}
}

func TestSummarizeStock(t *testing.T) {
	items := []StockItem{
		{AvailableKanban: 0, StockStatus: "OUT_OF_STOCK"},
		{AvailableKanban: 2, StockStatus: "LOW_STOCK"},
		{AvailableKanban: 3, StockStatus: "IN_STOCK"},
	}
	got := summarizeStock(items)
	if got.TotalRawMaterials != 3 || got.TotalInStockKanban != 5 || got.LowStockMaterials != 1 || got.OutOfStockMaterials != 1 {
		t.Fatalf("summary = %#v", got)
	}
}
