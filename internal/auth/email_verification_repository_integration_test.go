package auth

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	err := godotenv.Load("../../.env.test")
	if err != nil {
		t.Fatalf("failed to load .env.test: %v", err)
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin test transaction: %v", tx.Error)
	}

	t.Cleanup(func() {
		tx.Rollback()

		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	})

	return tx
}

const customerRoleID = "00000000-0000-0000-0000-000000000001"

func TestEmailVerificationRepositoryCreateAndFindByTokenHash(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)

	u := seedUser(t, db)

	verification := models.EmailVerification{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "token-hash-1",
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := repo.Create(context.Background(), verification); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByTokenHash(context.Background(), "token-hash-1")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if found.ID != verification.ID {
		t.Fatalf("FindByTokenHash() ID = %v, want %v", found.ID, verification.ID)
	}
}

func TestEmailVerificationRepositoryFindByTokenHashNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)

	_, err := repo.FindByTokenHash(context.Background(), "missing-hash")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("FindByTokenHash() error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}

func TestEmailVerificationRepositoryMarkVerified(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)

	u := seedUser(t, db)

	verification := models.EmailVerification{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "token-hash-2",
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := repo.Create(context.Background(), verification); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.MarkVerified(context.Background(), verification.ID); err != nil {
		t.Fatalf("MarkVerified() error = %v", err)
	}

	found, err := repo.FindByTokenHash(context.Background(), "token-hash-2")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if found.VerifiedAt == nil {
		t.Fatal("MarkVerified() did not set VerifiedAt")
	}
}

func TestEmailVerificationRepositoryMarkVerifiedNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)

	err := repo.MarkVerified(context.Background(), uuid.New())
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("MarkVerified() error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}

func TestEmailVerificationRepositoryInvalidateUserTokens(t *testing.T) {
	db := setupTestDB(t)
	repo := NewEmailVerificationRepository(db)

	u := seedUser(t, db)
	userID := u.ID
	active := models.EmailVerification{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: "active-hash",
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	now := time.Now()
	verified := models.EmailVerification{
		ID:         uuid.New(),
		UserID:     userID,
		TokenHash:  "verified-hash",
		ExpiresAt:  time.Now().Add(30 * time.Minute),
		VerifiedAt: &now,
	}

	for _, v := range []models.EmailVerification{active, verified} {
		if err := repo.Create(context.Background(), v); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if err := repo.InvalidateUserTokens(context.Background(), userID); err != nil {
		t.Fatalf("InvalidateUserTokens() error = %v", err)
	}

	activeFound, err := repo.FindByTokenHash(context.Background(), "active-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if !activeFound.ExpiresAt.Before(time.Now().Add(time.Second)) {
		t.Fatal("InvalidateUserTokens() did not expire active token")
	}

	verifiedFound, err := repo.FindByTokenHash(context.Background(), "verified-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if verifiedFound.ExpiresAt.Before(time.Now()) {
		t.Fatal("InvalidateUserTokens() expired already-verified token")
	}
}
