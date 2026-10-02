package productvariant

import (
	"context"
	"errors"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/product"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	Create(
		ctx context.Context,
		productID uuid.UUID,
		req CreateProductVariantRequest,
	) (models.ProductVariant, error)

	FindAllByProductID(
		ctx context.Context,
		productID uuid.UUID,
	) ([]models.ProductVariant, error)

	FindByID(
		ctx context.Context,
		productID uuid.UUID,
		id uuid.UUID,
	) (models.ProductVariant, error)

	Update(
		ctx context.Context,
		productID uuid.UUID,
		id uuid.UUID,
		req UpdateProductVariantRequest,
	) (models.ProductVariant, error)

	Delete(
		ctx context.Context,
		productID uuid.UUID,
		id uuid.UUID,
	) error
}

type service struct {
	repo        Repository
	productRepo product.Repository
	db          *gorm.DB
}

func NewService(repo Repository, productRepo product.Repository, db *gorm.DB) *service {
	return &service{
		repo:        repo,
		productRepo: productRepo,
		db:          db,
	}
}

func (s *service) Create(ctx context.Context, productID uuid.UUID, req CreateProductVariantRequest) (models.ProductVariant, error) {

	_, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return models.ProductVariant{}, err
	}

	_, err = s.repo.FindBySKU(ctx, req.SKU)
	if err == nil {
		return models.ProductVariant{}, ErrorVariantSKUAlreadyExists
	}

	if !errors.Is(err, ErrorProductVariantNotFound) {
		return models.ProductVariant{}, err
	}

	variant := models.ProductVariant{
		ID:        uuid.New(),
		ProductID: productID,
		SKU:       req.SKU,
		Name:      req.Name,
		Price:     req.Price,
		IsActive:  true,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)
		return s.repo.Create(txCtx, variant)
	})

	if err != nil {
		return models.ProductVariant{}, err
	}

	return variant, nil
}

func (s *service) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error) {

	variants, err := s.repo.FindAllByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}

	return variants, nil
}

func (s *service) FindByID(ctx context.Context, productID uuid.UUID, id uuid.UUID) (models.ProductVariant, error) {

	variant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return models.ProductVariant{}, err
	}

	if variant.ProductID != productID {
		return models.ProductVariant{}, ErrorProductVariantNotFound
	}

	return variant, nil
}

func (s *service) Update(ctx context.Context, productID uuid.UUID, id uuid.UUID, req UpdateProductVariantRequest) (models.ProductVariant, error) {

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return models.ProductVariant{}, err
	}

	if current.ProductID != productID {
		return models.ProductVariant{}, ErrorProductVariantNotFound
	}

	exists, err := s.repo.ExistsBySKUExceptID(ctx, req.SKU, id)
	if err != nil {
		return models.ProductVariant{}, err
	}

	if exists {
		return models.ProductVariant{}, ErrorVariantSKUAlreadyExists
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)
		return s.repo.Update(txCtx, id, models.ProductVariant{
			SKU:   req.SKU,
			Name:  req.Name,
			Price: req.Price,
		})
	})

	if err != nil {
		return models.ProductVariant{}, err
	}

	return models.ProductVariant{
		ID:        current.ID,
		ProductID: current.ProductID,
		SKU:       req.SKU,
		Name:      req.Name,
		Price:     req.Price,
		IsActive:  current.IsActive,
		CreatedAt: current.CreatedAt,
		UpdatedAt: current.UpdatedAt,
	}, nil
}

func (s *service) Delete(ctx context.Context, productID uuid.UUID, id uuid.UUID) error {

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if current.ProductID != productID {
		return ErrorProductVariantNotFound
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)
		return s.repo.Delete(txCtx, id)
	})

	if err != nil {
		return err
	}

	return nil
}
