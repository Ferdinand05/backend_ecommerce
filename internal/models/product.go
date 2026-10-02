package models

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	CategoryID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"type:varchar(150);not null"`
	Slug        string    `gorm:"type:varchar(180);not null;uniqueIndex"`
	Description *string   `gorm:"type:text"`
	IsActive    bool      `gorm:"not null;default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Category Category `gorm:"foreignKey:CategoryID"`

	Variants []ProductVariant `gorm:"foreignKey:ProductID"`
}