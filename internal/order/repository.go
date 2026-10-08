package order

import (
	"context"
	"errors"
	"fmt"

	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error)

	FindByIDForUpdate(ctx context.Context, orderID uuid.UUID) (models.Order, error)

	FindAllByUserID(
		ctx context.Context,
		userID uuid.UUID,
		page int,
		pageSize int,
	) (OrderListResult, error)

	FindAll(ctx context.Context, filter OrderFilter) (OrderListResult, error)

	Create(ctx context.Context, order *models.Order) error

	CreateItem(ctx context.Context, item *models.OrderItem) error

	CreateAddress(ctx context.Context, address *models.OrderAddress) error

	CreateStatusHistory(ctx context.Context, history *models.OrderStatusHistory) error

	UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

func (r *repository) FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var order models.Order

	err := db.
		Preload("Items").
		Preload("Address").
		Preload("StatusHistories").
		First(&order, orderID).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Order{}, ErrorOrderNotFound
		}

		return models.Order{}, fmt.Errorf("finding order:%w", err)
	}

	return order, nil
}

func (r *repository) FindByIDForUpdate(ctx context.Context, orderID uuid.UUID) (models.Order, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	var order models.Order

	err := db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&order, orderID).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Order{}, ErrorOrderNotFound
		}

		return models.Order{}, fmt.Errorf("finding order for update:%w", err)
	}

	return order, nil
}

func (r *repository) FindAllByUserID(
	ctx context.Context,
	userID uuid.UUID,
	page int,
	pageSize int,
) (OrderListResult, error) {

	filter := OrderFilter{
		UserID:   &userID,
		Page:     page,
		PageSize: pageSize,
	}

	return r.FindAll(ctx, filter)
}

func (r *repository) FindAll(ctx context.Context, filter OrderFilter) (OrderListResult, error) {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	// applyFilters membangun WHERE dari field non-nil. Dipanggil dua kali
	// (COUNT dan SELECT) agar tidak ada rantai statement gorm yang terpakai ulang.
	applyFilters := func(q *gorm.DB) *gorm.DB {

		if filter.UserID != nil {
			q = q.Where("user_id = ?", *filter.UserID)
		}

		if filter.Status != nil {
			q = q.Where("status = ?", string(*filter.Status))
		}

		if filter.OrderNumber != nil {
			q = q.Where("order_number ILIKE ?", *filter.OrderNumber+"%")
		}

		if filter.StartDate != nil {
			q = q.Where("created_at >= ?", *filter.StartDate)
		}

		// EndDate eksklusif: handler yang menggeser +1 hari agar inklusif.
		if filter.EndDate != nil {
			q = q.Where("created_at < ?", *filter.EndDate)
		}

		return q
	}

	var total int64

	if err := applyFilters(db.Model(&models.Order{})).Count(&total).Error; err != nil {
		return OrderListResult{}, fmt.Errorf("counting orders:%w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize

	var orders []models.Order

	err := applyFilters(db.Model(&models.Order{})).
		Order("created_at DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&orders).Error
	if err != nil {
		return OrderListResult{}, fmt.Errorf("finding orders:%w", err)
	}

	return OrderListResult{
		Orders:   orders,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

func (r *repository) Create(ctx context.Context, order *models.Order) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(order).Error; err != nil {
		return fmt.Errorf("creating order:%w", err)
	}

	return nil
}

func (r *repository) CreateItem(ctx context.Context, item *models.OrderItem) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(item).Error; err != nil {
		return fmt.Errorf("creating order item:%w", err)
	}

	return nil
}

func (r *repository) CreateAddress(ctx context.Context, address *models.OrderAddress) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(address).Error; err != nil {
		return fmt.Errorf("creating order address:%w", err)
	}

	return nil
}

func (r *repository) CreateStatusHistory(ctx context.Context, history *models.OrderStatusHistory) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	if err := db.Create(history).Error; err != nil {
		return fmt.Errorf("creating order status history:%w", err)
	}

	return nil
}

func (r *repository) UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error {

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.Model(&models.Order{}).
		Where("id = ?", orderID).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("updating order status:%w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorOrderNotFound
	}

	return nil
}
