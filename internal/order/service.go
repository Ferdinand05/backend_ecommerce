package order

import (
	"context"

	"ferdinand/ecommerce/database"
	cartitems "ferdinand/ecommerce/internal/cart_items"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"ferdinand/ecommerce/internal/shipping"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	CreateOrder(
		ctx context.Context,
		userID uuid.UUID,
		req CreateOrderRequest,
	) (models.Order, error)

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

	FindStatusHistories(
		ctx context.Context,
		orderID uuid.UUID,
	) ([]models.OrderStatusHistory, error)

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
	repository    Repository
	db            *gorm.DB
	cartService   cartitems.Service
	inventoryRepo inventory.InventoryRepository
	movementRepo  inventory.StockMovementRepository
	variantRepo   productvariant.Repository
	shippingSvc   shipping.Service
}

func NewService(
	repository Repository,
	db *gorm.DB,
	cartService cartitems.Service,
	inventoryRepo inventory.InventoryRepository,
	variantRepo productvariant.Repository,
	shippingSvc shipping.Service,
	movementRepo inventory.StockMovementRepository,
) *service {
	return &service{
		repository:    repository,
		db:            db,
		cartService:   cartService,
		inventoryRepo: inventoryRepo,
		movementRepo:  movementRepo,
		variantRepo:   variantRepo,
		shippingSvc:   shippingSvc,
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

func (s *service) FindStatusHistories(
	ctx context.Context,
	orderID uuid.UUID,
) ([]models.OrderStatusHistory, error) {
	return s.repository.FindStatusHistories(ctx, orderID)
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
