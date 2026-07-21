package masterdata

import (
	"context"

	"github.com/google/uuid"
)

type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

type MeasurementRepository interface {
	ListMeasurements(context.Context, Actor, ListQuery) ([]Measurement, int, error)
	GetMeasurement(context.Context, Actor, uuid.UUID) (Measurement, error)
	CreateMeasurement(context.Context, Actor, MeasurementInput) (Measurement, error)
	UpdateMeasurement(context.Context, Actor, uuid.UUID, MeasurementInput) (Measurement, error)
}

type MeasurementService struct{ repo MeasurementRepository }

func NewMeasurementService(repo MeasurementRepository) *MeasurementService {
	return &MeasurementService{repo: repo}
}

func (s *MeasurementService) List(ctx context.Context, actor Actor, query ListQuery) ([]Measurement, int, error) {
	query.Normalize()
	return s.repo.ListMeasurements(ctx, actor, query)
}
func (s *MeasurementService) Get(ctx context.Context, actor Actor, id uuid.UUID) (Measurement, error) {
	return s.repo.GetMeasurement(ctx, actor, id)
}
func (s *MeasurementService) Create(ctx context.Context, actor Actor, input MeasurementInput) (Measurement, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Measurement{}, ValidationError{Fields: fields}
	}
	return s.repo.CreateMeasurement(ctx, actor, input)
}
func (s *MeasurementService) Update(ctx context.Context, actor Actor, id uuid.UUID, input MeasurementInput) (Measurement, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Measurement{}, ValidationError{Fields: fields}
	}
	return s.repo.UpdateMeasurement(ctx, actor, id, input)
}
