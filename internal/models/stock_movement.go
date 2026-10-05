package models

import (
	"time"

	"github.com/google/uuid"
)

type StockMovement struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	InventoryItemID uuid.UUID `gorm:"type:uuid;not null;index"`

	Type          string     `gorm:"type:varchar(30);not null"`
	Quantity      int        `gorm:"not null"`
	Note          *string    `gorm:"type:text"`
	ReferenceType *string    `gorm:"type:varchar(50)"`
	ReferenceID   *uuid.UUID `gorm:"type:uuid"`

	CreatedAt time.Time

	InventoryItem InventoryItem `gorm:"foreignKey:InventoryItemID"`
}