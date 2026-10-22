package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Shipment struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	Provider      string  `gorm:"type:varchar(30);not null;default:BITESHIP"`
	ProviderOrderID *string `gorm:"type:varchar(100);uniqueIndex"`

	CourierCompany *string `gorm:"type:varchar(100)"`
	CourierType    *string `gorm:"type:varchar(100)"`
	TrackingNumber *string `gorm:"type:varchar(100);index"`
	TrackingURL    *string `gorm:"type:text"`

	Status         string  `gorm:"type:varchar(30);not null;default:PENDING;index"`
	ProviderStatus *string `gorm:"type:varchar(100)"`

	ShippingCost decimal.Decimal `gorm:"type:numeric(15,2);not null;default:0"`
	Currency     string          `gorm:"type:varchar(3);not null;default:IDR"`

	ShippedAt   *time.Time
	DeliveredAt *time.Time
	CancelledAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	Order Order `gorm:"foreignKey:OrderID;references:ID"`
}