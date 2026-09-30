package auth

import (
	"context"
	"errors"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/mail"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/role"
	"ferdinand/ecommerce/internal/user"
	"ferdinand/ecommerce/utils/crypto"
	userjwt "ferdinand/ecommerce/utils/jwt"
	"ferdinand/ecommerce/utils/token"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (user.UserResponse, error)
	Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
	VerifyEmail(
		ctx context.Context,
		rawToken string,
	) error
}

type service struct {
	userRepo       user.Repository
	roleRepo       role.Repository
	jwtSvc         userjwt.JWTManager
	emailSender    mail.Sender
	emailVerifRepo EmailVerificationRepository
	db             *gorm.DB
}

func NewService(userRepo user.Repository, roleRepo role.Repository, jwtSvc userjwt.JWTManager, emailSender mail.Sender, emailVerifRepo EmailVerificationRepository,
	db *gorm.DB) *service {
	return &service{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		jwtSvc:         jwtSvc,
		emailSender:    emailSender,
		emailVerifRepo: emailVerifRepo,
		db:             db,
	}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (user.UserResponse, error) {

	email := strings.ToLower(
		strings.TrimSpace(req.Email),
	)

	_, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return user.UserResponse{}, ErrorEmailExists
	}

	if !errors.Is(err, user.ErrorUserNotFound) {
		return user.UserResponse{}, fmt.Errorf("checking email: %w", err)
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return user.UserResponse{}, err
	}

	customerRole, err := s.roleRepo.FindByName(ctx, "customer")
	if err != nil {
		return user.UserResponse{}, fmt.Errorf("finding customer role: %w", err)
	}

	created, err := s.userRepo.Create(ctx, models.User{
		ID:           uuid.New(),
		RoleID:       customerRole.ID,
		Email:        email,
		PasswordHash: hash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		Status:       "active",
	})
	if err != nil {
		return user.UserResponse{}, err
	}

	rawToken, err := token.Generate()
	if err != nil {
		return user.UserResponse{},
			fmt.Errorf("generating verification token: %w", err)
	}

	tokenHash := token.Hash(rawToken)

	verification := models.EmailVerification{
		ID:        uuid.New(),
		UserID:    created.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := s.emailVerifRepo.Create(
		ctx,
		verification,
	); err != nil {
		return user.UserResponse{},
			fmt.Errorf("creating email verification: %w", err)
	}

	if err := s.emailSender.SendVerificationEmail(
		ctx,
		created.Email,
		rawToken,
	); err != nil {
		return user.UserResponse{},
			fmt.Errorf("sending verification email: %w", err)
	}

	return user.ToUserResponse(created), nil
}

func (s *service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	u, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, user.ErrorUserNotFound) {
			return LoginResponse{}, ErrorInvalidCredentials
		}
		return LoginResponse{}, err
	}

	if !crypto.CheckPassword(req.Password, u.PasswordHash) {
		return LoginResponse{}, ErrorInvalidCredentials
	}

	token, err := s.jwtSvc.GenerateToken(u.ID, u.Email, u.Role.Name)
	if err != nil {
		return LoginResponse{}, err
	}

	return LoginResponse{
		Token: token,
		User:  user.ToUserResponse(u),
	}, nil
}

func (s *service) VerifyEmail(
	ctx context.Context,
	rawToken string,
) error {
	tokenHash := token.Hash(rawToken)

	verification, err :=
		s.emailVerifRepo.FindByTokenHash(
			ctx,
			tokenHash,
		)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrorInvalidVerificationToken
		}

		return err
	}

	if verification.VerifiedAt != nil {
		return ErrorEmailAlreadyVerified
	}

	if time.Now().After(verification.ExpiresAt) {
		return ErrorVerificationTokenExpired
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)

		if err := s.userRepo.MarkEmailVerified(
			txCtx,
			verification.UserID,
		); err != nil {
			return err
		}

		if err := s.emailVerifRepo.MarkVerified(
			txCtx,
			verification.ID,
		); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil

}
