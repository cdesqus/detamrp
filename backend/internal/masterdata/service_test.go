package masterdata

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeUnitRepository struct {
	created UnitInput
	actor   Actor
	result  Unit
	err     error
}

func (f *fakeUnitRepository) ListUnits(context.Context, Actor, ListQuery) ([]Unit, int, error) {
	return nil, 0, f.err
}
func (f *fakeUnitRepository) GetUnit(context.Context, Actor, uuid.UUID) (Unit, error) {
	return f.result, f.err
}
func (f *fakeUnitRepository) CreateUnit(_ context.Context, actor Actor, input UnitInput) (Unit, error) {
	f.actor, f.created = actor, input
	return f.result, f.err
}
func (f *fakeUnitRepository) UpdateUnit(context.Context, Actor, uuid.UUID, UnitInput) (Unit, error) {
	return f.result, f.err
}

func TestServiceUnitNormalizesAndForwardsAuditActor(t *testing.T) {
	tenant, user := uuid.New(), uuid.New()
	repo := &fakeUnitRepository{result: Unit{Code: "KG"}}
	service := NewUnitService(repo)
	result, err := service.Create(context.Background(), Actor{TenantID: tenant, UserID: user}, UnitInput{Code: " kg ", Name: " Kilogram "})
	if err != nil || result.Code != "KG" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if repo.created.Code != "KG" || repo.created.Name != "Kilogram" || repo.actor.UserID != user {
		t.Fatalf("forwarded=%+v actor=%+v", repo.created, repo.actor)
	}
}

func TestServiceUnitReturnsFieldErrorsBeforeStore(t *testing.T) {
	service := NewUnitService(&fakeUnitRepository{})
	_, err := service.Create(context.Background(), Actor{TenantID: uuid.New(), UserID: uuid.New()}, UnitInput{Code: "KANBAN", Name: "Lot"})
	var validation ValidationError
	if !errors.As(err, &validation) || validation.Fields["code"] == "" {
		t.Fatalf("got %v", err)
	}
}
