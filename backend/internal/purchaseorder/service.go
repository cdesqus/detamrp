package purchaseorder

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines purchase-order persistence operations. Implementations must
// enforce the same state rules while holding their database locks.
type Repository interface {
	ListOrders(context.Context, Actor, ListQuery) ([]Order, int, error)
	GetOrder(context.Context, Actor, uuid.UUID) (Order, error)
	CreateOrder(context.Context, Actor, OrderInput) (Order, error)
	UpdateOrder(context.Context, Actor, uuid.UUID, OrderInput) (Order, error)
	SubmitOrder(context.Context, Actor, uuid.UUID) (Order, error)
	CancelOrder(context.Context, Actor, uuid.UUID) (Order, error)
	ListApprovals(context.Context, Actor, ListQuery) ([]Approval, int, error)
	Approve(context.Context, Actor, uuid.UUID, DecisionInput) (Approval, error)
	Reject(context.Context, Actor, uuid.UUID, DecisionInput) (Approval, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, actor Actor, query ListQuery) ([]Order, int, error) {
	query.Normalize()
	return s.repo.ListOrders(ctx, actor, query)
}

func (s *Service) Get(ctx context.Context, actor Actor, id uuid.UUID) (Order, error) {
	return s.repo.GetOrder(ctx, actor, id)
}

func (s *Service) Create(ctx context.Context, actor Actor, input OrderInput) (Order, error) {
	if fields := input.NormalizeAndValidate(false); len(fields) > 0 {
		return Order{}, ValidationError{Fields: fields}
	}
	return s.repo.CreateOrder(ctx, actor, input)
}

func (s *Service) Update(ctx context.Context, actor Actor, id uuid.UUID, input OrderInput) (Order, error) {
	order, err := s.Get(ctx, actor, id)
	if err != nil {
		return Order{}, err
	}
	if order.Status != StatusDraft {
		return Order{}, draftConflict("edited")
	}
	if fields := input.NormalizeAndValidate(false); len(fields) > 0 {
		return Order{}, ValidationError{Fields: fields}
	}
	return s.repo.UpdateOrder(ctx, actor, id, input)
}

func (s *Service) Submit(ctx context.Context, actor Actor, id uuid.UUID) (Order, error) {
	order, err := s.Get(ctx, actor, id)
	if err != nil {
		return Order{}, err
	}
	if order.Status != StatusDraft {
		return Order{}, draftConflict("submitted")
	}
	if len(order.Lines) == 0 {
		return Order{}, ValidationError{Fields: FieldErrors{"lines": "At least one Raw Material is required before submission"}}
	}
	return s.repo.SubmitOrder(ctx, actor, id)
}

func (s *Service) Cancel(ctx context.Context, actor Actor, id uuid.UUID) (Order, error) {
	order, err := s.Get(ctx, actor, id)
	if err != nil {
		return Order{}, err
	}
	if order.Status != StatusDraft {
		return Order{}, draftConflict("cancelled")
	}
	return s.repo.CancelOrder(ctx, actor, id)
}

func (s *Service) ListApprovals(ctx context.Context, actor Actor, query ListQuery) ([]Approval, int, error) {
	query.Normalize()
	return s.repo.ListApprovals(ctx, actor, query)
}

func (s *Service) Approve(ctx context.Context, actor Actor, id uuid.UUID, input DecisionInput) (Approval, error) {
	input.NormalizeAndValidate(false)
	return s.repo.Approve(ctx, actor, id, input)
}

func (s *Service) Reject(ctx context.Context, actor Actor, id uuid.UUID, input DecisionInput) (Approval, error) {
	if fields := input.NormalizeAndValidate(true); len(fields) > 0 {
		return Approval{}, ValidationError{Fields: fields}
	}
	return s.repo.Reject(ctx, actor, id, input)
}

func draftConflict(action string) ConflictError {
	return ConflictError{Fields: FieldErrors{"status": "Only draft purchase orders can be " + action}}
}
