package emailing

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"
)

func testLogoPNG(t *testing.T) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func TestBrandedEmailShellUsesCompanyIdentityAndCIDLogo(t *testing.T) {
	logo := testLogoPNG(t)
	rendered := emailShellWithBranding(CompanyBranding{CompanyName: `PT Buyer <Indonesia>`, Logo: logo, LogoMIME: "image/png"}, "Approval", "<p>Body</p>")
	for _, expected := range []string{"PT Buyer &lt;Indonesia&gt;", "DETA MRP", "cid:company-logo", `role="presentation"`, `@media only screen`} {
		if !strings.Contains(rendered.HTML, expected) {
			t.Fatalf("branded shell missing %q", expected)
		}
	}
	if len(rendered.Inline) != 1 || rendered.Inline[0].ContentID != "company-logo" || !rendered.Inline[0].Inline || !bytes.Equal(rendered.Inline[0].Content, logo) {
		t.Fatalf("inline logo = %#v", rendered.Inline)
	}
}

func TestBrandedEmailShellFallsBackToTextForInvalidLogo(t *testing.T) {
	rendered := emailShellWithBranding(CompanyBranding{CompanyName: "PT Buyer", Logo: []byte("corrupt"), LogoMIME: "image/png"}, "Test", "<p>Body</p>")
	if strings.Contains(rendered.HTML, "cid:company-logo") || len(rendered.Inline) != 0 {
		t.Fatal("invalid logo was embedded")
	}
	if !strings.Contains(rendered.HTML, "PT Buyer") {
		t.Fatal("text identity fallback is missing")
	}
}

func TestDecisionResultEmailExplainsRejectionActorTimeAndReason(t *testing.T) {
	data := DecisionMailData{
		PONumber: "PO-REJECT", SupplierName: "PT Supplier", PlantCode: "PLT-01", PlantName: "Cikarang",
		RecipientName: "Buyer", Status: "REJECTED", DecisionActor: "Director", DecisionAt: time.Date(2026, 7, 30, 9, 15, 0, 0, time.UTC),
		Reason: `Price > budget`,
	}
	rendered := decisionEmail(data, CompanyBranding{CompanyName: "PT Buyer"}).HTML
	for _, expected := range []string{"PT Buyer", "PO-REJECT", "PT Supplier", "PLT-01", "Cikarang", "REJECTED", "Director", "30 Jul 2026 09:15 UTC", "Price &gt; budget"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("decision email missing %q", expected)
		}
	}
}

func TestMIMEMessageEncodesInlineLogoForCIDRendering(t *testing.T) {
	logo := []byte("logo")
	message := Message{To: "to@example.com", Subject: "Brand", HTML: `<img src="cid:company-logo">`, Attachments: []Attachment{{
		Filename: "company-logo.png", ContentType: "image/png", Content: logo, Inline: true, ContentID: "company-logo",
	}}}
	payload, err := mimeMessage(SMTPSettings{FromName: "DETA MRP", FromEmail: "no-reply@example.com"}, message)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"multipart/related", `Content-Disposition: inline; filename="company-logo.png"`, "Content-Id: <company-logo>", base64.StdEncoding.EncodeToString(logo)} {
		if !bytes.Contains(payload, []byte(expected)) {
			t.Fatalf("MIME payload missing %q", expected)
		}
	}
}

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

func TestApprovalEmailShowsPlantReferencesAndDetailedMaterialSnapshots(t *testing.T) {
	data := ApprovalMailData{
		PONumber: "PO-202607-0009", SupplierName: "PT Supplier", ApproverName: "Director", CreatedByName: "Buyer",
		PlantCode: "PLT-01", PlantName: "Cikarang Plant", PlantAddress: "Kawasan Industri",
		Status: "SUBMITTED", Currency: "IDR", TotalAmount: "750000.000000", Notes: "Deliver before noon",
		Lines: []ApprovalMailLine{{Code: "RM-01", Name: "Steel Bracket", CategoryCode: "METAL", CategoryName: "Metal",
			PackingCode: "BOX", PackingName: "Carton Box", Unit: "PC", QtyPerKanban: "5.000000", TotalKanban: 3,
			TotalQuantity: "15.000000", UnitPrice: "50000.000000", LineTotal: "750000.000000"}},
	}
	rendered := approvalEmail(data, CompanyBranding{CompanyName: "PT Buyer"}, "https://app/approve", "https://app/reject", "https://app/detail").HTML
	for _, expected := range []string{"PT Buyer", "PLT-01", "Cikarang Plant", "Kawasan Industri", "PENDING APPROVAL", "METAL", "Metal", "BOX", "Carton Box", "IDR 50.000", "IDR 750.000", "Deliver before noon"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("detailed approval email missing %q", expected)
		}
	}
}
