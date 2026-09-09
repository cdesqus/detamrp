package salesorder

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type Repository interface {
	Create(context.Context, Actor, Input) (Order, error)
	Submit(context.Context, Actor, uuid.UUID) (Order, error)
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo} }
func (s *Service) Create(ctx context.Context, a Actor, input Input) (Order, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Order{}, fmt.Errorf("validation failed: %v", fields)
	}
	return s.repo.Create(ctx, a, input)
}
func (s *Service) Submit(ctx context.Context, a Actor, id uuid.UUID) (Order, error) {
	return s.repo.Submit(ctx, a, id)
}
