package category

import (
	"context"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/utils/slug"

	"github.com/google/uuid"
)

type Service interface {
	FindAll(ctx context.Context) ([]models.Category, error)
	FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error)
	FindBySlug(ctx context.Context, slug string) (models.Category, error)

	Create(ctx context.Context, category CreateCategoryRequest) (models.Category, error)
	Update(ctx context.Context, ID uuid.UUID, category UpdateCategoryRequest) (models.Category, error)
	Delete(ctx context.Context, ID uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{
		repo: repo,
	}
}

func (s *service) FindAll(ctx context.Context) ([]models.Category, error) {

	categories, err := s.repo.FindAll(ctx)

	if err != nil {
		return nil, err
	}

	return categories, nil

}

func (s *service) FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error) {
	category, err := s.repo.FindByID(ctx, ID)

	if err != nil {
		return models.Category{}, err
	}

	return category, nil
}

func (s *service) FindBySlug(ctx context.Context, slug string) (models.Category, error) {
	category, err := s.repo.FindBySlug(ctx, slug)

	if err != nil {
		return models.Category{}, err
	}

	return category, nil
}

func (s *service) Create(ctx context.Context, category CreateCategoryRequest) (models.Category, error) {

	request := models.Category{
		ID:          uuid.New(),
		Name:        category.Name,
		Description: category.Description,
		Slug:        slug.MakeSlug(category.Name),
	}

	created, err := s.repo.Create(ctx, request)

	if err != nil {
		return models.Category{}, err
	}

	return created, nil

}

func (s *service) Update(ctx context.Context, ID uuid.UUID, category UpdateCategoryRequest) (models.Category, error) {

	request := models.Category{
		Name:        category.Name,
		Description: category.Description,
		Slug:        slug.MakeSlug(category.Name),
	}

	updated, err := s.repo.Update(ctx, ID, request)

	if err != nil {
		return models.Category{}, err
	}

	return updated, nil

}

func (s *service) Delete(ctx context.Context, ID uuid.UUID) error {
	err := s.repo.Delete(ctx, ID)
	if err != nil {
		return err
	}

	return nil
}
