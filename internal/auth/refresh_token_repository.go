package auth

import (
	"context"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, refreshToken models.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (models.RefreshToken, error)
	Revoke(
		ctx context.Context,
		id uuid.UUID,
	) error

	RevokeAllByUserID(
		ctx context.Context,
		userID uuid.UUID,
	) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{
		db: db,
	}
}

func (r *refreshTokenRepository) Create(
	ctx context.Context,
	refreshToken models.RefreshToken,
) error {
	db := database.GetDB(ctx, r.db)

	return db.
		WithContext(ctx).
		Create(&refreshToken).
		Error
}

func (r *refreshTokenRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (models.RefreshToken, error) {
	db := database.GetDB(ctx, r.db)

	var refreshToken models.RefreshToken

	err := db.
		WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&refreshToken).
		Error

	return refreshToken, err
}

func (r *refreshTokenRepository) Revoke(
	ctx context.Context,
	id uuid.UUID,
) error {
	db := database.GetDB(ctx, r.db)

	now := time.Now()

	result := db.
		WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)

	return result.Error
}

func (r *refreshTokenRepository) RevokeAllByUserID(
	ctx context.Context,
	userID uuid.UUID,
) error {
	db := database.GetDB(ctx, r.db)

	now := time.Now()

	return db.
		WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).
		Error
}
