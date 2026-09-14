package production

import (
	"context"
	"github.com/google/uuid"
)

type Actor struct{ TenantID, UserID uuid.UUID }

type PlanRepository interface {
	Create(context.Context, Actor, Plan) (Plan, error)
	List(context.Context, Actor) ([]Plan, error)
	Get(context.Context, Actor, uuid.UUID) (Plan, error)
	Update(context.Context, Actor, uuid.UUID, Plan) (Plan, error)
	Delete(context.Context, Actor, uuid.UUID) error
	Approve(context.Context, Actor, uuid.UUID) (Plan, error)
	Close(context.Context, Actor, uuid.UUID) (Plan, error)
}

type PlanService struct{ repo PlanRepository }

func NewPlanService(repo PlanRepository) *PlanService { return &PlanService{repo: repo} }
func (s *PlanService) Create(ctx context.Context, a Actor, p Plan) (Plan, error) {
	if err := ValidatePlanInput(p); err != nil {
		return Plan{}, err
	}
	p.Status = PlanDraft
	return s.repo.Create(ctx, a, p)
}
func (s *PlanService) List(ctx context.Context, a Actor) ([]Plan, error) { return s.repo.List(ctx, a) }
func (s *PlanService) Get(ctx context.Context, a Actor, id uuid.UUID) (Plan, error) {
	return s.repo.Get(ctx, a, id)
}
func (s *PlanService) Update(ctx context.Context, a Actor, id uuid.UUID, p Plan) (Plan, error) {
	if err := ValidatePlanInput(p); err != nil {
		return Plan{}, err
	}
	return s.repo.Update(ctx, a, id, p)
}
func (s *PlanService) Delete(ctx context.Context, a Actor, id uuid.UUID) error {
	return s.repo.Delete(ctx, a, id)
}
func (s *PlanService) Approve(ctx context.Context, a Actor, id uuid.UUID) (Plan, error) {
	return s.repo.Approve(ctx, a, id)
}
func (s *PlanService) Close(ctx context.Context, a Actor, id uuid.UUID) (Plan, error) {
	return s.repo.Close(ctx, a, id)
}
