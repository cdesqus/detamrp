package settings

import (
	"context"
	"github.com/google/uuid"
)

type Repository interface {
	ListUsers(context.Context, Actor, ListQuery) ([]User, int, error)
	CreateUser(context.Context, Actor, UserInput) (User, error)
	UpdateUser(context.Context, Actor, uuid.UUID, UserInput) (User, error)
	ListRoles(context.Context, Actor, ListQuery) ([]Role, int, error)
	CreateRole(context.Context, Actor, RoleInput) (Role, error)
	UpdateRole(context.Context, Actor, uuid.UUID, RoleInput) (Role, error)
	ListPermissions(context.Context, Actor) ([]Permission, error)
	GetApprovalConfig(context.Context, Actor) (ApprovalConfig, error)
	UpdateApprovalConfig(context.Context, Actor, ApprovalConfigInput) (ApprovalConfig, error)
}
type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{repo: r} }
func (s *Service) ListUsers(c context.Context, a Actor, q ListQuery) ([]User, int, error) {
	q.Normalize()
	return s.repo.ListUsers(c, a, q)
}
func (s *Service) CreateUser(c context.Context, a Actor, i UserInput) (User, error) {
	if f := i.NormalizeAndValidate(true); len(f) > 0 {
		return User{}, ValidationError{f}
	}
	return s.repo.CreateUser(c, a, i)
}
func (s *Service) UpdateUser(c context.Context, a Actor, id uuid.UUID, i UserInput) (User, error) {
	if f := i.NormalizeAndValidate(false); len(f) > 0 {
		return User{}, ValidationError{f}
	}
	if id == a.UserID && !i.Active {
		return User{}, ConflictError{FieldErrors{"active": "You cannot lock your own account"}}
	}
	return s.repo.UpdateUser(c, a, id, i)
}
func (s *Service) ListRoles(c context.Context, a Actor, q ListQuery) ([]Role, int, error) {
	q.Normalize()
	return s.repo.ListRoles(c, a, q)
}
func (s *Service) CreateRole(c context.Context, a Actor, i RoleInput) (Role, error) {
	if f := i.NormalizeAndValidate(true); len(f) > 0 {
		return Role{}, ValidationError{f}
	}
	return s.repo.CreateRole(c, a, i)
}
func (s *Service) UpdateRole(c context.Context, a Actor, id uuid.UUID, i RoleInput) (Role, error) {
	if f := i.NormalizeAndValidate(false); len(f) > 0 {
		return Role{}, ValidationError{f}
	}
	return s.repo.UpdateRole(c, a, id, i)
}
func (s *Service) ListPermissions(c context.Context, a Actor) ([]Permission, error) {
	return s.repo.ListPermissions(c, a)
}
func (s *Service) GetApprovalConfig(c context.Context, a Actor) (ApprovalConfig, error) {
	return s.repo.GetApprovalConfig(c, a)
}
func (s *Service) UpdateApprovalConfig(c context.Context, a Actor, i ApprovalConfigInput) (ApprovalConfig, error) {
	if f := i.Validate(); len(f) > 0 {
		return ApprovalConfig{}, ValidationError{f}
	}
	return s.repo.UpdateApprovalConfig(c, a, i)
}
