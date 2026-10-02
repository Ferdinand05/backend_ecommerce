package product

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, product models.Product) (models.Product, error)

	FindAll(ctx context.Context) ([]models.Product, error)

	FindByID(ctx context.Context, id uuid.UUID) (models.Product, error)

	FindBySlug(ctx context.Context, slug string) (models.Product, error)

	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	Update(ctx context.Context, id uuid.UUID, product models.Product) (models.Product, error)

	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, product models.Product) (models.Product, error) {

	err := r.db.WithContext(ctx).Create(&product).Error
	if err != nil {
		return models.Product{}, fmt.Errorf("creating product:%w", err)
	}

	var createdProduct models.Product
	err = r.db.WithContext(ctx).
		Preload("Category").
		First(&createdProduct, product.ID).
		Error

	if err != nil {
		return models.Product{}, fmt.Errorf("finding created record:%w", err)
	}

	return createdProduct, nil

}

func (r *repository) FindAll(ctx context.Context) ([]models.Product, error) {

	var products []models.Product

	err := r.db.WithContext(ctx).Preload("Category").Find(&products).Error
	if err != nil {
		return nil, fmt.Errorf("finding products:%w", err)
	}

	return products, nil

}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (models.Product, error) {

	var product models.Product

	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Variants").
		First(&product, id).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Product{}, ErrorProductNotFound
		}

		return models.Product{}, fmt.Errorf("finding product:%w", err)
	}

	return product, nil

}

func (r *repository) FindBySlug(ctx context.Context, slug string) (models.Product, error) {

	var product models.Product

	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Variants").
		Where("slug = ?", slug).
		First(&product).
		Error

	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Product{}, ErrorProductNotFound
		}

		return models.Product{}, fmt.Errorf("finding product by slug:%w", err)
	}

	return product, nil

}

func (r *repository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {

	var exists bool

	err := r.db.WithContext(ctx).
		Model(&models.Product{}).
		Select("count(*) > 0").
		Where("slug = ?", slug).
		Find(&exists).
		Error

	if err != nil {
		return false, fmt.Errorf("checking product slug exists:%w", err)
	}

	return exists, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, product models.Product) (models.Product, error) {

	result := r.db.WithContext(ctx).
		Model(&models.Product{}).
		Where("id = ?", id).
		Updates(&product)

	if result.Error != nil {
		return models.Product{}, fmt.Errorf("updating product:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return models.Product{}, ErrorProductNotFound
	}

	var updatedProduct models.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		First(&updatedProduct, id).
		Error

	if err != nil {
		return models.Product{}, fmt.Errorf("finding updated record:%w", err)
	}

	return updatedProduct, nil

}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {

	result := r.db.WithContext(ctx).
		Delete(&models.Product{}, id)

	if result.Error != nil {
		return fmt.Errorf("deleting product:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorProductNotFound
	}

	return nil
}
