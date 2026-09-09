package salesmaster

import (
	"context"

	"github.com/google/uuid"
	"order-stock/backend/internal/rbac"
)

type Repository interface {
	ListCustomers(context.Context, Actor, ListQuery) ([]Customer, int, error)
	GetCustomer(context.Context, Actor, uuid.UUID) (Customer, error)
	CreateCustomer(context.Context, Actor, CustomerInput) (Customer, error)
	UpdateCustomer(context.Context, Actor, uuid.UUID, CustomerInput) (Customer, error)
	ListFinishedGoods(context.Context, Actor, ListQuery) ([]FinishedGood, int, error)
	GetFinishedGood(context.Context, Actor, uuid.UUID) (FinishedGood, error)
	CreateFinishedGood(context.Context, Actor, FinishedGoodInput) (FinishedGood, error)
	UpdateFinishedGood(context.Context, Actor, uuid.UUID, FinishedGoodInput) (FinishedGood, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) ListCustomers(ctx context.Context, actor Actor, query ListQuery) ([]Customer, int, error) {
	query.Normalize()
	return s.repository.ListCustomers(ctx, actor, query)
}

func (s *Service) GetCustomer(ctx context.Context, actor Actor, id uuid.UUID) (Customer, error) {
	return s.repository.GetCustomer(ctx, actor, id)
}

func (s *Service) CreateCustomer(ctx context.Context, actor Actor, input CustomerInput) (Customer, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Customer{}, ValidationError{Fields: fields}
	}
	return s.repository.CreateCustomer(ctx, actor, input)
}

func (s *Service) UpdateCustomer(ctx context.Context, actor Actor, id uuid.UUID, input CustomerInput) (Customer, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Customer{}, ValidationError{Fields: fields}
	}
	return s.repository.UpdateCustomer(ctx, actor, id, input)
}

func (s *Service) ListFinishedGoods(ctx context.Context, actor Actor, query ListQuery) ([]FinishedGood, int, error) {
	query.Normalize()
	return s.repository.ListFinishedGoods(ctx, actor, query)
}

func (s *Service) GetFinishedGood(ctx context.Context, actor Actor, id uuid.UUID) (FinishedGood, error) {
	return s.repository.GetFinishedGood(ctx, actor, id)
}

func (s *Service) CreateFinishedGood(ctx context.Context, actor Actor, input FinishedGoodInput) (FinishedGood, error) {
	if !rbac.Allows(actor.Permissions, "fg.price.manage") {
		return FinishedGood{}, ForbiddenError{Message: "FG Price Manage permission is required"}
	}
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return FinishedGood{}, ValidationError{Fields: fields}
	}
	return s.repository.CreateFinishedGood(ctx, actor, input)
}

func (s *Service) UpdateFinishedGood(ctx context.Context, actor Actor, id uuid.UUID, input FinishedGoodInput) (FinishedGood, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return FinishedGood{}, ValidationError{Fields: fields}
	}
	if !rbac.Allows(actor.Permissions, "fg.price.manage") {
		current, err := s.repository.GetFinishedGood(ctx, actor, id)
		if err != nil {
			return FinishedGood{}, err
		}
		if !current.SalesPrice.Equal(input.SalesPrice) || current.Currency != input.Currency {
			return FinishedGood{}, ForbiddenError{Message: "FG Price Manage permission is required to change price"}
		}
	}
	return s.repository.UpdateFinishedGood(ctx, actor, id, input)
}
