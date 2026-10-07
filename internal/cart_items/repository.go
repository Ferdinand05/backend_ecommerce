package cartitems

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

type Repository interface {
	// tulis
	AddQuantity(ctx context.Context, userID uuid.UUID, variantID uuid.UUID, quantity int) (models.CartItem, error)
	IncreaseQuantity(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, quantity int) (models.CartItem, error)
	DecreaseQuantity(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, quantity int) (models.CartItem, error)

	// baca
	FindByID(ctx context.Context, id uuid.UUID) (models.CartItem, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error)

	// hapus
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

// ponytail: atomicity dijamin single-statement upsert (ON CONFLICT), tanpa
// transaction eksplisit; kalau nanti butuh >1 statement, bungkus tx di sini.
func (r *repository) AddQuantity(ctx context.Context, userID uuid.UUID, variantID uuid.UUID, quantity int) (models.CartItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "product_variant_id"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity": gorm.Expr("cart_items.quantity + EXCLUDED.quantity"),
		}),
	}).Create(&models.CartItem{
		ID:               uuid.New(),
		UserID:           userID,
		ProductVariantID: variantID,
		Quantity:         quantity,
	}).Error

	if err != nil {
		return models.CartItem{}, fmt.Errorf("adding cart item quantity:%w", err)
	}

	return r.findByUserAndVariant(ctx, userID, variantID)
}

func (r *repository) IncreaseQuantity(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, quantity int) (models.CartItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Model(&models.CartItem{}).
		Where("id = ? AND user_id = ?", itemID, userID).
		Update("quantity", gorm.Expr("quantity + ?", quantity))

	if result.Error != nil {
		return models.CartItem{}, fmt.Errorf("increasing cart item quantity:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return models.CartItem{}, ErrorCartItemNotFound
	}

	return r.findByIDPreloaded(ctx, itemID)
}

func (r *repository) DecreaseQuantity(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, quantity int) (models.CartItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Model(&models.CartItem{}).
		Where("id = ? AND user_id = ?", itemID, userID).
		Update("quantity", gorm.Expr("quantity - ?", quantity))

	if result.Error != nil {
		return models.CartItem{}, fmt.Errorf("decreasing cart item quantity:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return models.CartItem{}, ErrorCartItemNotFound
	}

	return r.findByIDPreloaded(ctx, itemID)
}

func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (models.CartItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var item models.CartItem

	err := db.First(&item, id).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CartItem{}, ErrorCartItemNotFound
		}

		return models.CartItem{}, fmt.Errorf("finding cart item:%w", err)
	}

	return item, nil
}

func (r *repository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var items []models.CartItem

	err := db.
		Preload("ProductVariant.Product").
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&items).
		Error

	if err != nil {
		return nil, fmt.Errorf("finding cart items by user:%w", err)
	}

	return items, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Delete(&models.CartItem{}, id)

	if result.Error != nil {
		return fmt.Errorf("deleting cart item:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorCartItemNotFound
	}

	return nil
}

func (r *repository) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Where("user_id = ?", userID).Delete(&models.CartItem{})

	if result.Error != nil {
		return fmt.Errorf("deleting cart items by user:%w", result.Error)
	}

	return nil
}

func (r *repository) findByIDPreloaded(ctx context.Context, id uuid.UUID) (models.CartItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var item models.CartItem

	err := db.
		Preload("ProductVariant.Product").
		First(&item, id).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CartItem{}, ErrorCartItemNotFound
		}

		return models.CartItem{}, fmt.Errorf("finding cart item with relations:%w", err)
	}

	return item, nil
}

func (r *repository) findByUserAndVariant(ctx context.Context, userID uuid.UUID, variantID uuid.UUID) (models.CartItem, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var item models.CartItem

	err := db.
		Preload("ProductVariant.Product").
		Where("user_id = ? AND product_variant_id = ?", userID, variantID).
		First(&item).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CartItem{}, ErrorCartItemNotFound
		}

		return models.CartItem{}, fmt.Errorf("finding cart item by user and variant:%w", err)
	}

	return item, nil
}
