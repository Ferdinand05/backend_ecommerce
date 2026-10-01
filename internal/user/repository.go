package user

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

type Repository interface {
	FindAll(ctx context.Context) ([]models.User, error)
	FindByID(ctx context.Context, userID uuid.UUID) (models.User, error)
	FindByEmail(ctx context.Context, email string) (models.User, error)
	Create(ctx context.Context, user models.User) (models.User, error)

	MarkEmailVerified(
		ctx context.Context,
		userID uuid.UUID,
	) error

	UpdatePassword(
		ctx context.Context,
		userID uuid.UUID,
		passwordHash string,
	) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{
		db: db,
	}
}

func (r *repository) FindAll(ctx context.Context) ([]models.User, error) {

	var users []models.User

	err := r.db.WithContext(ctx).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("finding users:%w", err)
	}

	return users, nil
}

func (r *repository) FindByID(ctx context.Context, userID uuid.UUID) (models.User, error) {

	var user models.User

	err := r.db.WithContext(ctx).
		Preload("Role").
		First(&user, userID).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, ErrorUserNotFound
		}
		return models.User{}, fmt.Errorf("finding user:%w", err)
	}

	return user, nil
}

func (r *repository) FindByEmail(ctx context.Context, email string) (models.User, error) {

	var user models.User

	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("email = ?", email).
		First(&user).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, ErrorUserNotFound
		}
		return models.User{}, fmt.Errorf("finding user by email:%w", err)
	}

	return user, nil
}

func (r *repository) Create(ctx context.Context, user models.User) (models.User, error) {

	err := r.db.WithContext(ctx).Create(&user).Error
	if err != nil {
		return models.User{}, fmt.Errorf("creating user:%w", err)
	}

	return user, nil
}

func (r *repository) MarkEmailVerified(
	ctx context.Context,
	userID uuid.UUID,
) error {
	now := time.Now()

	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("email_verified_at", now)

	if result.Error != nil {
		return fmt.Errorf("marking user email verified: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorUserNotFound
	}

	return nil
}

func (r *repository) UpdatePassword(
	ctx context.Context,
	userID uuid.UUID,
	passwordHash string,
) error {
	db := database.GetDB(ctx, r.db).WithContext(ctx)

	result := db.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash)

	if result.Error != nil {
		return fmt.Errorf("updating user password: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrorUserNotFound
	}

	return nil
}
