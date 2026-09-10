package bom

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}
type Repository interface {
	List(context.Context, Actor) ([]BOM, error)
	Get(context.Context, Actor, uuid.UUID) (BOM, error)
	Create(context.Context, Actor, Input) (BOM, error)
	Update(context.Context, Actor, uuid.UUID, Input) (BOM, error)
	Activate(context.Context, Actor, uuid.UUID) (BOM, error)
}
type Service struct{ repository Repository }

func NewService(repository Repository) *Service                     { return &Service{repository} }
func (s *Service) List(ctx context.Context, a Actor) ([]BOM, error) { return s.repository.List(ctx, a) }
func (s *Service) Get(ctx context.Context, a Actor, id uuid.UUID) (BOM, error) {
	return s.repository.Get(ctx, a, id)
}
func (s *Service) Create(ctx context.Context, a Actor, input Input) (BOM, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return BOM{}, fmt.Errorf("validation failed: %v", fields)
	}
	return s.repository.Create(ctx, a, input)
}
func (s *Service) Update(ctx context.Context, a Actor, id uuid.UUID, input Input) (BOM, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 { return BOM{}, fmt.Errorf("validation failed: %v", fields) }
	return s.repository.Update(ctx, a, id, input)
}
func (s *Service) Activate(ctx context.Context, a Actor, id uuid.UUID) (BOM, error) {
	return s.repository.Activate(ctx, a, id)
}
