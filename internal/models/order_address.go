package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderAddress struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`

	RecipientName string `gorm:"type:varchar(150);not null"`
	Phone         string `gorm:"type:varchar(30);not null"`

	Address    string `gorm:"type:text;not null"`
	City       string `gorm:"type:varchar(100);not null"`
	Province   string `gorm:"type:varchar(100);not null"`
	PostalCode string `gorm:"type:varchar(20);not null"`

	Latitude  *decimal.Decimal `gorm:"type:numeric(10,7)"`
	Longitude *decimal.Decimal `gorm:"type:numeric(10,7)"`

	BiteshipAreaID   *string `gorm:"type:varchar(100)"`
	BiteshipAreaName *string `gorm:"type:varchar(255)"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
