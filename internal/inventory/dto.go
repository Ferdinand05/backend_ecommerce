package inventory

import (
	"time"

	"github.com/google/uuid"
)

type InventoryResponse struct {
	ID               uuid.UUID `json:"id"`
	ProductVariantID uuid.UUID `json:"product_variant_id"`
	Quantity         int       `json:"quantity"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type StockMovementResponse struct {
	ID              uuid.UUID  `json:"id"`
	InventoryItemID uuid.UUID  `json:"inventory_item_id"`
	Type            string     `json:"type"`
	Quantity        int        `json:"quantity"`
	Note            *string    `json:"note"`
	ReferenceType   *string    `json:"reference_type"`
	ReferenceID     *uuid.UUID `json:"reference_id"`
	CreatedAt       time.Time  `json:"created_at"`
}

type StockMovementType string

const (
	StockMovementRestock     StockMovementType = "RESTOCK"
	StockMovementSale        StockMovementType = "SALE"
	StockMovementReturn      StockMovementType = "RETURN"
	StockMovementDamage      StockMovementType = "DAMAGE"
	StockMovementAdjustment  StockMovementType = "ADJUSTMENT"
	StockMovementOrderCancel StockMovementType = "ORDER_CANCEL"
)

type CreateStockMovementRequest struct {
	Type          StockMovementType `json:"type" binding:"required"`
	Quantity      int               `json:"quantity" binding:"required"`
	Note          *string           `json:"note"`
	ReferenceType *string           `json:"reference_type"`
	ReferenceID   *uuid.UUID        `json:"reference_id"`
}
