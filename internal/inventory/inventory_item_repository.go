package inventory

import (
	"context"
	"errors"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InventoryRepository interface {
	Create(ctx context.Context, item models.InventoryItem) error

	FindByID(ctx context.Context, id uuid.UUID) (models.InventoryItem, error)

	FindByVariantID(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error)

	FindAll(ctx context.Context) ([]models.InventoryItem, error)

	UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) error

	FindByVariantIDForUpdate(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error)
}

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) *inventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) Create(ctx context.Context, item models.InventoryItem) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(&item).Error; err != nil {
		return fmt.Errorf("creating inventory item:%w", err)
	}

	return nil
}

func (r *inventoryRepository) FindByID(ctx context.Context, id uuid.UUID) (models.InventoryItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var item models.InventoryItem

	err := db.First(&item, id).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.InventoryItem{}, ErrorInventoryNotFound
		}

		return models.InventoryItem{}, fmt.Errorf("finding inventory item:%w", err)
	}

	return item, nil
}

func (r *inventoryRepository) FindByVariantID(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var item models.InventoryItem

	err := db.Where("product_variant_id = ?", variantID).First(&item).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.InventoryItem{}, ErrorInventoryNotFound
		}

		return models.InventoryItem{}, fmt.Errorf("finding inventory item by variant:%w", err)
	}

	return item, nil
}

func (r *inventoryRepository) FindAll(ctx context.Context) ([]models.InventoryItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var items []models.InventoryItem

	err := db.Order("created_at ASC").Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("finding inventory items:%w", err)
	}

	return items, nil
}

func (r *inventoryRepository) UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Model(&models.InventoryItem{}).
		Where("id = ?", id).
		Update("quantity", quantity)

	if result.Error != nil {
		return fmt.Errorf("updating inventory quantity:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorInventoryNotFound
	}

	return nil
}

func (r *inventoryRepository) FindByVariantIDForUpdate(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var item models.InventoryItem

	err := db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("product_variant_id = ?", variantID).
		First(&item).
		Error

	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.InventoryItem{}, ErrorInventoryNotFound
		}

		return models.InventoryItem{}, fmt.Errorf("locking inventory item by variant:%w", err)
	}

	return item, nil
}
