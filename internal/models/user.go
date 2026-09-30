package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID          uuid.UUID `gorm:"type:uuid;not null"`
	Email           string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash    string    `gorm:"type:text;not null"`
	FirstName       string    `gorm:"type:varchar(100);not null"`
	LastName        *string   `gorm:"type:varchar(100)"`
	Phone           *string   `gorm:"type:varchar(30)"`
	Status          string    `gorm:"type:varchar(30);not null;default:'active'"`
	EmailVerifiedAt *time.Time
	LastLoginAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	Role Role `gorm:"foreignKey:RoleID"`
}
