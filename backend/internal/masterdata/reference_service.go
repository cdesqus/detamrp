package masterdata

import (
	"context"

	"github.com/google/uuid"
)

type CategoryRepository interface {
	ListCategories(context.Context, Actor, ListQuery) ([]Category, int, error)
	GetCategory(context.Context, Actor, uuid.UUID) (Category, error)
	CreateCategory(context.Context, Actor, CategoryInput) (Category, error)
	UpdateCategory(context.Context, Actor, uuid.UUID, CategoryInput) (Category, error)
}

type CategoryService struct{ repo CategoryRepository }

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}
func (s *CategoryService) List(ctx context.Context, actor Actor, query ListQuery) ([]Category, int, error) {
	query.Normalize()
	return s.repo.ListCategories(ctx, actor, query)
}
func (s *CategoryService) Get(ctx context.Context, actor Actor, id uuid.UUID) (Category, error) {
	return s.repo.GetCategory(ctx, actor, id)
}
func (s *CategoryService) Create(ctx context.Context, actor Actor, input CategoryInput) (Category, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Category{}, ValidationError{Fields: fields}
	}
	return s.repo.CreateCategory(ctx, actor, input)
}
func (s *CategoryService) Update(ctx context.Context, actor Actor, id uuid.UUID, input CategoryInput) (Category, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Category{}, ValidationError{Fields: fields}
	}
	return s.repo.UpdateCategory(ctx, actor, id, input)
}

type PackingRepository interface {
	ListPackings(context.Context, Actor, ListQuery) ([]Packing, int, error)
	GetPacking(context.Context, Actor, uuid.UUID) (Packing, error)
	CreatePacking(context.Context, Actor, PackingInput) (Packing, error)
	UpdatePacking(context.Context, Actor, uuid.UUID, PackingInput) (Packing, error)
}

type PackingService struct{ repo PackingRepository }

func NewPackingService(repo PackingRepository) *PackingService {
	return &PackingService{repo: repo}
}
func (s *PackingService) List(ctx context.Context, actor Actor, query ListQuery) ([]Packing, int, error) {
	query.Normalize()
	return s.repo.ListPackings(ctx, actor, query)
}
func (s *PackingService) Get(ctx context.Context, actor Actor, id uuid.UUID) (Packing, error) {
	return s.repo.GetPacking(ctx, actor, id)
}
func (s *PackingService) Create(ctx context.Context, actor Actor, input PackingInput) (Packing, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Packing{}, ValidationError{Fields: fields}
	}
	return s.repo.CreatePacking(ctx, actor, input)
}
func (s *PackingService) Update(ctx context.Context, actor Actor, id uuid.UUID, input PackingInput) (Packing, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Packing{}, ValidationError{Fields: fields}
	}
	return s.repo.UpdatePacking(ctx, actor, id, input)
}

type PlantRepository interface {
	ListPlants(context.Context, Actor, ListQuery) ([]Plant, int, error)
	GetPlant(context.Context, Actor, uuid.UUID) (Plant, error)
	CreatePlant(context.Context, Actor, PlantInput) (Plant, error)
	UpdatePlant(context.Context, Actor, uuid.UUID, PlantInput) (Plant, error)
}

type PlantService struct{ repo PlantRepository }

func NewPlantService(repo PlantRepository) *PlantService {
	return &PlantService{repo: repo}
}
func (s *PlantService) List(ctx context.Context, actor Actor, query ListQuery) ([]Plant, int, error) {
	query.Normalize()
	return s.repo.ListPlants(ctx, actor, query)
}
func (s *PlantService) Get(ctx context.Context, actor Actor, id uuid.UUID) (Plant, error) {
	return s.repo.GetPlant(ctx, actor, id)
}
func (s *PlantService) Create(ctx context.Context, actor Actor, input PlantInput) (Plant, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Plant{}, ValidationError{Fields: fields}
	}
	return s.repo.CreatePlant(ctx, actor, input)
}
func (s *PlantService) Update(ctx context.Context, actor Actor, id uuid.UUID, input PlantInput) (Plant, error) {
	if fields := input.NormalizeAndValidate(); len(fields) > 0 {
		return Plant{}, ValidationError{Fields: fields}
	}
	return s.repo.UpdatePlant(ctx, actor, id, input)
}
