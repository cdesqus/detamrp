package settings

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeRepo struct {
	created UserInput
	updated UserInput
	company CompanyConfigInput
}

func (f *fakeRepo) GetCompanyConfig(context.Context, Actor) (CompanyConfig, error) {
	return CompanyConfig{CompanyName: "Existing Company"}, nil
}
func (f *fakeRepo) UpdateCompanyConfig(_ context.Context, _ Actor, in CompanyConfigInput) (CompanyConfig, error) {
	f.company = in
	return CompanyConfig{CompanyName: in.CompanyName}, nil
}

func (f *fakeRepo) ListUsers(context.Context, Actor, ListQuery) ([]User, int, error) {
	return nil, 0, nil
}
func (f *fakeRepo) CreateUser(_ context.Context, _ Actor, in UserInput) (User, error) {
	f.created = in
	return User{Username: in.Username, Email: in.Email}, nil
}
func (f *fakeRepo) UpdateUser(_ context.Context, _ Actor, _ uuid.UUID, in UserInput) (User, error) {
	f.updated = in
	return User{Username: in.Username}, nil
}
func (f *fakeRepo) ListRoles(context.Context, Actor, ListQuery) ([]Role, int, error) {
	return nil, 0, nil
}
func (f *fakeRepo) CreateRole(context.Context, Actor, RoleInput) (Role, error) { return Role{}, nil }
func (f *fakeRepo) UpdateRole(context.Context, Actor, uuid.UUID, RoleInput) (Role, error) {
	return Role{}, nil
}
func (f *fakeRepo) ListPermissions(context.Context, Actor) ([]Permission, error) { return nil, nil }
func (f *fakeRepo) GetApprovalConfig(context.Context, Actor) (ApprovalConfig, error) {
	return ApprovalConfig{}, nil
}
func (f *fakeRepo) UpdateApprovalConfig(context.Context, Actor, ApprovalConfigInput) (ApprovalConfig, error) {
	return ApprovalConfig{}, nil
}

func TestServiceNormalizesUserAndRequiresCreatePassword(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo)
	actor := Actor{TenantID: uuid.New(), UserID: uuid.New()}
	role := uuid.New()
	_, err := service.CreateUser(context.Background(), actor, UserInput{Username: " Director ", DisplayName: " Director ", Email: " DIRECTOR@EXAMPLE.COM ", Password: "12345678", RoleIDs: []uuid.UUID{role}, Active: true, IsPurchaseOrderApprover: true})
	if err != nil {
		t.Fatal(err)
	}
	if repo.created.Username != "director" || repo.created.Email != "director@example.com" {
		t.Fatalf("not normalized: %#v", repo.created)
	}
	if !repo.created.IsPurchaseOrderApprover {
		t.Fatal("approver flag was not forwarded")
	}
	_, err = service.CreateUser(context.Background(), actor, UserInput{Username: "x", DisplayName: "X", Email: "x@example.com", RoleIDs: []uuid.UUID{role}, Active: true})
	if _, ok := err.(ValidationError); !ok {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestServiceNormalizesAndRequiresCompanyName(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo)
	actor := Actor{TenantID: uuid.New(), UserID: uuid.New()}
	item, err := service.UpdateCompanyConfig(context.Background(), actor, CompanyConfigInput{CompanyName: "  PT Buyer Indonesia  "})
	if err != nil || item.CompanyName != "PT Buyer Indonesia" || repo.company.CompanyName != "PT Buyer Indonesia" {
		t.Fatalf("company update = %#v, %v", item, err)
	}
	if _, err = service.UpdateCompanyConfig(context.Background(), actor, CompanyConfigInput{}); err == nil {
		t.Fatal("empty company name was accepted")
	}
}

func TestServiceRejectsSelfLock(t *testing.T) {
	repo := &fakeRepo{}
	service := NewService(repo)
	actor := Actor{TenantID: uuid.New(), UserID: uuid.New()}
	_, err := service.UpdateUser(context.Background(), actor, actor.UserID, UserInput{Username: "admin", DisplayName: "Admin", Email: "admin@example.com", RoleIDs: []uuid.UUID{uuid.New()}, Active: false})
	if _, ok := err.(ConflictError); !ok {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestRoleAndApproverInputValidation(t *testing.T) {
	role := RoleInput{Code: " director ", Name: "", PermissionCodes: nil, Active: true}
	fields := role.NormalizeAndValidate(true)
	if role.Code != "DIRECTOR" || fields["name"] == "" || fields["permissionCodes"] == "" {
		t.Fatalf("unexpected role validation: %#v %#v", role, fields)
	}
	if fields := (&ApprovalConfigInput{}).Validate(); fields["defaultApproverUserId"] == "" {
		t.Fatal("approver must be required")
	}
}

func TestPermissionGroupsMatchNavigationModules(t *testing.T) {
	cases := map[string]string{
		"dashboard.view":   "Dashboard",
		"master_data.view": "Data Master",
		"po.view":          "Procurement",
		"receiving.view":   "Logistics",
		"inventory.view":   "Inventory",
		"integration.view": "Integration",
		"role.manage":      "Settings",
	}
	for code, want := range cases {
		if got := permissionGroup(code); got != want {
			t.Errorf("permissionGroup(%q) = %q, want %q", code, got, want)
		}
	}
}
