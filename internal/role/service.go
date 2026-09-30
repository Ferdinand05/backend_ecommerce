package role

import (
	"context"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
)

type Service interface {
	FindByID(ctx context.Context, roleID uuid.UUID) (models.Role, error)
	FindAll(ctx context.Context) ([]models.Role, error)
	Update(ctx context.Context, role models.Role) (models.Role, error)
	Delete(ctx context.Context, roleID uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{
		repo: repo,
	}
}

func (s *service) FindByID(ctx context.Context, roleID uuid.UUID) (models.Role, error) {
	return s.repo.FindByID(ctx, roleID)
}

func (s *service) FindAll(ctx context.Context) ([]models.Role, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) Update(ctx context.Context, role models.Role) (models.Role, error) {
	return s.repo.Update(ctx, role)
}

func (s *service) Delete(ctx context.Context, roleID uuid.UUID) error {
	return s.repo.Delete(ctx, roleID)
}
