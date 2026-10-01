package auth

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/user"
	"ferdinand/ecommerce/utils/crypto"
	"ferdinand/ecommerce/utils/token"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func seedUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()

	u := models.User{
		ID:           uuid.New(),
		RoleID:       uuid.MustParse(customerRoleID),
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "hashed-password",
		FirstName:    "Test",
		Status:       "active",
	}

	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	return u
}

func seedVerification(t *testing.T, db *gorm.DB, userID uuid.UUID, rawToken string, expiresAt time.Time, verifiedAt *time.Time) {
	t.Helper()

	v := models.EmailVerification{
		ID:         uuid.New(),
		UserID:     userID,
		TokenHash:  token.Hash(rawToken),
		ExpiresAt:  expiresAt,
		VerifiedAt: verifiedAt,
	}

	if err := db.Create(&v).Error; err != nil {
		t.Fatalf("failed to seed verification: %v", err)
	}
}

func newVerifyService(db *gorm.DB) *service {
	return &service{
		userRepo:       user.NewRepository(db),
		roleRepo:       nil,
		jwtSvc:         nil,
		emailSender:    nil,
		emailVerifRepo: NewEmailVerificationRepository(db),
		db:             db,
	}
}

func TestVerifyEmail(t *testing.T) {
	tests := []struct {
		name       string
		rawToken   string
		seed       func(t *testing.T, db *gorm.DB, userID uuid.UUID)
		wantErr    error
		wantUserID uuid.UUID
	}{
		{
			name:     "invalid token",
			rawToken: "does-not-exist",
			seed:     func(t *testing.T, db *gorm.DB, userID uuid.UUID) {},
			wantErr:  ErrorInvalidVerificationToken,
		},
		{
			name:     "expired token",
			rawToken: "expired-token",
			seed: func(t *testing.T, db *gorm.DB, userID uuid.UUID) {
				seedVerification(t, db, userID, "expired-token", time.Now().Add(-time.Minute), nil)
			},
			wantErr: ErrorVerificationTokenExpired,
		},
		{
			name:     "already verified",
			rawToken: "verified-token",
			seed: func(t *testing.T, db *gorm.DB, userID uuid.UUID) {
				now := time.Now()
				seedVerification(t, db, userID, "verified-token", time.Now().Add(30*time.Minute), &now)
			},
			wantErr: ErrorEmailAlreadyVerified,
		},
		{
			name:     "success",
			rawToken: "valid-token",
			seed: func(t *testing.T, db *gorm.DB, userID uuid.UUID) {
				seedVerification(t, db, userID, "valid-token", time.Now().Add(30*time.Minute), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)

			u := seedUser(t, db)
			tt.seed(t, db, u.ID)

			svc := newVerifyService(db)

			err := svc.VerifyEmail(context.Background(), tt.rawToken)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("VerifyEmail() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("VerifyEmail() error = %v", err)
			}

			var foundUser models.User
			if err := db.First(&foundUser, u.ID).Error; err != nil {
				t.Fatalf("failed to find user: %v", err)
			}

			if foundUser.EmailVerifiedAt == nil {
				t.Fatal("VerifyEmail() did not set user EmailVerifiedAt")
			}

			var foundVerif models.EmailVerification
			if err := db.Where("token_hash = ?", token.Hash(tt.rawToken)).First(&foundVerif).Error; err != nil {
				t.Fatalf("failed to find verification: %v", err)
			}

			if foundVerif.VerifiedAt == nil {
				t.Fatal("VerifyEmail() did not set verification VerifiedAt")
			}
		})
	}
}

