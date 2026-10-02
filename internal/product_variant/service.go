package productvariant

import (
	"context"

	"github.com/google/uuid"
)

type Service interface {
	Create(
		ctx context.Context,
		productID uuid.UUID,
		req CreateProductVariantRequest,
	) (ProductVariantResponse, error)

	FindAllByProductID(
		ctx context.Context,
		productID uuid.UUID,
	) ([]ProductVariantResponse, error)

	FindByID(
		ctx context.Context,
		productID uuid.UUID,
		id uuid.UUID,
	) (ProductVariantResponse, error)

	Update(
		ctx context.Context,
		productID uuid.UUID,
		id uuid.UUID,
		req UpdateProductVariantRequest,
	) (ProductVariantResponse, error)

	Delete(
		ctx context.Context,
		productID uuid.UUID,
		id uuid.UUID,
	) error
}
