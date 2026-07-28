package emailing

import (
	"bytes"
	"strings"
	"testing"
)

func TestSecretBoxRoundTripAndRandomNonce(t *testing.T) {
	box, err := NewSecretBox("test-key")
	if err != nil {
		t.Fatal(err)
	}
	first, err := box.Encrypt("smtp-secret")
	if err != nil {
		t.Fatal(err)
	}
	second, err := box.Encrypt("smtp-secret")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("ciphertexts must use independent nonces")
	}
	plain, err := box.Decrypt(first)
	if err != nil || plain != "smtp-secret" {
		t.Fatalf("decrypt = %q, %v", plain, err)
	}
}

func TestApprovalTemplateContainsActionsAndDetailLink(t *testing.T) {
	data := ApprovalMailData{PONumber: "PO-1", SupplierName: "Supplier <One>", ApproverName: "Director", CreatedByName: "Buyer", Currency: "IDR", TotalAmount: "1000"}
	rendered := approvalHTML(data, "https://app/approve", "https://app/reject", "https://app/supplier-orders/1")
	for _, expected := range []string{"APPROVE", "REJECT", "https://app/supplier-orders/1", "Supplier &lt;One&gt;"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("template missing %q", expected)
		}
	}
}

func TestApprovalTemplateFormatsBusinessNumbers(t *testing.T) {
	data := ApprovalMailData{
		PONumber: "PO-1", SupplierName: "Supplier", ApproverName: "Director",
		CreatedByName: "Buyer", Currency: "IDR", TotalAmount: "2000000.000000",
		Lines: []ApprovalMailLine{
			{Code: "RM-1", Name: "First", Unit: "PC", QtyPerKanban: "5.000000", TotalKanban: 10, TotalQuantity: "50.000000"},
			{Code: "RM-2", Name: "Second", Unit: "BOX", QtyPerKanban: "5.250000", TotalKanban: 2, TotalQuantity: "10.500000"},
		},
	}

	rendered := approvalHTML(data, "https://app/approve", "https://app/reject", "https://app/detail")

	for _, expected := range []string{"IDR 2.000.000", ">5 PC<", ">5,25 BOX<", ">50 PC<", ">10,5 BOX<"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("template missing formatted value %q", expected)
		}
	}
	if strings.Contains(rendered, ".000000") {
		t.Fatal("template contains database scale padding")
	}
}
