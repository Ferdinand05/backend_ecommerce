package product

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateProductRequest struct {
	CategoryID  uuid.UUID `json:"category_id" binding:"required,uuid"`
	Name        string    `json:"name" binding:"required,max=150"`
	Description *string   `json:"description"`
}

type UpdateProductRequest struct {
	CategoryID  uuid.UUID `json:"category_id" binding:"required,uuid"`
	Name        string    `json:"name" binding:"required,max=150"`
	Description *string   `json:"description"`
}

type CategoryResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type ProductVariantStockResponse struct {
	Quantity         int `json:"quantity"`
	ReservedQuantity int `json:"reserved_quantity"`
}

type ProductVariantResponse struct {
	ID        uuid.UUID                    `json:"id"`
	SKU       string                       `json:"sku"`
	Name      string                       `json:"name"`
	Price     decimal.Decimal              `json:"price"`
	IsActive  bool                         `json:"is_active"`
	Inventory *ProductVariantStockResponse `json:"inventory,omitempty"`
}

type ProductImageResponse struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	Alt       *string   `json:"alt"`
	SortOrder int       `json:"sort_order"`
}

type ProductResponse struct {
	ID          uuid.UUID              `json:"id"`
	CategoryID  uuid.UUID              `json:"category_id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Description *string                `json:"description"`
	IsActive    bool                   `json:"is_active"`
	Category    *CategoryResponse      `json:"category"`
	Images      []ProductImageResponse `json:"images"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ProductDetailResponse struct {
	ProductResponse
	Variants []ProductVariantResponse `json:"variants"`
}
