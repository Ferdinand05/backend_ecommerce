package auth

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/user"
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
