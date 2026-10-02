package productvariant

import (
	"context"
	"errors"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, variant models.ProductVariant) error
	FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error)
	FindByID(ctx context.Context, id uuid.UUID) (models.ProductVariant, error)
	FindBySKU(ctx context.Context, sku string) (models.ProductVariant, error)
	ExistsBySKUExceptID(ctx context.Context, sku string, id uuid.UUID) (bool, error)
	Update(ctx context.Context, id uuid.UUID, variant models.ProductVariant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, variant models.ProductVariant) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(&variant).Error; err != nil {
		return fmt.Errorf("creating product variant:%w", err)
	}

	return nil
}

func (r *repository) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var variants []models.ProductVariant

	err := db.
		Where("product_id = ?", productID).
		Order("created_at ASC").
		Find(&variants).
		Error

	if err != nil {
		return nil, fmt.Errorf("finding product variants:%w", err)
	}

	return variants, nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (models.ProductVariant, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var variant models.ProductVariant

	err := db.First(&variant, id).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ProductVariant{}, ErrorProductVariantNotFound
		}

		return models.ProductVariant{}, fmt.Errorf("finding product variant:%w", err)
	}

	return variant, nil
}

func (r *repository) FindBySKU(ctx context.Context, sku string) (models.ProductVariant, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var variant models.ProductVariant

	err := db.Where("sku = ?", sku).First(&variant).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ProductVariant{}, ErrorProductVariantNotFound
		}

		return models.ProductVariant{}, fmt.Errorf("finding product variant by sku:%w", err)
	}

	return variant, nil
}

func (r *repository) ExistsBySKUExceptID(ctx context.Context, sku string, id uuid.UUID) (bool, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var exists bool

	err := db.
		Model(&models.ProductVariant{}).
		Select("count(*) > 0").
		Where("sku = ? AND id <> ?", sku, id).
		Find(&exists).
		Error

	if err != nil {
		return false, fmt.Errorf("checking product variant sku exists:%w", err)
	}

	return exists, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, variant models.ProductVariant) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.
		Model(&models.ProductVariant{}).
		Where("id = ?", id).
		Updates(&variant)

	if result.Error != nil {
		return fmt.Errorf("updating product variant:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorProductVariantNotFound
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Delete(&models.ProductVariant{}, id)

	if result.Error != nil {
		return fmt.Errorf("deleting product variant:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorProductVariantNotFound
	}

	return nil
}
