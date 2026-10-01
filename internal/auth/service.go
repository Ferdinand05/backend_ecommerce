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

	Refresh(
		ctx context.Context,
		rawRefreshToken string,
	) (LoginResponse, error)

	Logout(
		ctx context.Context,
		rawRefreshToken string,
	) error

	ResendVerification(
		ctx context.Context,
		email string,
	) error
}

type service struct {
	userRepo         user.Repository
	roleRepo         role.Repository
	jwtSvc           userjwt.JWTManager
	emailSender      mail.Sender
	emailVerifRepo   EmailVerificationRepository
	db               *gorm.DB
	refreshTokenRepo RefreshTokenRepository
}

func NewService(userRepo user.Repository, roleRepo role.Repository, jwtSvc userjwt.JWTManager, emailSender mail.Sender, emailVerifRepo EmailVerificationRepository,
	refreshTokenRepo RefreshTokenRepository, db *gorm.DB) *service {
	return &service{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		jwtSvc:           jwtSvc,
		emailSender:      emailSender,
		emailVerifRepo:   emailVerifRepo,
		refreshTokenRepo: refreshTokenRepo,
		db:               db,
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

	// 15 minutes token
	accessToken, err := s.jwtSvc.GenerateToken(
		u.ID,
		u.Email,
		u.Role.Name,
	)
	if err != nil {
		return LoginResponse{}, err
	}

	rawRefreshToken, err := token.Generate()
	if err != nil {
		return LoginResponse{},
			fmt.Errorf("generating refresh token: %w", err)
	}

	// refresh token 30 days
	refreshTokenHash := token.Hash(rawRefreshToken)

	refreshToken := models.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {

		txCtx := database.InjectTx(ctx, tx)

		return s.refreshTokenRepo.Create(
			txCtx,
			refreshToken,
		)

	})

	if err != nil {
		return LoginResponse{},
			fmt.Errorf("creating refresh token: %w", err)
	}

	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64((15 * time.Minute).Seconds()),
		User:         user.ToUserResponse(u),
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

func (s *service) Refresh(
	ctx context.Context,
	rawRefreshToken string,
) (LoginResponse, error) {
	tokenHash := token.Hash(rawRefreshToken)

	rt, err := s.refreshTokenRepo.FindByTokenHash(
		ctx,
		tokenHash,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LoginResponse{}, ErrorInvalidRefreshToken
		}

		return LoginResponse{},
			fmt.Errorf("finding refresh token: %w", err)
	}

	if rt.RevokedAt != nil {
		return LoginResponse{}, ErrorInvalidRefreshToken
	}

	if time.Now().After(rt.ExpiresAt) {
		return LoginResponse{}, ErrorRefreshTokenExpired
	}

	u, err := s.userRepo.FindByID(ctx, rt.UserID)
	if err != nil {
		if errors.Is(err, user.ErrorUserNotFound) {
			return LoginResponse{}, ErrorInvalidRefreshToken
		}

		return LoginResponse{}, err
	}

	accessToken, err := s.jwtSvc.GenerateToken(
		u.ID,
		u.Email,
		u.Role.Name,
	)
	if err != nil {
		return LoginResponse{}, err
	}

	newRawToken, err := token.Generate()
	if err != nil {
		return LoginResponse{},
			fmt.Errorf("generating refresh token: %w", err)
	}

	newRefreshToken := models.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: token.Hash(newRawToken),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)

		if err := s.refreshTokenRepo.Revoke(
			txCtx,
			rt.ID,
		); err != nil {
			return err
		}

		return s.refreshTokenRepo.Create(
			txCtx,
			newRefreshToken,
		)
	}); err != nil {
		return LoginResponse{},
			fmt.Errorf("rotating refresh token: %w", err)
	}

	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRawToken,
		ExpiresIn:    int64((15 * time.Minute).Seconds()),
		User:         user.ToUserResponse(u),
	}, nil
}

func (s *service) Logout(
	ctx context.Context,
	rawRefreshToken string,
) error {
	tokenHash := token.Hash(rawRefreshToken)

	rt, err := s.refreshTokenRepo.FindByTokenHash(
		ctx,
		tokenHash,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return fmt.Errorf("finding refresh token: %w", err)
	}

	if rt.RevokedAt != nil {
		return nil
	}

	if err := s.refreshTokenRepo.Revoke(
		ctx,
		rt.ID,
	); err != nil {
		return fmt.Errorf("revoking refresh token: %w", err)
	}

	return nil
}

func (s *service) ResendVerification(
	ctx context.Context,
	email string,
) error {
	email = strings.ToLower(
		strings.TrimSpace(email),
	)

	u, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrorUserNotFound) {
			return nil
		}

		return fmt.Errorf("finding user: %w", err)
	}

	if u.EmailVerifiedAt != nil {
		return nil
	}

	rawToken, err := token.Generate()
	if err != nil {
		return fmt.Errorf("generating verification token: %w", err)
	}

	verification := models.EmailVerification{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: token.Hash(rawToken),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)

		if err := s.emailVerifRepo.InvalidateUserTokens(
			txCtx,
			u.ID,
		); err != nil {
			return err
		}

		return s.emailVerifRepo.Create(
			txCtx,
			verification,
		)
	}); err != nil {
		return fmt.Errorf("creating email verification: %w", err)
	}

	if err := s.emailSender.SendVerificationEmail(
		ctx,
		u.Email,
		rawToken,
	); err != nil {
		return fmt.Errorf("sending verification email: %w", err)
	}

	return nil
}
