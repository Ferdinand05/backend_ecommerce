package productimage

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
	Create(ctx context.Context, image models.ProductImage) error

	FindByID(ctx context.Context, id uuid.UUID) (models.ProductImage, error)

	FindAllByProductID(
		ctx context.Context,
		productID uuid.UUID,
	) ([]models.ProductImage, error)

	FindAllByVariantID(
		ctx context.Context,
		productID uuid.UUID,
		variantID uuid.UUID,
	) ([]models.ProductImage, error)

	Delete(ctx context.Context, id uuid.UUID) error
}



type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db:db}
}

func (r *repository) Create(ctx context.Context, image models.ProductImage) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(&image).Error; err != nil {
		return fmt.Errorf("creating product image:%w", err)
	}

	return nil
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (models.ProductImage, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var image models.ProductImage

	err := db.First(&image, id).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ProductImage{}, ErrorProductImageNotFound
		}

		return models.ProductImage{}, fmt.Errorf("finding product image:%w", err)
	}

	return image, nil
}

func (r *repository) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductImage, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var images []models.ProductImage

	err := db.
		Where("product_id = ? AND product_variant_id IS NULL", productID).
		Order("sort_order ASC, created_at ASC").
		Find(&images).
		Error

	if err != nil {
		return nil, fmt.Errorf("finding product images:%w", err)
	}

	return images, nil
}

func (r *repository) FindAllByVariantID(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) ([]models.ProductImage, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var images []models.ProductImage

	err := db.
		Where("product_id = ? AND product_variant_id = ?", productID, variantID).
		Order("sort_order ASC, created_at ASC").
		Find(&images).
		Error

	if err != nil {
		return nil, fmt.Errorf("finding product variant images:%w", err)
	}

	return images, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Delete(&models.ProductImage{}, id)

	if result.Error != nil {
		return fmt.Errorf("deleting product image:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorProductImageNotFound
	}

	return nil
}