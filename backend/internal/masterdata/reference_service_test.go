package masterdata

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeCategoryRepository struct {
	input CategoryInput
}

func (f *fakeCategoryRepository) ListCategories(context.Context, Actor, ListQuery) ([]Category, int, error) {
	return nil, 0, nil
}
func (f *fakeCategoryRepository) GetCategory(context.Context, Actor, uuid.UUID) (Category, error) {
	return Category{}, nil
}
func (f *fakeCategoryRepository) CreateCategory(_ context.Context, _ Actor, input CategoryInput) (Category, error) {
	f.input = input
	return Category{Code: input.Code}, nil
}
func (f *fakeCategoryRepository) UpdateCategory(context.Context, Actor, uuid.UUID, CategoryInput) (Category, error) {
	return Category{}, nil
}

type fakePackingRepository struct {
	input PackingInput
}

func (f *fakePackingRepository) ListPackings(context.Context, Actor, ListQuery) ([]Packing, int, error) {
	return nil, 0, nil
}
func (f *fakePackingRepository) GetPacking(context.Context, Actor, uuid.UUID) (Packing, error) {
	return Packing{}, nil
}
func (f *fakePackingRepository) CreatePacking(_ context.Context, _ Actor, input PackingInput) (Packing, error) {
	f.input = input
	return Packing{Code: input.Code}, nil
}
func (f *fakePackingRepository) UpdatePacking(context.Context, Actor, uuid.UUID, PackingInput) (Packing, error) {
	return Packing{}, nil
}

type fakePlantRepository struct {
	input PlantInput
}

func (f *fakePlantRepository) ListPlants(context.Context, Actor, ListQuery) ([]Plant, int, error) {
	return nil, 0, nil
}
func (f *fakePlantRepository) GetPlant(context.Context, Actor, uuid.UUID) (Plant, error) {
	return Plant{}, nil
}
func (f *fakePlantRepository) CreatePlant(_ context.Context, _ Actor, input PlantInput) (Plant, error) {
	f.input = input
	return Plant{Code: input.Code}, nil
}
func (f *fakePlantRepository) UpdatePlant(context.Context, Actor, uuid.UUID, PlantInput) (Plant, error) {
	return Plant{}, nil
}

func TestReferenceServicesNormalizeBeforeCreate(t *testing.T) {
	actor := Actor{TenantID: uuid.New(), UserID: uuid.New()}
	categoryRepo := &fakeCategoryRepository{}
	packingRepo := &fakePackingRepository{}
	plantRepo := &fakePlantRepository{}

	if _, err := NewCategoryService(categoryRepo).Create(context.Background(), actor, CategoryInput{Code: " raw ", Name: " Raw "}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPackingService(packingRepo).Create(context.Background(), actor, PackingInput{Code: " box ", Name: " Box "}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPlantService(plantRepo).Create(context.Background(), actor, PlantInput{Code: " jkt ", Name: " Jakarta "}); err != nil {
		t.Fatal(err)
	}

	if categoryRepo.input.Code != "RAW" || packingRepo.input.Code != "BOX" || plantRepo.input.Code != "JKT" {
		t.Fatalf("inputs not normalized: category=%#v packing=%#v plant=%#v", categoryRepo.input, packingRepo.input, plantRepo.input)
	}
}

func TestReferenceServicesRejectInvalidCreateBeforeRepository(t *testing.T) {
	actor := Actor{TenantID: uuid.New(), UserID: uuid.New()}
	_, err := NewCategoryService(&fakeCategoryRepository{}).Create(context.Background(), actor, CategoryInput{})
	var validation ValidationError
	if err == nil || !asValidationError(err, &validation) || validation.Fields["code"] == "" {
		t.Fatalf("error = %#v, want code validation error", err)
	}
}

func asValidationError(err error, target *ValidationError) bool {
	value, ok := err.(ValidationError)
	if ok {
		*target = value
	}
	return ok
}
