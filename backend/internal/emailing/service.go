package emailing

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"order-stock/backend/internal/purchaseorder"
)

type Service struct {
	store     emailStore
	box       *SecretBox
	transport mailTransport
	baseURL   string
}

type emailStore interface {
	GetSMTP(context.Context, uuid.UUID) (storedSMTP, error)
	SaveSMTP(context.Context, Actor, SMTPSettingsInput, []byte, bool) error
	CreateLog(context.Context, Actor, string, string, uuid.UUID, string, string, string) (uuid.UUID, error)
	FinishLog(context.Context, uuid.UUID, uuid.UUID, error) error
	ListLogs(context.Context, Actor, string) ([]EmailLog, error)
	ApprovalData(context.Context, Actor, uuid.UUID) (ApprovalMailData, error)
	SaveApprovalToken(context.Context, uuid.UUID, uuid.UUID, []byte, time.Time) error
	ResolveToken(context.Context, uuid.UUID, []byte) (TokenContext, error)
	MarkTokenUsed(context.Context, uuid.UUID, []byte) error
	SupplierEmail(context.Context, uuid.UUID, uuid.UUID) (string, error)
	Branding(context.Context, uuid.UUID) (CompanyBranding, error)
	DecisionData(context.Context, uuid.UUID, uuid.UUID) (DecisionMailData, error)
}

type mailTransport interface {
	Send(SMTPSettings, string, Message) error
}

func NewService(store emailStore, box *SecretBox, baseURL string) *Service {
	return &Service{store: store, box: box, transport: SMTPTransport{}, baseURL: strings.TrimRight(baseURL, "/")}
}
func (s *Service) GetSettings(ctx context.Context, actor Actor) (SMTPSettings, error) {
	x, e := s.store.GetSMTP(ctx, actor.TenantID)
	return x.SMTPSettings, e
}
func (s *Service) UpdateSettings(ctx context.Context, actor Actor, in SMTPSettingsInput) (SMTPSettings, map[string]string, error) {
	in.Normalize()
	current, e := s.store.GetSMTP(ctx, actor.TenantID)
	if e != nil {
		return SMTPSettings{}, nil, e
	}
	if f := in.Validate(current.PasswordSet); len(f) > 0 {
		return SMTPSettings{}, f, nil
	}
	var encrypted []byte
	keep := in.Password == ""
	if !keep {
		encrypted, e = s.box.Encrypt(in.Password)
		if e != nil {
			return SMTPSettings{}, nil, e
		}
	}
	if e = s.store.SaveSMTP(ctx, actor, in, encrypted, keep); e != nil {
		return SMTPSettings{}, nil, e
	}
	settings, err := s.GetSettings(ctx, actor)
	return settings, nil, err
}
func (s *Service) smtp(ctx context.Context, tenantID uuid.UUID) (SMTPSettings, string, error) {
	stored, e := s.store.GetSMTP(ctx, tenantID)
	if e != nil {
		return SMTPSettings{}, "", e
	}
	input := SMTPSettingsInput{Host: stored.Host, Port: stored.Port, Security: stored.Security, Username: stored.Username, FromName: stored.FromName, FromEmail: stored.FromEmail}
	if f := input.Validate(stored.PasswordSet); len(f) > 0 {
		return SMTPSettings{}, "", errors.New("SMTP settings are incomplete")
	}
	password := ""
	if stored.PasswordSet {
		password, e = s.box.Decrypt(stored.PasswordEncrypted)
	}
	return stored.SMTPSettings, password, e
}
func (s *Service) deliver(ctx context.Context, actor Actor, typ, refType string, refID uuid.UUID, refNumber string, message Message) error {
	config, password, e := s.smtp(ctx, actor.TenantID)
	logID, logErr := s.store.CreateLog(ctx, actor, typ, refType, refID, refNumber, message.To, message.Subject)
	if logErr != nil {
		return logErr
	}
	if e == nil {
		e = s.transport.Send(config, password, message)
	}
	_ = s.store.FinishLog(ctx, actor.TenantID, logID, e)
	return e
}
func (s *Service) Test(ctx context.Context, actor Actor, to string) error {
	to = strings.ToLower(strings.TrimSpace(to))
	a, e := mail.ParseAddress(to)
	if e != nil || a.Address != to {
		return errors.New("valid recipient email is required")
	}
	branding, e := s.store.Branding(ctx, actor.TenantID)
	if e != nil {
		return e
	}
	rendered := emailShellWithBranding(branding, "SMTP Test Successful", `<p style="margin:0;color:#52525b;line-height:1.6">Your SMTP configuration is active and DETA MRP can deliver branded business notifications.</p><div style="margin-top:18px;padding:14px 16px;border:1px solid #bbf7d0;background:#f0fdf4;border-radius:8px;color:#166534"><b>Connection verified</b><br><span style="font-size:13px">Approval and supplier emails are ready to send.</span></div>`)
	return s.deliver(ctx, actor, "TEST", "", uuid.Nil, "", Message{To: to, Subject: "DETA MRP SMTP Test", HTML: rendered.HTML, Attachments: rendered.Inline})
}
func (s *Service) SendApproval(ctx context.Context, actor Actor, poID uuid.UUID) error {
	data, e := s.store.ApprovalData(ctx, actor, poID)
	if e != nil {
		return e
	}
	token, hash, e := NewToken()
	if e != nil {
		return e
	}
	if e = s.store.SaveApprovalToken(ctx, actor.TenantID, data.ApprovalID, hash, time.Now().Add(72*time.Hour)); e != nil {
		return e
	}
	base := fmt.Sprintf("%s/api/public/email-approval?tenant=%s&token=%s", s.baseURL, actor.TenantID, token)
	branding, e := s.store.Branding(ctx, actor.TenantID)
	if e != nil {
		return e
	}
	rendered := approvalEmail(data, branding, base+"&decision=approve", base+"&decision=reject", fmt.Sprintf("%s/supplier-orders/%s", s.baseURL, poID))
	return s.deliver(ctx, actor, "APPROVAL", "PURCHASE_ORDER", poID, data.PONumber, Message{To: data.ApproverEmail, Subject: "Approval Required — " + data.PONumber + " — " + data.SupplierName, HTML: rendered.HTML, Attachments: rendered.Inline})
}
func (s *Service) SendSupplier(ctx context.Context, actor Actor, order purchaseorder.Order, po, dn, labels purchaseorder.PDFDocument) error {
	if order.Documents == nil {
		return errors.New("delivery documents are not available")
	}
	recipient, e := s.store.SupplierEmail(ctx, actor.TenantID, order.SupplierID)
	if e != nil {
		return e
	}
	branding, e := s.store.Branding(ctx, actor.TenantID)
	if e != nil {
		return e
	}
	message, e := supplierMessageWithBranding(order, branding, recipient, po, dn, labels)
	if e != nil {
		return e
	}
	return s.deliver(ctx, actor, "SUPPLIER", "PURCHASE_ORDER", order.ID, order.PONumber, message)
}

