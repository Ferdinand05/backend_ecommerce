package order

import (
	"context"

	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	FindMyOrders(
		ctx context.Context,
		userID uuid.UUID,
		page int,
		pageSize int,
	) (OrderListResult, error)

	// FindMyOrder menolak order milik user lain dengan ErrorOrderNotFound
	// agar keberadaan order tidak bocor.
	FindMyOrder(
		ctx context.Context,
		userID uuid.UUID,
		orderID uuid.UUID,
	) (models.Order, error)

	FindByID(
		ctx context.Context,
		orderID uuid.UUID,
	) (models.Order, error)

	FindAll(
		ctx context.Context,
		filter OrderFilter,
	) (OrderListResult, error)

	UpdateStatus(
		ctx context.Context,
		orderID uuid.UUID,
		status OrderStatus,
		note *string,
	) (models.Order, error)
}

type service struct {
	repository Repository
	db         *gorm.DB
}

func NewService(repository Repository, db *gorm.DB) *service {
	return &service{
		repository: repository,
		db:         db,
	}
}

func (s *service) FindMyOrders(ctx context.Context, userID uuid.UUID, page int, pageSize int) (OrderListResult, error) {
	return s.repository.FindAllByUserID(ctx, userID, page, pageSize)
}

func (s *service) FindMyOrder(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (models.Order, error) {

	order, err := s.repository.FindByID(ctx, orderID)
	if err != nil {
		return models.Order{}, err
	}

	if order.UserID != userID {
		return models.Order{}, ErrorOrderNotFound
	}

	return order, nil
}

func (s *service) FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error) {
	return s.repository.FindByID(ctx, orderID)
}

func (s *service) FindAll(ctx context.Context, filter OrderFilter) (OrderListResult, error) {
	return s.repository.FindAll(ctx, filter)
}

// UpdateStatus atomic: kunci baris order, validasi transisi, update status,
// lalu insert history dalam satu transaksi.
func (s *service) UpdateStatus(ctx context.Context, orderID uuid.UUID, status OrderStatus, note *string) (models.Order, error) {

	if !status.IsValid() {
		return models.Order{}, ErrorInvalidOrderStatus
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		txCtx := database.InjectTx(ctx, tx)

		current, err := s.repository.FindByIDForUpdate(txCtx, orderID)
		if err != nil {
			return err
		}

		if !CanTransition(OrderStatus(current.Status), status) {
			return ErrorInvalidStatusTransition
		}

		if err := s.repository.UpdateStatus(txCtx, orderID, string(status)); err != nil {
			return err
		}

		return s.repository.CreateStatusHistory(txCtx, &models.OrderStatusHistory{
			ID:      uuid.New(),
			OrderID: orderID,
			Status:  string(status),
			Note:    note,
		})
	})
	if err != nil {
		return models.Order{}, err
	}

	return s.repository.FindByID(ctx, orderID)
}
