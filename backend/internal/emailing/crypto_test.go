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
