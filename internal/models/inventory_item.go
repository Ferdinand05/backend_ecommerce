package models

import (
	"time"

	"github.com/google/uuid"
)

type InventoryItem struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	ProductVariantID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Quantity         int       `gorm:"not null;default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time

	ProductVariant ProductVariant `gorm:"foreignKey:ProductVariantID"`
	Movements      []StockMovement `gorm:"foreignKey:InventoryItemID"`
}