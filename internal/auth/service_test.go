package auth

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/user"
	"ferdinand/ecommerce/utils/crypto"
	userjwt "ferdinand/ecommerce/utils/jwt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fakeUserRepo struct {
	findByEmailRes models.User
	findByEmailErr error
	findByIDRes    models.User
	findByIDErr    error
	createRes      models.User
	createErr      error
	createCalled   bool
	createdUser    models.User
}

func (f *fakeUserRepo) FindAll(ctx context.Context) ([]models.User, error) {
	return nil, nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, userID uuid.UUID) (models.User, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (models.User, error) {
	return f.findByEmailRes, f.findByEmailErr
}

func (f *fakeUserRepo) Create(ctx context.Context, u models.User) (models.User, error) {
	f.createCalled = true
	f.createdUser = u
	if f.createErr != nil {
		return models.User{}, f.createErr
	}
	if f.createRes.ID != uuid.Nil {
		return f.createRes, nil
	}
	return u, nil
}

func (f *fakeUserRepo) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func (f *fakeUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}

type fakeRoleRepo struct {
	findByNameRes models.Role
	findByNameErr error
}

func (f *fakeRoleRepo) FindByName(ctx context.Context, name string) (models.Role, error) {
	return f.findByNameRes, f.findByNameErr
}

func (f *fakeRoleRepo) FindByID(ctx context.Context, roleID uuid.UUID) (models.Role, error) {
	return models.Role{}, nil
}

func (f *fakeRoleRepo) FindAll(ctx context.Context) ([]models.Role, error) {
	return nil, nil
}

func (f *fakeRoleRepo) Update(ctx context.Context, role models.Role) (models.Role, error) {
	return role, nil
}

func (f *fakeRoleRepo) Delete(ctx context.Context, roleID uuid.UUID) error {
	return nil
}

type fakeJWT struct {
	token     string
	genErr    error
	genCalled bool
	genUserID uuid.UUID
	genEmail  string
	genRole   string
}

func (f *fakeJWT) GenerateToken(userID uuid.UUID, email string, role string) (string, error) {
	f.genCalled = true
	f.genUserID = userID
	f.genEmail = email
	f.genRole = role
	return f.token, f.genErr
}

func (f *fakeJWT) ValidateToken(tokenString string) (*userjwt.Claims, error) {
	return &userjwt.Claims{}, nil
}

type fakeSender struct {
	sentTo      string
	sentToken   string
	sendErr     error
	resetSentTo string
	resetErr    error
}

func (f *fakeSender) SendVerificationEmail(ctx context.Context, to string, token string) error {
	f.sentTo = to
	f.sentToken = token
	return f.sendErr
}

func (f *fakeSender) SendPasswordResetEmail(ctx context.Context, to string, token string) error {
	f.resetSentTo = to
	return f.resetErr
}

type fakeEmailVerifRepo struct {
	createCalled        bool
	createErr           error
	createdVerification models.EmailVerification
	invalidateCalledID  uuid.UUID
	invalidateErr       error
}

func (f *fakeEmailVerifRepo) Create(ctx context.Context, verification models.EmailVerification) error {
	f.createCalled = true
	f.createdVerification = verification
	return f.createErr
}

func (f *fakeEmailVerifRepo) FindByTokenHash(ctx context.Context, tokenHash string) (models.EmailVerification, error) {
	return models.EmailVerification{}, gorm.ErrRecordNotFound
}

func (f *fakeEmailVerifRepo) MarkVerified(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (f *fakeEmailVerifRepo) InvalidateUserTokens(ctx context.Context, userID uuid.UUID) error {
	f.invalidateCalledID = userID
	return f.invalidateErr
}

type fakeRefreshTokenRepo struct {
	findByHashRes    models.RefreshToken
	findByHashErr    error
	revokedID        uuid.UUID
	createdToken     models.RefreshToken
	revokeErr        error
	createErr        error
	revokedAllUserID uuid.UUID
	revokeAllErr     error
}

func (f *fakeRefreshTokenRepo) Create(ctx context.Context, refreshToken models.RefreshToken) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.createdToken = refreshToken
	return nil
}

func (f *fakeRefreshTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (models.RefreshToken, error) {
	return f.findByHashRes, f.findByHashErr
}

func (f *fakeRefreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	f.revokedID = id
	return nil
}

func (f *fakeRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	f.revokedAllUserID = userID
	return f.revokeAllErr
}

type fakePasswordResetRepo struct {
	findByHashRes models.PasswordReset
	findByHashErr error
	createCalled  bool
	createdReset  models.PasswordReset
	createErr     error
	markUsedID    uuid.UUID
	markUsedErr   error
	invalidateID  uuid.UUID
	invalidateErr error
}

func (f *fakePasswordResetRepo) Create(ctx context.Context, reset models.PasswordReset) error {
	f.createCalled = true
	f.createdReset = reset
	return f.createErr
}

func (f *fakePasswordResetRepo) FindByTokenHash(ctx context.Context, tokenHash string) (models.PasswordReset, error) {
	return f.findByHashRes, f.findByHashErr
}

func (f *fakePasswordResetRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	f.markUsedID = id
	return f.markUsedErr
}

func (f *fakePasswordResetRepo) InvalidateUserTokens(ctx context.Context, userID uuid.UUID) error {
	f.invalidateID = userID
	return f.invalidateErr
}

func newTestService(ur *fakeUserRepo, rr *fakeRoleRepo, jwt *fakeJWT, sender *fakeSender, evr *fakeEmailVerifRepo, rtr *fakeRefreshTokenRepo) *service {
	return &service{
		userRepo:         ur,
		roleRepo:         rr,
		jwtSvc:           jwt,
		emailSender:      sender,
		emailVerifRepo:   evr,
		refreshTokenRepo: rtr,
	}
}

func TestRegister(t *testing.T) {
	customerRole := models.Role{
		ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name: "customer",
	}

	roleErr := errors.New("role not found")

	tests := []struct {
		name          string
		req           RegisterRequest
		userRepo      *fakeUserRepo
		roleRepo      *fakeRoleRepo
		sender        *fakeSender
		verifRepo     *fakeEmailVerifRepo
		wantErr       error
		wantHashMatch bool
		wantRoleID    uuid.UUID
	}{
		{
			name: "success",
			req: RegisterRequest{
				Email:     "user@example.com",
				Password:  "password123",
				FirstName: "Test",
			},
			userRepo:      &fakeUserRepo{findByEmailErr: user.ErrorUserNotFound},
			roleRepo:      &fakeRoleRepo{findByNameRes: customerRole},
			sender:        &fakeSender{},
			verifRepo:     &fakeEmailVerifRepo{},
			wantHashMatch: true,
			wantRoleID:    customerRole.ID,
		},
		{
			name: "normalizes email",
			req: RegisterRequest{
				Email:     "  User@Example.COM ",
				Password:  "password123",
				FirstName: "Test",
			},
			userRepo:      &fakeUserRepo{findByEmailErr: user.ErrorUserNotFound},
			roleRepo:      &fakeRoleRepo{findByNameRes: customerRole},
			sender:        &fakeSender{},
			verifRepo:     &fakeEmailVerifRepo{},
			wantHashMatch: true,
			wantRoleID:    customerRole.ID,
		},
		{
			name: "email exists",
			req: RegisterRequest{
				Email:     "user@example.com",
				Password:  "password123",
				FirstName: "Test",
			},
			userRepo: &fakeUserRepo{},
			roleRepo: &fakeRoleRepo{findByNameRes: customerRole},
			sender:   &fakeSender{},
			wantErr:  ErrorEmailExists,
		},
		{
			name: "role missing",
			req: RegisterRequest{
				Email:     "user@example.com",
				Password:  "password123",
				FirstName: "Test",
			},
			userRepo: &fakeUserRepo{findByEmailErr: user.ErrorUserNotFound},
			roleRepo: &fakeRoleRepo{findByNameErr: roleErr},
			sender:   &fakeSender{},
			wantErr:  roleErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(tt.userRepo, tt.roleRepo, &fakeJWT{}, tt.sender, tt.verifRepo, &fakeRefreshTokenRepo{})

			resp, err := svc.Register(context.Background(), tt.req)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("Register() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Register() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Register() error = %v", err)
			}

			if !tt.userRepo.createCalled {
				t.Fatal("Register() did not call userRepo.Create")
			}

			created := tt.userRepo.createdUser
			if created.RoleID != tt.wantRoleID {
				t.Fatalf("created RoleID = %v, want %v", created.RoleID, tt.wantRoleID)
			}

			if tt.wantHashMatch && !crypto.CheckPassword(tt.req.Password, created.PasswordHash) {
				t.Fatal("created PasswordHash does not match request password")
			}

			if created.PasswordHash == tt.req.Password {
				t.Fatal("Register() stored plaintext password")
			}

			if !tt.verifRepo.createCalled {
				t.Fatal("Register() did not create email verification")
			}

			if tt.sender.sentTo != created.Email {
				t.Fatalf("sender sentTo = %q, want %q", tt.sender.sentTo, created.Email)
			}

			if resp.Email != created.Email {
				t.Fatalf("Register() Email = %q, want %q", resp.Email, created.Email)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	hash, err := crypto.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	id := uuid.New()
	existingUser := models.User{
		ID:           id,
		Email:        "user@example.com",
		PasswordHash: hash,
		Role:         models.Role{Name: "customer"},
	}

	signErr := errors.New("sign failed")

	tests := []struct {
		name     string
		req      LoginRequest
		userRepo *fakeUserRepo
		jwt      *fakeJWT
		wantErr  error
	}{
		{
			name: "unknown email",
			req: LoginRequest{
				Email:    "missing@example.com",
				Password: "password123",
			},
			userRepo: &fakeUserRepo{findByEmailErr: user.ErrorUserNotFound},
			jwt:      &fakeJWT{},
			wantErr:  ErrorInvalidCredentials,
		},
		{
			name: "wrong password",
			req: LoginRequest{
				Email:    "user@example.com",
				Password: "wrong-password",
			},
			userRepo: &fakeUserRepo{findByEmailRes: existingUser},
			jwt:      &fakeJWT{},
			wantErr:  ErrorInvalidCredentials,
		},
		{
			name: "jwt error",
			req: LoginRequest{
				Email:    "user@example.com",
				Password: "password123",
			},
			userRepo: &fakeUserRepo{findByEmailRes: existingUser},
			jwt:      &fakeJWT{genErr: signErr},
			wantErr:  signErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(tt.userRepo, &fakeRoleRepo{}, tt.jwt, &fakeSender{}, &fakeEmailVerifRepo{}, &fakeRefreshTokenRepo{})

			_, err := svc.Login(context.Background(), tt.req)
			if err == nil {
				t.Fatal("Login() error = nil, want error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	now := time.Now()
	dbErr := errors.New("db down")

	tests := []struct {
		name     string
		userRepo *fakeUserRepo
		rtr      *fakeRefreshTokenRepo
		wantErr  error
	}{
		{
			name:     "not found",
			userRepo: &fakeUserRepo{},
			rtr:      &fakeRefreshTokenRepo{findByHashErr: gorm.ErrRecordNotFound},
			wantErr:  ErrorInvalidRefreshToken,
		},
		{
			name:     "revoked",
			userRepo: &fakeUserRepo{},
			rtr: &fakeRefreshTokenRepo{findByHashRes: models.RefreshToken{
				RevokedAt: &now,
			}},
			wantErr: ErrorInvalidRefreshToken,
		},
		{
			name:     "expired",
			userRepo: &fakeUserRepo{},
			rtr: &fakeRefreshTokenRepo{findByHashRes: models.RefreshToken{
				ExpiresAt: now.Add(-time.Minute),
			}},
			wantErr: ErrorRefreshTokenExpired,
		},
		{
			name:     "user not found",
			userRepo: &fakeUserRepo{findByIDErr: user.ErrorUserNotFound},
			rtr: &fakeRefreshTokenRepo{findByHashRes: models.RefreshToken{
				ExpiresAt: now.Add(time.Hour),
			}},
			wantErr: ErrorInvalidRefreshToken,
		},
		{
			name:     "user repo error",
			userRepo: &fakeUserRepo{findByIDErr: dbErr},
			rtr: &fakeRefreshTokenRepo{findByHashRes: models.RefreshToken{
				ExpiresAt: now.Add(time.Hour),
			}},
			wantErr: dbErr,
		},
		{
			name:     "find error not record",
			userRepo: &fakeUserRepo{},
			rtr:      &fakeRefreshTokenRepo{findByHashErr: dbErr},
			wantErr:  dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(tt.userRepo, &fakeRoleRepo{}, &fakeJWT{}, &fakeSender{}, &fakeEmailVerifRepo{}, tt.rtr)

			_, err := svc.Refresh(context.Background(), "raw-token")
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("Refresh() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Refresh() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Refresh() error = %v", err)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	now := time.Now()
	dbErr := errors.New("db down")
	id := uuid.New()

	tests := []struct {
		name          string
		rtr           *fakeRefreshTokenRepo
		wantErr       error
		wantRevokedID uuid.UUID
	}{
		{
			name: "token not found is idempotent",
			rtr:  &fakeRefreshTokenRepo{findByHashErr: gorm.ErrRecordNotFound},
		},
		{
			name: "already revoked is idempotent",
			rtr: &fakeRefreshTokenRepo{findByHashRes: models.RefreshToken{
				ID:        id,
				RevokedAt: &now,
			}},
		},
		{
			name:    "find error",
			rtr:     &fakeRefreshTokenRepo{findByHashErr: dbErr},
			wantErr: dbErr,
		},
		{
			name: "revoke error",
			rtr: &fakeRefreshTokenRepo{
				findByHashRes: models.RefreshToken{ID: id},
				revokeErr:     dbErr,
			},
			wantErr: dbErr,
		},
		{
			name: "success revokes token",
			rtr: &fakeRefreshTokenRepo{
				findByHashRes: models.RefreshToken{ID: id},
			},
			wantRevokedID: id,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(&fakeUserRepo{}, &fakeRoleRepo{}, &fakeJWT{}, &fakeSender{}, &fakeEmailVerifRepo{}, tt.rtr)

			err := svc.Logout(context.Background(), "raw-token")
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("Logout() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Logout() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Logout() error = %v", err)
			}

			if tt.rtr.revokedID != tt.wantRevokedID {
				t.Fatalf("Logout() revokedID = %v, want %v", tt.rtr.revokedID, tt.wantRevokedID)
			}
		})
	}
}

func TestResendVerification(t *testing.T) {
	dbErr := errors.New("db down")
	verifiedAt := time.Now()

	tests := []struct {
		name     string
		userRepo *fakeUserRepo
		wantErr  error
	}{
		{
			name:     "unknown email is silent",
			userRepo: &fakeUserRepo{findByEmailErr: user.ErrorUserNotFound},
		},
		{
			name: "already verified is a no-op",
			userRepo: &fakeUserRepo{findByEmailRes: models.User{
				EmailVerifiedAt: &verifiedAt,
			}},
		},
		{
			name:     "find error",
			userRepo: &fakeUserRepo{findByEmailErr: dbErr},
			wantErr:  dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifRepo := &fakeEmailVerifRepo{}
			sender := &fakeSender{}
			svc := newTestService(tt.userRepo, &fakeRoleRepo{}, &fakeJWT{}, sender, verifRepo, &fakeRefreshTokenRepo{})

			err := svc.ResendVerification(context.Background(), "user@example.com")
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("ResendVerification() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ResendVerification() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ResendVerification() error = %v", err)
			}

			if verifRepo.createCalled {
				t.Fatal("ResendVerification() created a token unexpectedly")
			}

			if sender.sentTo != "" {
				t.Fatal("ResendVerification() sent an email unexpectedly")
			}
		})
	}
}

func TestForgotPassword(t *testing.T) {
	dbErr := errors.New("db down")

	tests := []struct {
		name     string
		userRepo *fakeUserRepo
		pr       *fakePasswordResetRepo
		wantErr  error
	}{
		{
			name:     "unknown email is silent",
			userRepo: &fakeUserRepo{findByEmailErr: user.ErrorUserNotFound},
			pr:       &fakePasswordResetRepo{},
		},
		{
			name:     "find error",
			userRepo: &fakeUserRepo{findByEmailErr: dbErr},
			pr:       &fakePasswordResetRepo{},
			wantErr:  dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &fakeSender{}
			svc := &service{userRepo: tt.userRepo, passwordResetRepo: tt.pr, emailSender: sender}

			err := svc.ForgotPassword(context.Background(), "user@example.com")
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("ForgotPassword() error = nil, want error")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ForgotPassword() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("ForgotPassword() error = %v", err)
			}

			if tt.pr.createCalled {
				t.Fatal("ForgotPassword() created a token unexpectedly")
			}

			if sender.resetSentTo != "" {
				t.Fatal("ForgotPassword() sent an email unexpectedly")
			}
		})
	}
}

func TestResetPassword(t *testing.T) {
	now := time.Now()
	dbErr := errors.New("db down")
	resetID := uuid.New()

	tests := []struct {
		name    string
		pr      *fakePasswordResetRepo
		wantErr error
	}{
		{
			name:    "invalid token",
			pr:      &fakePasswordResetRepo{findByHashErr: gorm.ErrRecordNotFound},
			wantErr: ErrorInvalidResetToken,
		},
		{
			name: "token used",
			pr: &fakePasswordResetRepo{findByHashRes: models.PasswordReset{
				ID:        resetID,
				UsedAt:    &now,
				ExpiresAt: now.Add(time.Hour),
			}},
			wantErr: ErrorResetTokenUsed,
		},
		{
			name: "token expired",
			pr: &fakePasswordResetRepo{findByHashRes: models.PasswordReset{
				ID:        resetID,
				ExpiresAt: now.Add(-time.Minute),
			}},
			wantErr: ErrorResetTokenExpired,
		},
		{
			name:    "find error",
			pr:      &fakePasswordResetRepo{findByHashErr: dbErr},
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &service{passwordResetRepo: tt.pr}

			err := svc.ResetPassword(context.Background(), "raw-token", "newpassword123")
			if err == nil {
				t.Fatal("ResetPassword() error = nil, want error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResetPassword() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
