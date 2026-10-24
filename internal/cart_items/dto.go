package cartitems

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AddCartItemRequest struct {
	ProductVariantID uuid.UUID `json:"product_variant_id" binding:"required"`
}

type CartItemResponse struct {
	ID               uuid.UUID `json:"id"`
	ProductVariantID uuid.UUID `json:"product_variant_id"`
	Quantity         int       `json:"quantity"`

	SKU         string                  `json:"sku"`
	Name        string                  `json:"name"`
	Price       decimal.Decimal         `json:"price"`
	ProductID   uuid.UUID               `json:"product_id"`
	ProductName string                  `json:"product_name"`
	Images      []CartItemImageResponse `json:"images"`
}

type CartItemImageResponse struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	Alt       *string   `json:"alt"`
	SortOrder int       `json:"sort_order"`
}

type CartResponse struct {
	Items []CartItemResponse `json:"items"`
}
