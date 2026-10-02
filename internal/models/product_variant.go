package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ProductVariant struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey"`
	ProductID uuid.UUID       `gorm:"type:uuid;not null;index"`
	SKU       string          `gorm:"type:varchar(100);not null;uniqueIndex"`
	Name      string          `gorm:"type:varchar(150);not null"`
	Price     decimal.Decimal `gorm:"type:numeric(19,4);not null"`
	IsActive  bool            `gorm:"not null;default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Product Product `gorm:"foreignKey:ProductID"`
}