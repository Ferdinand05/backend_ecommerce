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
	UpdateProfile(
		ctx context.Context,
		userID uuid.UUID,
		req UpdateProfileRequest,
	) (models.User, error)
	UpdateStatus(
		ctx context.Context,
		userID uuid.UUID,
		status string,
	) (models.User, error)
	UpdateRole(
		ctx context.Context,
		userID uuid.UUID,
		roleID uuid.UUID,
	) (models.User, error)
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

func (s *service) UpdateProfile(
	ctx context.Context,
	userID uuid.UUID,
	req UpdateProfileRequest,
) (models.User, error) {
	if err := s.repo.UpdateProfile(
		ctx,
		userID,
		req.FirstName,
		req.LastName,
		req.Phone,
	); err != nil {
		return models.User{}, err
	}

	return s.repo.FindByID(ctx, userID)
}

func (s *service) UpdateStatus(
	ctx context.Context,
	userID uuid.UUID,
	status string,
) (models.User, error) {
	if status != "active" && status != "blocked" {
		return models.User{}, ErrorInvalidUserStatus
	}

	if err := s.repo.UpdateStatus(ctx, userID, status); err != nil {
		return models.User{}, err
	}

	return s.repo.FindByID(ctx, userID)
}

func (s *service) UpdateRole(
	ctx context.Context,
	userID uuid.UUID,
	roleID uuid.UUID,
) (models.User, error) {
	exists, err := s.repo.RoleExists(ctx, roleID)
	if err != nil {
		return models.User{}, err
	}

	if !exists {
		return models.User{}, ErrorRoleNotFound
	}

	if err := s.repo.UpdateRole(ctx, userID, roleID); err != nil {
		return models.User{}, err
	}

	return s.repo.FindByID(ctx, userID)
}