func TestLoginCreatesRefreshToken(t *testing.T) {
	db := setupTestDB(t)

	hash, err := crypto.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	u := seedUser(t, db)

	if err := db.Model(&models.User{}).Where("id = ?", u.ID).Update("password_hash", hash).Error; err != nil {
		t.Fatalf("failed to set password hash: %v", err)
	}

	fakeJ := &fakeJWT{token: "access-token"}
	svc := &service{
		userRepo:         user.NewRepository(db),
		jwtSvc:           fakeJ,
		refreshTokenRepo: NewRefreshTokenRepository(db),
		db:               db,
	}

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    u.Email,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if resp.AccessToken != "access-token" {
		t.Fatalf("AccessToken = %q, want %q", resp.AccessToken, "access-token")
	}

	if resp.RefreshToken == "" {
		t.Fatal("RefreshToken is empty")
	}

	if resp.ExpiresIn != 900 {
		t.Fatalf("ExpiresIn = %d, want 900", resp.ExpiresIn)
	}

	if fakeJ.genRole != "customer" {
		t.Fatalf("GenerateToken role = %q, want customer", fakeJ.genRole)
	}

	var stored models.RefreshToken
	if err := db.Where("token_hash = ?", token.Hash(resp.RefreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}

	if stored.RevokedAt != nil {
		t.Fatal("fresh refresh token marked revoked")
	}
}

func TestRefreshRotation(t *testing.T) {
	db := setupTestDB(t)

	u := seedUser(t, db)

	rawOld := "old-refresh-token"
	old := models.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: token.Hash(rawOld),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	if err := db.Create(&old).Error; err != nil {
		t.Fatalf("failed to seed refresh token: %v", err)
	}

	fakeJ := &fakeJWT{token: "new-access-token"}
	svc := &service{
		userRepo:         user.NewRepository(db),
		jwtSvc:           fakeJ,
		refreshTokenRepo: NewRefreshTokenRepository(db),
		db:               db,
	}

	resp, err := svc.Refresh(context.Background(), rawOld)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	if resp.AccessToken != "new-access-token" {
		t.Fatalf("AccessToken = %q, want new-access-token", resp.AccessToken)
	}

	if resp.RefreshToken == "" || resp.RefreshToken == rawOld {
		t.Fatal("RefreshToken not rotated")
	}

	if fakeJ.genRole != "customer" {
		t.Fatalf("GenerateToken role = %q, want customer", fakeJ.genRole)
	}

	var oldStored models.RefreshToken
	if err := db.Where("token_hash = ?", token.Hash(rawOld)).First(&oldStored).Error; err != nil {
		t.Fatalf("old refresh token not found: %v", err)
	}

	if oldStored.RevokedAt == nil {
		t.Fatal("old refresh token not revoked")
	}

	var newStored models.RefreshToken
	if err := db.Where("token_hash = ?", token.Hash(resp.RefreshToken)).First(&newStored).Error; err != nil {
		t.Fatalf("new refresh token not stored: %v", err)
	}

	if newStored.RevokedAt != nil {
		t.Fatal("new refresh token marked revoked")
	}

	if newStored.ID == old.ID {
		t.Fatal("new refresh token reused old ID")
	}
}

func TestLogoutRevokesToken(t *testing.T) {
	db := setupTestDB(t)

	u := seedUser(t, db)

	raw := "logout-refresh-token"
	rt := models.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: token.Hash(raw),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	if err := db.Create(&rt).Error; err != nil {
		t.Fatalf("failed to seed refresh token: %v", err)
	}

	svc := &service{
		refreshTokenRepo: NewRefreshTokenRepository(db),
		db:               db,
	}

	if err := svc.Logout(context.Background(), raw); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	var stored models.RefreshToken
	if err := db.Where("token_hash = ?", token.Hash(raw)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token row missing after logout: %v", err)
	}

	if stored.RevokedAt == nil {
		t.Fatal("Logout() did not set RevokedAt")
	}

	if err := svc.Logout(context.Background(), raw); err != nil {
		t.Fatalf("Logout() second call error = %v, want nil (idempotent)", err)
	}

	if err := svc.Logout(context.Background(), "unknown-token"); err != nil {
		t.Fatalf("Logout() unknown token error = %v, want nil (idempotent)", err)
	}
}
