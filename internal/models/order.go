package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Order struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`
	OrderNumber string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Status      string    `gorm:"type:varchar(30);not null"`

	Subtotal       decimal.Decimal `gorm:"type:numeric(19,4);not null"`
	ShippingCost   decimal.Decimal `gorm:"type:numeric(19,4);not null;default:0"`
	DiscountAmount decimal.Decimal `gorm:"type:numeric(19,4);not null;default:0"`
	TotalAmount    decimal.Decimal `gorm:"type:numeric(19,4);not null"`

	CreatedAt time.Time
	UpdatedAt time.Time

	User            User                 `gorm:"foreignKey:UserID"`
	Items           []OrderItem          `gorm:"foreignKey:OrderID"`
	Address         *OrderAddress         `gorm:"foreignKey:OrderID"`
	StatusHistories []OrderStatusHistory `gorm:"foreignKey:OrderID"`
}