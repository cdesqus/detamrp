package emailing

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"order-stock/backend/internal/purchaseorder"
)

type recordingEmailStore struct {
	supplierEmail string
	finishedWith  error
	branding      CompanyBranding
	approvalData  ApprovalMailData
	decisionData  DecisionMailData
}

func (s *recordingEmailStore) GetSMTP(context.Context, uuid.UUID) (storedSMTP, error) {
	return storedSMTP{SMTPSettings: SMTPSettings{Host: "smtp.example.com", Port: 465, Security: "TLS", FromName: "Order Stock", FromEmail: "noreply@example.com"}}, nil
}
func (s *recordingEmailStore) SaveSMTP(context.Context, Actor, SMTPSettingsInput, []byte, bool) error {
	return nil
}
func (s *recordingEmailStore) CreateLog(context.Context, Actor, string, string, uuid.UUID, string, string, string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *recordingEmailStore) FinishLog(_ context.Context, _ uuid.UUID, _ uuid.UUID, sendErr error) error {
	s.finishedWith = sendErr
	return nil
}
func (s *recordingEmailStore) ListLogs(context.Context, Actor, string) ([]EmailLog, error) {
	return nil, nil
}
func (s *recordingEmailStore) ApprovalData(context.Context, Actor, uuid.UUID) (ApprovalMailData, error) {
	return s.approvalData, nil
}
func (s *recordingEmailStore) Branding(context.Context, uuid.UUID) (CompanyBranding, error) {
	return s.branding, nil
}
func (s *recordingEmailStore) DecisionData(context.Context, uuid.UUID, uuid.UUID) (DecisionMailData, error) {
	return s.decisionData, nil
}
func (s *recordingEmailStore) SaveApprovalToken(context.Context, uuid.UUID, uuid.UUID, []byte, time.Time) error {
	return nil
}
func (s *recordingEmailStore) ResolveToken(context.Context, uuid.UUID, []byte) (TokenContext, error) {
	return TokenContext{}, nil
}
func (s *recordingEmailStore) MarkTokenUsed(context.Context, uuid.UUID, []byte) error {
	return nil
}
func (s *recordingEmailStore) SupplierEmail(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return s.supplierEmail, nil
}

type recordingTransport struct {
	message Message
	err     error
}

func (t *recordingTransport) Send(_ SMTPSettings, _ string, message Message) error {
	t.message = message
	return t.err
}

func TestSupplierMessageIncludesRecipientReferencesAndThreePDFs(t *testing.T) {
	order := purchaseorder.Order{
		ID: uuid.New(), PONumber: "PO-202607-00001", SupplierName: "PT Supplier",
		ExpectedDeliveryDate: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
		Documents:            &purchaseorder.DocumentSummary{DeliveryNoteNumber: "DN-202607-00001"},
		Lines: []purchaseorder.OrderLine{
			{TotalKanban: decimal.NewFromInt(2)},
			{TotalKanban: decimal.NewFromInt(3)},
		},
	}
	documents := []purchaseorder.PDFDocument{
		{Filename: "Purchase Order PO-202607-00001.pdf", Content: []byte("%PDF-po")},
		{Filename: "Delivery Note DN-202607-00001.pdf", Content: []byte("%PDF-dn")},
		{Filename: "Kanban Labels PO-202607-00001.pdf", Content: []byte("%PDF-labels")},
	}

	message, err := supplierMessage(order, "supplier@example.com", documents[0], documents[1], documents[2])
	if err != nil {
		t.Fatal(err)
	}
	if message.To != "supplier@example.com" {
		t.Fatalf("recipient = %q", message.To)
	}
	for _, reference := range []string{"PO-202607-00001", "DN-202607-00001"} {
		if !bytes.Contains([]byte(message.HTML+message.Subject), []byte(reference)) {
			t.Fatalf("message missing reference %q", reference)
		}
	}
	if len(message.Attachments) != 3 {
		t.Fatalf("attachments = %d", len(message.Attachments))
	}
	for index, attachment := range message.Attachments {
		if attachment.Filename != documents[index].Filename {
			t.Fatalf("attachment %d filename = %q", index, attachment.Filename)
		}
		if attachment.ContentType != "application/pdf" {
			t.Fatalf("attachment %d content type = %q", index, attachment.ContentType)
		}
		if len(attachment.Content) == 0 {
			t.Fatalf("attachment %d is empty", index)
		}
	}
	payload, err := mimeMessage(SMTPSettings{FromName: "Order Stock", FromEmail: "noreply@example.com"}, message)
	if err != nil {
		t.Fatal(err)
	}
	for _, document := range documents {
		if !bytes.Contains(payload, []byte(document.Filename)) {
			t.Fatalf("MIME payload missing %q", document.Filename)
		}
	}
}

