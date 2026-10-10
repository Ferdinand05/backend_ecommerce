package productvariant

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateProductVariantRequest struct {
	SKU         string          `json:"sku" binding:"required,max=100"`
	Name        string          `json:"name" binding:"required,max=150"`
	Price       decimal.Decimal `json:"price" binding:"required"`
	WeightGrams int             `json:"weight_grams" binding:"required,min=1,max=100000"`
}

type UpdateProductVariantRequest struct {
	SKU         string          `json:"sku" binding:"required,max=100"`
	Name        string          `json:"name" binding:"required,max=150"`
	Price       decimal.Decimal `json:"price" binding:"required"`
	WeightGrams int             `json:"weight_grams" binding:"required,min=1,max=100000"`
}

type ProductVariantResponse struct {
	ID          uuid.UUID       `json:"id"`
	ProductID   uuid.UUID       `json:"product_id"`
	SKU         string          `json:"sku"`
	Name        string          `json:"name"`
	Price       decimal.Decimal `json:"price"`
	IsActive    bool            `json:"is_active"`
	WeightGrams int             `json:"weight_grams"`
}
