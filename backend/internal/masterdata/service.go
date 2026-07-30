package masterdata

import (
	"context"

	"github.com/google/uuid"
)

type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

type UnitRepository interface {
	ListUnits(context.Context, Actor, ListQuery) ([]Unit, int, error)
	GetUnit(context.Context, Actor, uuid.UUID) (Unit, error)
	CreateUnit(context.Context, Actor, UnitInput) (Unit, error)
	UpdateUnit(context.Context, Actor, uuid.UUID, UnitInput) (Unit, error)
}

type UnitService struct{ repo UnitRepository }

type SupplierRepository interface {
	ListSuppliers(context.Context, Actor, ListQuery) ([]Supplier, int, error)
	GetSupplier(context.Context, Actor, uuid.UUID) (Supplier, error)
	CreateSupplier(context.Context, Actor, SupplierInput) (Supplier, error)
	UpdateSupplier(context.Context, Actor, uuid.UUID, SupplierInput) (Supplier, error)
}
type SupplierService struct{ repo SupplierRepository }

type RawMaterialRepository interface {
	ListRawMaterials(context.Context, Actor, ListQuery) ([]RawMaterial, int, error)
	GetRawMaterial(context.Context, Actor, uuid.UUID) (RawMaterial, error)
	CreateRawMaterial(context.Context, Actor, RawMaterialInput) (RawMaterial, error)
	UpdateRawMaterial(context.Context, Actor, uuid.UUID, RawMaterialInput) (RawMaterial, error)
}
type RawMaterialService struct{ repo RawMaterialRepository }

func NewRawMaterialService(repo RawMaterialRepository) *RawMaterialService {
	return &RawMaterialService{repo: repo}
}
func (s *RawMaterialService) List(ctx context.Context, a Actor, q ListQuery) ([]RawMaterial, int, error) {
	q.Normalize()
	return s.repo.ListRawMaterials(ctx, a, q)
}
func (s *RawMaterialService) Get(ctx context.Context, a Actor, id uuid.UUID) (RawMaterial, error) {
	return s.repo.GetRawMaterial(ctx, a, id)
}
func (s *RawMaterialService) Create(ctx context.Context, a Actor, in RawMaterialInput) (RawMaterial, error) {
	if f := in.NormalizeAndValidate(); len(f) > 0 {
		return RawMaterial{}, ValidationError{Fields: f}
	}
	return s.repo.CreateRawMaterial(ctx, a, in)
}
func (s *RawMaterialService) Update(ctx context.Context, a Actor, id uuid.UUID, in RawMaterialInput) (RawMaterial, error) {
	if f := in.NormalizeAndValidate(); len(f) > 0 {
		return RawMaterial{}, ValidationError{Fields: f}
	}
	return s.repo.UpdateRawMaterial(ctx, a, id, in)
}

func NewSupplierService(repo SupplierRepository) *SupplierService {
	return &SupplierService{repo: repo}
}
func (s *SupplierService) List(ctx context.Context, a Actor, q ListQuery) ([]Supplier, int, error) {
	q.Normalize()
	return s.repo.ListSuppliers(ctx, a, q)
}
func (s *SupplierService) Get(ctx context.Context, a Actor, id uuid.UUID) (Supplier, error) {
	return s.repo.GetSupplier(ctx, a, id)
}
func (s *SupplierService) Create(ctx context.Context, a Actor, in SupplierInput) (Supplier, error) {
	if f := in.NormalizeAndValidate(); len(f) > 0 {
		return Supplier{}, ValidationError{Fields: f}
	}
	return s.repo.CreateSupplier(ctx, a, in)
}
func (s *SupplierService) Update(ctx context.Context, a Actor, id uuid.UUID, in SupplierInput) (Supplier, error) {
	if f := in.NormalizeAndValidate(); len(f) > 0 {
		return Supplier{}, ValidationError{Fields: f}
	}
	return s.repo.UpdateSupplier(ctx, a, id, in)
}

func NewUnitService(repo UnitRepository) *UnitService {
	return &UnitService{repo: repo}
}

func (s *UnitService) List(ctx context.Context, actor Actor, query ListQuery) ([]Unit, int, error) {
	query.Normalize()
	return s.repo.ListUnits(ctx, actor, query)
}
func (s *UnitService) Get(ctx context.Context, actor Actor, id uuid.UUID) (Unit, error) {
	return s.repo.GetUnit(ctx, actor, id)
}
func (s *UnitService) Create(ctx context.Context, actor Actor, input UnitInput) (Unit, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Unit{}, ValidationError{Fields: fields}
	}
	return s.repo.CreateUnit(ctx, actor, input)
}
func (s *UnitService) Update(ctx context.Context, actor Actor, id uuid.UUID, input UnitInput) (Unit, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Unit{}, ValidationError{Fields: fields}
	}
	return s.repo.UpdateUnit(ctx, actor, id, input)
}
