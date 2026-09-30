package user

import (
	"context"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
)

type Service interface {
	FindAll(ctx context.Context) ([]models.User, error)
	FindByID(ctx context.Context, userID uuid.UUID) (models.User, error)
	FindByEmail(ctx context.Context, email string) (models.User, error)
	Create(ctx context.Context, user models.User) (models.User, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{
		repo: repo,
	}
}

func (s *service) FindAll(ctx context.Context) ([]models.User, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) FindByID(ctx context.Context, userID uuid.UUID) (models.User, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *service) FindByEmail(ctx context.Context, email string) (models.User, error) {
	return s.repo.FindByEmail(ctx, email)
}

func (s *service) Create(ctx context.Context, user models.User) (models.User, error) {
	return s.repo.Create(ctx, user)
}
