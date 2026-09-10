package salesorder

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type Repository interface {
	Create(context.Context, Actor, Input) (Order, error)
	Submit(context.Context, Actor, uuid.UUID) (Order, error)
	Delete(context.Context, Actor, uuid.UUID) error
	Get(context.Context, Actor, uuid.UUID) (Order, error)
	List(context.Context, Actor) ([]Order, error)
	CreateDelivery(context.Context, Actor, uuid.UUID, DeliveryInput) (Delivery, error)
	ListDeliveries(context.Context, Actor, uuid.UUID) ([]DeliverySummary, error)
	GetDelivery(context.Context, Actor, uuid.UUID) (DeliveryDetail, error)
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
func (s *Service) Delete(ctx context.Context, a Actor, id uuid.UUID) error {
	return s.repo.Delete(ctx, a, id)
}
func (s *Service) Get(ctx context.Context, a Actor, id uuid.UUID) (Order, error) {
	return s.repo.Get(ctx, a, id)
}
func (s *Service) List(ctx context.Context, a Actor) ([]Order, error) { return s.repo.List(ctx, a) }
func (s *Service) CreateDelivery(ctx context.Context, a Actor, orderID uuid.UUID, input DeliveryInput) (Delivery, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Delivery{}, fmt.Errorf("validation failed: %v", fields)
	}
	return s.repo.CreateDelivery(ctx, a, orderID, input)
}
func (s *Service) ListDeliveries(ctx context.Context, a Actor, orderID uuid.UUID) ([]DeliverySummary, error) {
	return s.repo.ListDeliveries(ctx, a, orderID)
}
func (s *Service) GetDelivery(ctx context.Context, a Actor, id uuid.UUID) (DeliveryDetail, error) {
	return s.repo.GetDelivery(ctx, a, id)
}
