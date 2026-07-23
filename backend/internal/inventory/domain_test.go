package inventory

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestStockStatus(t *testing.T) {
	tests := []struct {
		name     string
		quantity string
		minimum  string
		want     string
	}{
		{"zero stock", "0", "0", "OUT_OF_STOCK"},
		{"below minimum", "5", "10", "LOW_STOCK"},
		{"equal to minimum", "10", "10", "LOW_STOCK"},
		{"above minimum", "11", "10", "IN_STOCK"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StockStatus(decimal.RequireFromString(tt.quantity), decimal.RequireFromString(tt.minimum))
			if got != tt.want {
				t.Fatalf("StockStatus(%s, %s) = %q, want %q", tt.quantity, tt.minimum, got, tt.want)
			}
		})
	}
}
