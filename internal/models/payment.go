package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const PaymentProviderXendit = "XENDIT"

type Payment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	OrderID uuid.UUID `gorm:"type:uuid;not null;index"`
	Order   Order     `gorm:"foreignKey:OrderID"`

	Provider string `gorm:"type:varchar(30);not null"`

	// ID unik yang dibuat oleh aplikasi dan dikirim ke Xendit.
	ExternalID string `gorm:"type:varchar(100);not null;uniqueIndex"`

	// ID invoice yang diberikan Xendit setelah invoice berhasil dibuat.
	ProviderInvoiceID *string `gorm:"type:varchar(100);uniqueIndex"`

	Status string `gorm:"type:varchar(20);not null;default:'PENDING';index"`

	Amount   decimal.Decimal `gorm:"type:numeric(19,4);not null"`
	Currency string          `gorm:"type:varchar(3);not null;default:'IDR'"`

	CheckoutURL *string    `gorm:"type:text"`
	ExpiresAt   *time.Time `gorm:"type:timestamptz"`
	PaidAt      *time.Time `gorm:"type:timestamptz"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
