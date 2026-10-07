package models

import (
	"time"

	"github.com/google/uuid"
)

type CartItem struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index"`
	ProductVariantID uuid.UUID `gorm:"type:uuid;not null;index"`
	Quantity         int       `gorm:"not null"`

	CreatedAt time.Time
	UpdatedAt time.Time

	User           User           `gorm:"foreignKey:UserID"`
	ProductVariant ProductVariant `gorm:"foreignKey:ProductVariantID"`
}
