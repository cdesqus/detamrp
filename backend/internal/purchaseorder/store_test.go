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

func TestApprovalSelectIncludesPurchaseOrderAndSupplierDetails(t *testing.T) {
	query := strings.ToLower(approvalSelect)
	for _, fragment := range []string{
		"p.po_number", "p.supplier_id", "s.name",
		"join purchase_orders p on p.tenant_id=a.tenant_id and p.id=a.purchase_order_id",
		"join suppliers s on s.tenant_id=p.tenant_id and s.id=p.supplier_id",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("approval query missing %q", fragment)
		}
	}
}
