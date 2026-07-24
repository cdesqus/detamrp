package emailing

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Actor struct{ TenantID, UserID uuid.UUID }

type SMTPSettings struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Security    string `json:"security"`
	Username    string `json:"username"`
	PasswordSet bool   `json:"passwordSet"`
	FromName    string `json:"fromName"`
	FromEmail   string `json:"fromEmail"`
}

type SMTPSettingsInput struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Security  string `json:"security"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromName  string `json:"fromName"`
	FromEmail string `json:"fromEmail"`
}

func (i *SMTPSettingsInput) Normalize() {
	i.Host = strings.TrimSpace(i.Host)
	i.Security = strings.ToUpper(strings.TrimSpace(i.Security))
	i.Username = strings.TrimSpace(i.Username)
	i.FromName = strings.TrimSpace(i.FromName)
	i.FromEmail = strings.ToLower(strings.TrimSpace(i.FromEmail))
}

func (i SMTPSettingsInput) Validate(passwordAlreadySet bool) map[string]string {
	f := map[string]string{}
	if i.Host == "" {
		f["host"] = "SMTP Host is required"
	}
	if i.Port < 1 || i.Port > 65535 {
		f["port"] = "Port must be between 1 and 65535"
	}
	if i.Security != "STARTTLS" && i.Security != "TLS" && i.Security != "NONE" {
		f["security"] = "Select STARTTLS, TLS, or NONE"
	}
	if i.Username != "" && i.Password == "" && !passwordAlreadySet {
		f["password"] = "Password is required when Username is set"
	}
	if i.FromName == "" {
		f["fromName"] = "From Name is required"
	}
	if a, err := mail.ParseAddress(i.FromEmail); err != nil || a.Address != i.FromEmail {
		f["fromEmail"] = "Valid From Email is required"
	}
	return f
}

type EmailLog struct {
	ID              uuid.UUID  `json:"id"`
	EmailType       string     `json:"emailType"`
	ReferenceNumber string     `json:"referenceNumber"`
	Recipient       string     `json:"recipient"`
	Subject         string     `json:"subject"`
	Status          string     `json:"status"`
	Attempts        int        `json:"attempts"`
	LastError       string     `json:"lastError"`
	SentAt          *time.Time `json:"sentAt"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type Message struct {
	To          string
	Subject     string
	HTML        string
	Attachments []Attachment
}
type Attachment struct {
	Filename, ContentType string
	Content               []byte
}

type ApprovalMailData struct {
	ApprovalID, PurchaseOrderID, ApproverUserID                        uuid.UUID
	PONumber, SupplierName, ApproverEmail, ApproverName, CreatedByName string
	OrderDate, ExpectedDeliveryDate                                    time.Time
	Currency, TotalAmount, Notes                                       string
	Lines                                                              []ApprovalMailLine
}
type ApprovalMailLine struct {
	Code, Name, Unit, QtyPerKanban string
	TotalKanban                    int64
	TotalQuantity                  string
}
