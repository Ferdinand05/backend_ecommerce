package auth

import (
	"context"
	"errors"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PasswordResetRepository interface {
	Create(
		ctx context.Context,
		reset models.PasswordReset,
	) error

	FindByTokenHash(
		ctx context.Context,
		tokenHash string,
	) (models.PasswordReset, error)

	MarkUsed(
		ctx context.Context,
		id uuid.UUID,
	) error

	InvalidateUserTokens(
		ctx context.Context,
		userID uuid.UUID,
	) error
}

type passwordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(
	db *gorm.DB,
) *passwordResetRepository {
	return &passwordResetRepository{
		db: db,
	}
}

func (r *passwordResetRepository) Create(
	ctx context.Context,
	reset models.PasswordReset,
) error {
	db := database.GetDB(ctx, r.db)

	if err := db.
		WithContext(ctx).
		Create(&reset).
		Error; err != nil {
		return fmt.Errorf("creating password reset: %w", err)
	}

	return nil
}

func (r *passwordResetRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (models.PasswordReset, error) {
	db := database.GetDB(ctx, r.db)

	var reset models.PasswordReset

	err := db.
		WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&reset).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.PasswordReset{}, gorm.ErrRecordNotFound
		}

		return models.PasswordReset{},
			fmt.Errorf("finding password reset: %w", err)
	}

	return reset, nil
}

func (r *passwordResetRepository) MarkUsed(
	ctx context.Context,
	id uuid.UUID,
) error {
	db := database.GetDB(ctx, r.db)

	now := time.Now()

	result := db.
		WithContext(ctx).
		Model(&models.PasswordReset{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", now)

	if result.Error != nil {
		return fmt.Errorf("marking password reset used: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *passwordResetRepository) InvalidateUserTokens(
	ctx context.Context,
	userID uuid.UUID,
) error {
	db := database.GetDB(ctx, r.db)

	now := time.Now()

	return db.
		WithContext(ctx).
		Model(&models.PasswordReset{}).
		Where(
			"user_id = ? AND used_at IS NULL AND expires_at > ?",
			userID,
			now,
		).
		Update("expires_at", now).
		Error
}
