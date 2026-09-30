package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailVerificationRepository interface {
	Create(
		ctx context.Context,
		verification models.EmailVerification,
	) error

	FindByTokenHash(
		ctx context.Context,
		tokenHash string,
	) (models.EmailVerification, error)

	MarkVerified(
		ctx context.Context,
		id uuid.UUID,
	) error

	InvalidateUserTokens(
		ctx context.Context,
		userID uuid.UUID,
	) error
}

type emailVerificationRepository struct {
	db *gorm.DB
}

func NewEmailVerificationRepository(
	db *gorm.DB,
) *emailVerificationRepository {
	return &emailVerificationRepository{
		db: db,
	}
}

func (r *emailVerificationRepository) Create(
	ctx context.Context,
	verification models.EmailVerification,
) error {
	if err := r.db.
		WithContext(ctx).
		Create(&verification).
		Error; err != nil {
		return fmt.Errorf("creating email verification: %w", err)
	}

	return nil
}

func (r *emailVerificationRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (models.EmailVerification, error) {
	var verification models.EmailVerification

	err := r.db.
		WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&verification).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.EmailVerification{}, gorm.ErrRecordNotFound
		}

		return models.EmailVerification{},
			fmt.Errorf("finding email verification: %w", err)
	}

	return verification, nil
}

func (r *emailVerificationRepository) MarkVerified(
	ctx context.Context,
	id uuid.UUID,
) error {
	now := time.Now()

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.
		Model(&models.EmailVerification{}).
		Where("id = ?", id).
		Where("verified_at IS NULL").
		Updates(map[string]any{
			"verified_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("marking email verification: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *emailVerificationRepository) InvalidateUserTokens(
	ctx context.Context,
	userID uuid.UUID,
) error {
	now := time.Now()

	return r.db.
		WithContext(ctx).
		Model(&models.EmailVerification{}).
		Where(
			"user_id = ? AND verified_at IS NULL AND expires_at > ?",
			userID,
			now,
		).
		Update("expires_at", now).
		Error
}
