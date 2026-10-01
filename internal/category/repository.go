package category

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context) ([]models.Category, error)
	FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error)
	FindBySlug(ctx context.Context, slug string) (models.Category, error)

	Create(ctx context.Context, category models.Category) (models.Category, error)
	Update(ctx context.Context, ID uuid.UUID, category models.Category) (models.Category, error)
	Delete(ctx context.Context, ID uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{
		db: db,
	}
}

func (r *repository) FindAll(ctx context.Context) ([]models.Category, error) {
	var categories []models.Category

	err := r.db.WithContext(ctx).Find(&categories).Error
	if err != nil {
		return nil, fmt.Errorf("finding categories:%w", err)
	}

	return categories, nil

}

func (r *repository) FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error) {
	var category models.Category

	result := r.db.WithContext(ctx).First(&category, ID)
	if result.Error != nil {

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return models.Category{}, ErrorCategoryNotFound
		}

		return models.Category{}, fmt.Errorf("find category by id:%w", result.Error)
	}

	return category, nil
}

func (r *repository) FindBySlug(ctx context.Context, slug string) (models.Category, error) {
	var category models.Category

	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&category).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Category{}, ErrorCategoryNotFound
		}

		return models.Category{}, fmt.Errorf("find category by slug:%w", err)
	}

	return category, nil
}

func (r *repository) Create(ctx context.Context, category models.Category) (models.Category, error) {

	err := r.db.WithContext(ctx).Create(&category).Error
	if err != nil {
		return models.Category{}, fmt.Errorf("creating category:%w", err)
	}

	return category, nil

}

func (r *repository) Update(ctx context.Context, ID uuid.UUID, category models.Category) (models.Category, error) {

	result := r.db.WithContext(ctx).Model(&models.Category{}).
		Where("id = ?", ID).
		Updates(&category)

	if result.Error != nil {
		return models.Category{}, fmt.Errorf("updating category:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return models.Category{}, ErrorCategoryNotFound
	}

	var updatedCategory models.Category
	err := r.db.WithContext(ctx).First(&updatedCategory, ID).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Category{}, ErrorCategoryNotFound
		}

		return models.Category{}, fmt.Errorf("finding updated category:%w", err)
	}

	return updatedCategory, nil

}

func (r *repository) Delete(ctx context.Context, ID uuid.UUID) error {

	result := r.db.WithContext(ctx).
		Delete(&models.Category{}, ID)

	if result.Error != nil {
		return fmt.Errorf("deleting category:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorCategoryNotFound
	}

	return nil
}
