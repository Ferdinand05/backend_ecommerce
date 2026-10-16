package checkout

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PreviewResponse struct {
	Items    []PreviewItemResponse `json:"items"`
	Subtotal decimal.Decimal       `json:"subtotal"`
}

type PreviewItemResponse struct {
	ProductVariantID uuid.UUID       `json:"product_variant_id"`
	ProductName      string          `json:"product_name"`
	VariantName      string          `json:"variant_name"`
	SKU              string          `json:"sku"`
	Quantity         int             `json:"quantity"`
	UnitPrice        decimal.Decimal `json:"unit_price"`
	Subtotal         decimal.Decimal `json:"subtotal"`
	WeightGrams      int             `json:"weight_grams"`
	AvailableStock   int             `json:"available_stock"`
}
