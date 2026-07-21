package masterdata

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeMeasurementRepository struct {
	created MeasurementInput
	actor   Actor
	result  Measurement
	err     error
}

func (f *fakeMeasurementRepository) ListMeasurements(context.Context, Actor, ListQuery) ([]Measurement, int, error) {
	return nil, 0, f.err
}
func (f *fakeMeasurementRepository) GetMeasurement(context.Context, Actor, uuid.UUID) (Measurement, error) {
	return f.result, f.err
}
func (f *fakeMeasurementRepository) CreateMeasurement(_ context.Context, actor Actor, input MeasurementInput) (Measurement, error) {
	f.actor, f.created = actor, input
	return f.result, f.err
}
func (f *fakeMeasurementRepository) UpdateMeasurement(context.Context, Actor, uuid.UUID, MeasurementInput) (Measurement, error) {
	return f.result, f.err
}

func TestServiceMeasurementNormalizesAndForwardsAuditActor(t *testing.T) {
	tenant, user := uuid.New(), uuid.New()
	repo := &fakeMeasurementRepository{result: Measurement{Code: "KG"}}
	service := NewMeasurementService(repo)
	result, err := service.Create(context.Background(), Actor{TenantID: tenant, UserID: user}, MeasurementInput{Code: " kg ", Name: " Kilogram "})
	if err != nil || result.Code != "KG" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if repo.created.Code != "KG" || repo.created.Name != "Kilogram" || repo.actor.UserID != user {
		t.Fatalf("forwarded=%+v actor=%+v", repo.created, repo.actor)
	}
}

func TestServiceMeasurementReturnsFieldErrorsBeforeStore(t *testing.T) {
	service := NewMeasurementService(&fakeMeasurementRepository{})
	_, err := service.Create(context.Background(), Actor{TenantID: uuid.New(), UserID: uuid.New()}, MeasurementInput{Code: "KANBAN", Name: "Lot"})
	var validation ValidationError
	if !errors.As(err, &validation) || validation.Fields["code"] == "" {
		t.Fatalf("got %v", err)
	}
}
