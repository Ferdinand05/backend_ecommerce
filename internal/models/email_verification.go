package models

import (
	"time"

	"github.com/google/uuid"
)

type EmailVerification struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash  string    `gorm:"type:text;not null;index"`
	ExpiresAt  time.Time `gorm:"not null;index"`
	VerifiedAt *time.Time
	CreatedAt  time.Time

	User User `gorm:"foreignKey:UserID"`
}
