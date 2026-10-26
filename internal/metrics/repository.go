package metrics

import (
	"context"
	"time"

	"ferdinand/ecommerce/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Repository interface {
	Summary(ctx context.Context, since time.Time) (SummaryResponse, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

func (r *repository) Summary(ctx context.Context, since time.Time) (SummaryResponse, error) {
	var result SummaryResponse
	var revenue decimal.NullDecimal
	if err := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("status IN ?", []string{"PAID", "PROCESSING", "SHIPPED", "DELIVERED"}).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&revenue).Error; err != nil {
		return SummaryResponse{}, err
	}
	if revenue.Valid {
		result.TotalRevenue = revenue.Decimal
	}

	if err := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("status IN ?", []string{"PENDING", "PAID", "PROCESSING", "SHIPPED"}).
		Count(&result.ActiveOrders).Error; err != nil {
		return SummaryResponse{}, err
	}

	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("created_at >= ?", since).
		Count(&result.NewUsers).Error; err != nil {
		return SummaryResponse{}, err
	}

	if err := r.db.WithContext(ctx).Model(&models.InventoryItem{}).
		Where("quantity - reserved_quantity <= ?", 10).
		Count(&result.LowStockCount).Error; err != nil {
		return SummaryResponse{}, err
	}

	return result, nil
}
