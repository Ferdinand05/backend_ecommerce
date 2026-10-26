package product

import (
	"time"

	"ferdinand/ecommerce/internal/models"

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

// ProductFilter Deskripsi query list produk. Nil pada IsActive berarti
// semua status (dipakai admin), pointer ke true berarti hanya produk aktif.
type ProductFilter struct {
	Q          string
	CategoryID *uuid.UUID
	Sort       string
	Page       int
	PageSize   int
	IsActive   *bool
}

const (
	defaultProductPageSize = 20
	maxProductPageSize     = 100
)

// Normalize mengisi default dan memotong page size ke batas maksimum.
func (f ProductFilter) Normalize() ProductFilter {
	if f.Page < 1 {
		f.Page = 1
	}

	if f.PageSize < 1 {
		f.PageSize = defaultProductPageSize
	}

	if f.PageSize > maxProductPageSize {
		f.PageSize = maxProductPageSize
	}

	return f
}

// ProductListResult hasil list produk beserta metadata pagination.
type ProductListResult struct {
	Products []models.Product
	Total    int64
	Page     int
	PageSize int
}

type ProductListMeta struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
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
	ID       uuid.UUID       `json:"id"`
	SKU      string          `json:"sku"`
	Name     string          `json:"name"`
	Price    decimal.Decimal `json:"price"`
	IsActive bool            `json:"is_active"`
}

type ProductVariantDetailResponse struct {
	ProductVariantResponse
	WeightGrams int                          `json:"weight_grams"`
	Inventory   *ProductVariantStockResponse `json:"inventory,omitempty"`
	Images      []ProductImageResponse       `json:"images"`
}

type ProductImageResponse struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	Alt       *string   `json:"alt"`
	SortOrder int       `json:"sort_order"`
}

type ProductResponse struct {
	ID          uuid.UUID                `json:"id"`
	CategoryID  uuid.UUID                `json:"category_id"`
	Name        string                   `json:"name"`
	Slug        string                   `json:"slug"`
	Description *string                  `json:"description"`
	IsActive    bool                     `json:"is_active"`
	Category    *CategoryResponse        `json:"category"`
	Images      []ProductImageResponse   `json:"images"`
	Variants    []ProductVariantResponse `json:"variants"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

type ProductDetailResponse struct {
	ID          uuid.UUID                      `json:"id"`
	CategoryID  uuid.UUID                      `json:"category_id"`
	Name        string                         `json:"name"`
	Slug        string                         `json:"slug"`
	Description *string                        `json:"description"`
	IsActive    bool                           `json:"is_active"`
	Category    *CategoryResponse              `json:"category"`
	Images      []ProductImageResponse         `json:"images"`
	Variants    []ProductVariantDetailResponse `json:"variants"`
	CreatedAt   time.Time                      `json:"created_at"`
	UpdatedAt   time.Time                      `json:"updated_at"`
}
