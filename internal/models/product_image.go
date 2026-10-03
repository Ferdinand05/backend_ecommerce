package models

import (
	"time"

	"github.com/google/uuid"
)

type ProductImage struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ProductID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	ProductVariantID *uuid.UUID `gorm:"type:uuid;index"`

	StorageKey string  `gorm:"type:text;not null"`
	Alt        *string `gorm:"type:varchar(255)"`
	SortOrder  int     `gorm:"not null;default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Product        Product         `gorm:"foreignKey:ProductID"`
	ProductVariant *ProductVariant `gorm:"foreignKey:ProductVariantID"`
}
