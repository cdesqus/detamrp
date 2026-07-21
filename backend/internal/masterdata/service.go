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
