package productvariant

import (
	"context"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, variant models.ProductVariant) error
	FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error)
	FindByID(ctx context.Context, id uuid.UUID) (models.ProductVariant, error)
	FindBySKU(ctx context.Context, sku string) (models.ProductVariant, error)
	ExistsBySKUExceptID(ctx context.Context, sku string, id uuid.UUID) (bool, error)
	Update(ctx context.Context, id uuid.UUID, variant models.ProductVariant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}
