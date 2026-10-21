package inventory

import (
	"context"
	"errors"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StockMovementRepository interface {
	Create(ctx context.Context, movement models.StockMovement) error

	FindByID(ctx context.Context, id uuid.UUID) (models.StockMovement, error)

	FindAllByInventoryID(ctx context.Context, inventoryID uuid.UUID) ([]models.StockMovement, error)

	// FindAll: list global untuk admin, paging + filter type opsional.
	FindAll(ctx context.Context, movementType *string, page, pageSize int) ([]models.StockMovement, int64, error)
}

type stockMovementRepository struct {
	db *gorm.DB
}

func NewStockMovementRepository(db *gorm.DB) *stockMovementRepository {
	return &stockMovementRepository{
		db: db,
	}
}

func (r *stockMovementRepository) Create(ctx context.Context, movement models.StockMovement) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(&movement).Error; err != nil {
		return fmt.Errorf("creating stock movement:%w", err)
	}

	return nil
}

func (r *stockMovementRepository) FindByID(ctx context.Context, id uuid.UUID) (models.StockMovement, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var movement models.StockMovement

	err := db.First(&movement, id).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.StockMovement{}, ErrorStockMovementNotFound
		}

		return models.StockMovement{}, fmt.Errorf("finding stock movement:%w", err)
	}

	return movement, nil
}

func (r *stockMovementRepository) FindAllByInventoryID(ctx context.Context, inventoryID uuid.UUID) ([]models.StockMovement, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var movements []models.StockMovement

	err := db.
		Where("inventory_item_id = ?", inventoryID).
		Order("created_at ASC").
		Find(&movements).
		Error

	if err != nil {
		return nil, fmt.Errorf("finding stock movements:%w", err)
	}

	return movements, nil
}

func (r *stockMovementRepository) FindAll(
	ctx context.Context,
	movementType *string,
	page, pageSize int,
) ([]models.StockMovement, int64, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx).Model(&models.StockMovement{})

	if movementType != nil {
		db = db.Where("type = ?", *movementType)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting stock movements: %w", err)
	}

	var movements []models.StockMovement

	err := db.
		Preload("InventoryItem.ProductVariant").
		Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&movements).
		Error

	if err != nil {
		return nil, 0, fmt.Errorf("finding stock movements: %w", err)
	}

	return movements, total, nil
}
