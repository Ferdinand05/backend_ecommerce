package shipment

import (
	"context"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, shipment *models.Shipment) error

	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (*models.Shipment, error)

	FindByOrderID(
		ctx context.Context,
		orderID uuid.UUID,
	) (*models.Shipment, error)

	FindByIDForUpdate(
		ctx context.Context,
		id uuid.UUID,
	) (*models.Shipment, error)

	Update(ctx context.Context, shipment *models.Shipment) error

	FindByTrackingNumber(
		ctx context.Context,
		trackingNumber string,
	) (*models.Shipment, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository{
	return &repository{db:db}
}