func TestSupplierEmailShowsDestinationAndMaterialPreparationDetails(t *testing.T) {
	order := purchaseorder.Order{
		PONumber: "PO-DETAIL", CompanyName: "PT Buyer", SupplierName: "PT Supplier", PlantCode: "PLT-02", PlantName: "Karawang", PlantAddress: "Industrial Estate",
		OrderDate: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC), ExpectedDeliveryDate: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
		Currency: "IDR", TotalAmount: decimal.NewFromInt(150000), Documents: &purchaseorder.DocumentSummary{DeliveryNoteNumber: "DN-DETAIL"},
		Lines: []purchaseorder.OrderLine{{RawMaterialCode: "RM-1", RawMaterialName: "Bracket", CategoryCode: "METAL", CategoryName: "Metal", PackingCode: "BOX", PackingName: "Carton", BaseUnitCode: "PC", QtyPerKanbanSnapshot: decimal.NewFromInt(5), TotalKanban: decimal.NewFromInt(3), OrderedBaseQty: decimal.NewFromInt(15)}},
	}
	rendered := supplierEmailHTML(order, CompanyBranding{CompanyName: "PT Buyer"}).HTML
	for _, expected := range []string{"PT Buyer", "PLT-02", "Karawang", "Industrial Estate", "RM-1", "Bracket", "METAL", "Metal", "BOX", "Carton", "5 PC", "15 PC", "Print the attached Delivery Note", "attach one Kanban Label"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("supplier email missing %q", expected)
		}
	}
}

func TestSendSupplierRecordsTransportFailureAsFailedDelivery(t *testing.T) {
	store := &recordingEmailStore{supplierEmail: "supplier@example.com"}
	transport := &recordingTransport{err: errors.New("SMTP unavailable")}
	service := &Service{store: store, transport: transport, baseURL: "https://app.example.com"}
	order := purchaseorder.Order{
		ID: uuid.New(), TenantID: uuid.New(), SupplierID: uuid.New(), PONumber: "PO-1", SupplierName: "Supplier",
		ExpectedDeliveryDate: time.Now(), Documents: &purchaseorder.DocumentSummary{DeliveryNoteNumber: "DN-1"},
		Lines: []purchaseorder.OrderLine{{TotalKanban: decimal.NewFromInt(1)}},
	}
	document := purchaseorder.PDFDocument{Filename: "document.pdf", Content: []byte("%PDF")}

	err := service.SendSupplier(context.Background(), Actor{TenantID: order.TenantID, UserID: uuid.New()}, order, document, document, document)

	if err == nil || err.Error() != "SMTP unavailable" {
		t.Fatalf("SendSupplier error = %v", err)
	}
	if transport.message.To != "supplier@example.com" {
		t.Fatalf("transport recipient = %q", transport.message.To)
	}
	if store.finishedWith == nil || store.finishedWith.Error() != "SMTP unavailable" {
		t.Fatalf("FinishLog error = %v", store.finishedWith)
	}
}

func TestSendApprovalAppliesCompanyBrandingAndInlineLogo(t *testing.T) {
	approvalID, poID, approverID := uuid.New(), uuid.New(), uuid.New()
	store := &recordingEmailStore{
		branding: CompanyBranding{CompanyName: "PT Buyer", Logo: testLogoPNG(t), LogoMIME: "image/png"},
		approvalData: ApprovalMailData{ApprovalID: approvalID, PurchaseOrderID: poID, ApproverUserID: approverID, PONumber: "PO-BRAND",
			SupplierName: "PT Supplier", ApproverEmail: "director@example.com", ApproverName: "Director"},
	}
	transport := &recordingTransport{}
	service := &Service{store: store, transport: transport, baseURL: "https://app.example.com"}
	if err := service.SendApproval(context.Background(), Actor{TenantID: uuid.New(), UserID: uuid.New()}, poID); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(transport.message.HTML, "PT Buyer") || !strings.Contains(transport.message.HTML, "cid:company-logo") {
		t.Fatal("approval message did not use company branding")
	}
	if len(transport.message.Attachments) != 1 || !transport.message.Attachments[0].Inline {
		t.Fatalf("approval inline attachments = %#v", transport.message.Attachments)
	}
}

func TestSMTPTestUsesCompanyBranding(t *testing.T) {
	store := &recordingEmailStore{branding: CompanyBranding{CompanyName: "PT Buyer", Logo: testLogoPNG(t), LogoMIME: "image/png"}}
	transport := &recordingTransport{}
	service := &Service{store: store, transport: transport}
	if err := service.Test(context.Background(), Actor{TenantID: uuid.New(), UserID: uuid.New()}, "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"PT Buyer", "SMTP configuration is active", "cid:company-logo"} {
		if !strings.Contains(transport.message.HTML, expected) {
			t.Fatalf("SMTP test missing %q", expected)
		}
	}
}

func TestSendDecisionResultUsesCreatorRecipientAndDecisionLogType(t *testing.T) {
	store := &recordingEmailStore{branding: CompanyBranding{CompanyName: "PT Buyer"}, decisionData: DecisionMailData{
		PONumber: "PO-RESULT", RecipientEmail: "buyer@example.com", RecipientName: "Buyer", Status: "APPROVED", DecisionActor: "Director", DecisionAt: time.Now(),
	}}
	transport := &recordingTransport{}
	service := &Service{store: store, transport: transport}
	if err := service.SendDecisionResult(context.Background(), Actor{TenantID: uuid.New(), UserID: uuid.New()}, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if transport.message.To != "buyer@example.com" || !strings.Contains(transport.message.Subject, "APPROVED") || !strings.Contains(transport.message.HTML, "PO-RESULT") {
		t.Fatalf("decision message = %#v", transport.message)
	}
}
