package order

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderResponse struct {
	ID             uuid.UUID       `json:"id"`
	OrderNumber    string          `json:"order_number"`
	Status         string          `json:"status"`
	Subtotal       decimal.Decimal `json:"subtotal"`
	ShippingCost   decimal.Decimal `json:"shipping_cost"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	TotalAmount    decimal.Decimal `json:"total_amount"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type OrderItemResponse struct {
	ID               uuid.UUID `json:"id"`
	ProductVariantID uuid.UUID `json:"product_variant_id"`

	ProductName string `json:"product_name"`
	VariantName string `json:"variant_name"`
	SKU         string `json:"sku"`

	Price    decimal.Decimal `json:"price"`
	Quantity int             `json:"quantity"`
	Subtotal decimal.Decimal `json:"subtotal"`
}

type OrderAddressResponse struct {
	ID             uuid.UUID        `json:"id"`
	RecipientName  string           `json:"recipient_name"`
	Phone          string           `json:"phone"`
	Address        string           `json:"address"`
	City           string           `json:"city"`
	Province       string           `json:"province"`
	PostalCode     string           `json:"postal_code"`
	Latitude       *decimal.Decimal `json:"latitude"`
	Longitude      *decimal.Decimal `json:"longitude"`
	BiteshipAreaID *string          `json:"biteship_area_id"`
}

type OrderStatusHistoryResponse struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	Note      *string   `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateStatusRequest struct {
	Status string  `json:"status" binding:"required"`
	Note   *string `json:"note"`
}

type OrderListMeta struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type OrderDetailResponse struct {
	ID             uuid.UUID       `json:"id"`
	OrderNumber    string          `json:"order_number"`
	Status         string          `json:"status"`
	Subtotal       decimal.Decimal `json:"subtotal"`
	ShippingCost   decimal.Decimal `json:"shipping_cost"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	TotalAmount    decimal.Decimal `json:"total_amount"`

	Items           []OrderItemResponse          `json:"items"`
	Address         OrderAddressResponse         `json:"address"`
	StatusHistories []OrderStatusHistoryResponse `json:"status_histories"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
