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
	return s.deliver(ctx, actor, "TEST", "", uuid.Nil, "", Message{To: to, Subject: "Order Stock SMTP Test", HTML: emailShell("SMTP Test", `<p>Your Order Stock SMTP configuration is working.</p>`)})
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
	html := approvalHTML(data, base+"&decision=approve", base+"&decision=reject", fmt.Sprintf("%s/supplier-orders/%s", s.baseURL, poID))
	return s.deliver(ctx, actor, "APPROVAL", "PURCHASE_ORDER", poID, data.PONumber, Message{To: data.ApproverEmail, Subject: "Approval Required — " + data.PONumber + " — " + data.SupplierName, HTML: html})
}
func (s *Service) SendSupplier(ctx context.Context, actor Actor, order purchaseorder.Order, po, dn, labels purchaseorder.PDFDocument) error {
	if order.Documents == nil {
		return errors.New("delivery documents are not available")
	}
	recipient, e := s.store.SupplierEmail(ctx, actor.TenantID, order.SupplierID)
	if e != nil {
		return e
	}
	message, e := supplierMessage(order, recipient, po, dn, labels)
	if e != nil {
		return e
	}
	return s.deliver(ctx, actor, "SUPPLIER", "PURCHASE_ORDER", order.ID, order.PONumber, message)
}

func supplierMessage(order purchaseorder.Order, recipient string, po, dn, labels purchaseorder.PDFDocument) (Message, error) {
	if order.Documents == nil {
		return Message{}, errors.New("delivery documents are not available")
	}
	totalKanban := int64(0)
	for _, line := range order.Lines {
		totalKanban += line.TotalKanban.IntPart()
	}
	attachments := []Attachment{{po.Filename, "application/pdf", po.Content}, {dn.Filename, "application/pdf", dn.Content}, {labels.Filename, "application/pdf", labels.Content}}
	size := 0
	for _, a := range attachments {
		size += len(a.Content)
	}
	if size > 20*1024*1024 {
		return Message{}, errors.New("email attachments exceed 20 MB")
	}
	body := supplierHTML(order.PONumber, order.SupplierName, order.Documents.DeliveryNoteNumber, order.ExpectedDeliveryDate.Format("02 Jan 2006"), len(order.Lines), totalKanban)
	return Message{To: recipient, Subject: "Purchase Order & Delivery Documents — " + order.PONumber, HTML: body, Attachments: attachments}, nil
}
func (s *Service) ListLogs(ctx context.Context, actor Actor, search string) ([]EmailLog, error) {
	return s.store.ListLogs(ctx, actor, search)
}
