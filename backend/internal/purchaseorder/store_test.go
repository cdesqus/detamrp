package purchaseorder

import (
	"strings"
	"testing"
)

func TestGetOrderHeaderQueryLocksHeaderBeforeLoadingLines(t *testing.T) {
	if !strings.Contains(strings.ToLower(getOrderHeaderQuery), "for share") {
		t.Fatal("GetOrder header query must hold a shared row lock while loading lines")
	}
}
