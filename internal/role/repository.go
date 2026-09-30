package role

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindByName(ctx context.Context, name string) (models.Role, error)
	FindByID(ctx context.Context, roleID uuid.UUID) (models.Role, error)
	FindAll(ctx context.Context) ([]models.Role, error)
	Update(ctx context.Context, role models.Role) (models.Role, error)
	Delete(ctx context.Context, roleID uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *repository {
	return &repository{
		db: db,
	}
}

func (r *repository) FindByName(ctx context.Context, name string) (models.Role, error) {

	var role models.Role

	err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Role{}, ErrorRoleNotFound
		}
		return models.Role{}, fmt.Errorf("finding role by name: %w", err)
	}

	return role, nil
}

func (r *repository) FindByID(ctx context.Context, roleID uuid.UUID) (models.Role, error) {

	var role models.Role

	err := r.db.WithContext(ctx).First(&role, roleID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Role{}, ErrorRoleNotFound
		}
		return models.Role{}, fmt.Errorf("finding role: %w", err)
	}

	return role, nil
}

func (r *repository) FindAll(ctx context.Context) ([]models.Role, error) {

	var roles []models.Role

	err := r.db.WithContext(ctx).Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("finding roles: %w", err)
	}

	return roles, nil
}

func (r *repository) Update(ctx context.Context, role models.Role) (models.Role, error) {

	err := r.db.WithContext(ctx).Model(&models.Role{}).Where("id = ?", role.ID).Update("name", role.Name).Error
	if err != nil {
		return models.Role{}, fmt.Errorf("updating role: %w", err)
	}

	return r.FindByID(ctx, role.ID)
}

func (r *repository) Delete(ctx context.Context, roleID uuid.UUID) error {

	err := r.db.WithContext(ctx).Delete(&models.Role{}, roleID).Error
	if err != nil {
		return fmt.Errorf("deleting role: %w", err)
	}

	return nil
}