func supplierMessage(order purchaseorder.Order, recipient string, po, dn, labels purchaseorder.PDFDocument) (Message, error) {
	return supplierMessageWithBranding(order, CompanyBranding{CompanyName: order.CompanyName}, recipient, po, dn, labels)
}

func supplierMessageWithBranding(order purchaseorder.Order, branding CompanyBranding, recipient string, po, dn, labels purchaseorder.PDFDocument) (Message, error) {
	if order.Documents == nil {
		return Message{}, errors.New("delivery documents are not available")
	}
	attachments := []Attachment{
		{Filename: po.Filename, ContentType: "application/pdf", Content: po.Content},
		{Filename: dn.Filename, ContentType: "application/pdf", Content: dn.Content},
		{Filename: labels.Filename, ContentType: "application/pdf", Content: labels.Content},
	}
	size := 0
	for _, a := range attachments {
		size += len(a.Content)
	}
	if size > 20*1024*1024 {
		return Message{}, errors.New("email attachments exceed 20 MB")
	}
	rendered := supplierEmailHTML(order, branding)
	attachments = append(attachments, rendered.Inline...)
	return Message{To: recipient, Subject: "Purchase Order & Delivery Documents — " + order.PONumber, HTML: rendered.HTML, Attachments: attachments}, nil
}
func (s *Service) ListLogs(ctx context.Context, actor Actor, search string) ([]EmailLog, error) {
	return s.store.ListLogs(ctx, actor, search)
}

func (s *Service) SendDecisionResult(ctx context.Context, actor Actor, approvalID uuid.UUID) error {
	data, err := s.store.DecisionData(ctx, actor.TenantID, approvalID)
	if err != nil {
		return err
	}
	branding, err := s.store.Branding(ctx, actor.TenantID)
	if err != nil {
		return err
	}
	rendered := decisionEmail(data, branding)
	subject := fmt.Sprintf("PO %s — %s", data.Status, data.PONumber)
	return s.deliver(ctx, actor, "DECISION", "PURCHASE_ORDER", data.PurchaseOrderID, data.PONumber,
		Message{To: data.RecipientEmail, Subject: subject, HTML: rendered.HTML, Attachments: rendered.Inline})
}
