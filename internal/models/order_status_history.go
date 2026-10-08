package models

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatusHistory struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID uuid.UUID `gorm:"type:uuid;not null;index"`

	Status string  `gorm:"type:varchar(30);not null"`
	Note   *string `gorm:"type:text"`

	CreatedAt time.Time

	Order Order `gorm:"foreignKey:OrderID"`
}
