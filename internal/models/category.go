package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Slug        string    `gorm:"type:varchar(120);uniqueIndex;not null"`
	Description *string   `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
