package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderItem struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	OrderID          uuid.UUID `gorm:"type:uuid;not null;index"`
	ProductVariantID uuid.UUID `gorm:"type:uuid;not null;index"`

	ProductName string `gorm:"type:varchar(150);not null"`
	VariantName string `gorm:"type:varchar(150);not null"`
	SKU         string `gorm:"type:varchar(100);not null"`

	Price    decimal.Decimal `gorm:"type:numeric(19,4);not null"`
	Quantity int             `gorm:"not null"`
	Subtotal decimal.Decimal `gorm:"type:numeric(19,4);not null"`

	CreatedAt time.Time

	Order          Order          `gorm:"foreignKey:OrderID"`
	ProductVariant ProductVariant `gorm:"foreignKey:ProductVariantID"`
}