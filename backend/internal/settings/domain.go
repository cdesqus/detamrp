package settings

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FieldErrors map[string]string
type Actor struct{ TenantID, UserID uuid.UUID }
type ListQuery struct {
	Search        string
	Active        *bool
	Limit, Offset int
}

func (q *ListQuery) Normalize() {
	q.Search = strings.TrimSpace(q.Search)
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
}

type Audit struct {
	CreatedBy     uuid.UUID `json:"createdBy"`
	CreatedByName string    `json:"createdByName"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedBy     uuid.UUID `json:"updatedBy"`
	UpdatedByName string    `json:"updatedByName"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
type RoleSummary struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}
type User struct {
	ID                      uuid.UUID     `json:"id"`
	Username                string        `json:"username"`
	DisplayName             string        `json:"displayName"`
	Email                   string        `json:"email"`
	Active                  bool          `json:"active"`
	IsPurchaseOrderApprover bool          `json:"isPurchaseOrderApprover"`
	Roles                   []RoleSummary `json:"roles"`
	Permissions             []string      `json:"permissions"`
	Audit
}
type Role struct {
	ID              uuid.UUID `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Active          bool      `json:"active"`
	System          bool      `json:"system"`
	PermissionCodes []string  `json:"permissionCodes"`
	UserCount       int       `json:"userCount"`
	Audit
}
type Permission struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Group       string `json:"group"`
}
type ApprovalConfig struct {
	DefaultApproverUserID uuid.UUID `json:"defaultApproverUserId"`
	DisplayName           string    `json:"displayName"`
	Email                 string    `json:"email"`
}
type UserInput struct {
	Username                string      `json:"username"`
	DisplayName             string      `json:"displayName"`
	Email                   string      `json:"email"`
	Password                string      `json:"password"`
	RoleIDs                 []uuid.UUID `json:"roleIds"`
	Active                  bool        `json:"active"`
	IsPurchaseOrderApprover bool        `json:"isPurchaseOrderApprover"`
}

func (i *UserInput) NormalizeAndValidate(create bool) FieldErrors {
	i.Username = strings.ToLower(strings.TrimSpace(i.Username))
	i.DisplayName = strings.TrimSpace(i.DisplayName)
	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	f := FieldErrors{}
	if i.Username == "" {
		f["username"] = "Username is required"
	}
	if i.DisplayName == "" {
		f["displayName"] = "Display Name is required"
	}
	if a, e := mail.ParseAddress(i.Email); e != nil || a.Address != i.Email {
		f["email"] = "Valid email is required"
	}
	if create && len(i.Password) < 8 {
		f["password"] = "Password must be at least 8 characters"
	}
	if !create && i.Password != "" && len(i.Password) < 8 {
		f["password"] = "Password must be at least 8 characters"
	}
	if len(i.RoleIDs) == 0 {
		f["roleIds"] = "Select at least one role"
	}
	return f
}

type RoleInput struct {
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	PermissionCodes []string `json:"permissionCodes"`
	Active          bool     `json:"active"`
}

func (i *RoleInput) NormalizeAndValidate(create bool) FieldErrors {
	i.Code = strings.ToUpper(strings.TrimSpace(i.Code))
	i.Name = strings.TrimSpace(i.Name)
	f := FieldErrors{}
	if create && i.Code == "" {
		f["code"] = "Role Code is required"
	}
	if i.Name == "" {
		f["name"] = "Role Name is required"
	}
	if len(i.PermissionCodes) == 0 {
		f["permissionCodes"] = "Select at least one permission"
	}
	return f
}

type ApprovalConfigInput struct {
	DefaultApproverUserID uuid.UUID `json:"defaultApproverUserId"`
}

func (i *ApprovalConfigInput) Validate() FieldErrors {
	if i.DefaultApproverUserID == uuid.Nil {
		return FieldErrors{"defaultApproverUserId": "Default Approver is required"}
	}
	return nil
}
func permissionGroup(code string) string {
	switch {
	case strings.HasPrefix(code, "po."):
		return "Procurement"
	case strings.HasPrefix(code, "receiving."), strings.HasPrefix(code, "dn."):
		return "Logistics"
	case strings.HasPrefix(code, "inventory."):
		return "Inventory"
	case strings.HasPrefix(code, "integration."):
		return "Integration"
	case strings.HasPrefix(code, "master_data."):
		return "Master Data"
	default:
		return "Settings"
	}
}
