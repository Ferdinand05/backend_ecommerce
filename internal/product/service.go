package product

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/utils/slug"

	"github.com/google/uuid"
)

type Service interface {
	Create(
		ctx context.Context,
		req CreateProductRequest,
	) (models.Product, error)

	FindAll(
		ctx context.Context,
	) ([]models.Product, error)

	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (models.Product, error)

	FindBySlug(
		ctx context.Context,
		slug string,
	) (models.Product, error)

	Update(
		ctx context.Context,
		id uuid.UUID,
		req UpdateProductRequest,
	) (models.Product, error)

	Delete(
		ctx context.Context,
		id uuid.UUID,
	) error
}

type service struct {
	repo         Repository
	categoryRepo category.Repository
}

func NewService(repo Repository, categoryRepo category.Repository) *service {
	return &service{
		repo:         repo,
		categoryRepo: categoryRepo,
	}
}

func (s *service) Create(ctx context.Context, req CreateProductRequest) (models.Product, error) {

	generatedSlug := slug.MakeSlug(req.Name)

	exists, err := s.repo.ExistsBySlug(ctx, generatedSlug)
	if err != nil {
		return models.Product{}, err
	}

	if exists {
		return models.Product{}, ErrorProductSlugAlreadyExists
	}

	_, err = s.categoryRepo.FindByID(ctx, req.CategoryID)
	if err != nil {

		if errors.Is(err, category.ErrorCategoryNotFound) {
			return models.Product{}, category.ErrorCategoryNotFound
		}

		return models.Product{}, err
	}

	created, err := s.repo.Create(ctx, models.Product{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        generatedSlug,
		Description: req.Description,
		CategoryID:  req.CategoryID,
	})

	if err != nil {
		return models.Product{}, err
	}

	return created, nil

}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateProductRequest) (models.Product, error) {

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return models.Product{}, err
	}

	_, err = s.categoryRepo.FindByID(ctx, req.CategoryID)
	if err != nil {

		if errors.Is(err, category.ErrorCategoryNotFound) {
			return models.Product{}, category.ErrorCategoryNotFound
		}

		return models.Product{}, err
	}

	generatedSlug := slug.MakeSlug(req.Name)

	if current.Slug != generatedSlug {

		exists, err := s.repo.ExistsBySlug(ctx, generatedSlug)
		if err != nil {
			return models.Product{}, err
		}

		if exists {
			return models.Product{}, ErrorProductSlugAlreadyExists
		}
	}

	updated, err := s.repo.Update(ctx, id, models.Product{
		Name:        req.Name,
		Slug:        generatedSlug,
		Description: req.Description,
		CategoryID:  req.CategoryID,
	})

	if err != nil {
		return models.Product{}, err
	}

	return updated, nil

}

func (s *service) FindAll(ctx context.Context) ([]models.Product, error) {

	products, err := s.repo.FindAll(ctx)

	if err != nil {
		return nil, err
	}

	return products, nil

}

func (s *service) FindByID(ctx context.Context, ID uuid.UUID) (models.Product, error) {
	product, err := s.repo.FindByID(ctx, ID)

	if err != nil {
		return models.Product{}, err
	}

	return product, nil
}

func (s *service) FindBySlug(ctx context.Context, slug string) (models.Product, error) {
	product, err := s.repo.FindBySlug(ctx, slug)

	if err != nil {
		return models.Product{}, err
	}

	return product, nil
}

func (s *service) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	err := s.repo.Delete(ctx, id)

	if err != nil {
		return err
	}

	return nil
}
