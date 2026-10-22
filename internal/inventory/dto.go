package inventory

import (
	"time"

	"github.com/google/uuid"
)

type InventoryResponse struct {
	ID               uuid.UUID `json:"id"`
	ProductVariantID uuid.UUID `json:"product_variant_id"`
	Quantity         int       `json:"quantity"`
	ReservedQuantity int       `json:"reserved_quantity"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AdminStockMovementResponse adalah item list global stock movement
// untuk admin: identitas item + sku/variant supaya enak di datatable.
type AdminStockMovementResponse struct {
	StockMovementResponse
	SKU         string `json:"sku"`
	VariantName string `json:"variant_name"`
}

type StockMovementListMeta struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type StockMovementListResponse struct {
	Movements []AdminStockMovementResponse `json:"movements"`
	Meta      StockMovementListMeta        `json:"meta"`
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
	StockMovementReserve     StockMovementType = "RESERVE"
)

type CreateStockMovementRequest struct {
	Type          StockMovementType `json:"type" binding:"required"`
	Quantity      int               `json:"quantity" binding:"required"`
	Note          *string           `json:"note"`
	ReferenceType *string           `json:"reference_type"`
	ReferenceID   *uuid.UUID        `json:"reference_id"`
}
