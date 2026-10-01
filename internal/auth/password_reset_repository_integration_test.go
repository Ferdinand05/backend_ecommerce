package auth

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestPasswordResetRepositoryCreateAndFindByTokenHash(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPasswordResetRepository(db)

	u := seedUser(t, db)

	pr := models.PasswordReset{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "reset-hash-1",
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := repo.Create(context.Background(), pr); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByTokenHash(context.Background(), "reset-hash-1")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if found.ID != pr.ID {
		t.Fatalf("FindByTokenHash() ID = %v, want %v", found.ID, pr.ID)
	}
}

func TestPasswordResetRepositoryFindByTokenHashNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPasswordResetRepository(db)

	_, err := repo.FindByTokenHash(context.Background(), "missing-hash")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("FindByTokenHash() error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}

func TestPasswordResetRepositoryMarkUsed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPasswordResetRepository(db)

	u := seedUser(t, db)

	pr := models.PasswordReset{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "reset-hash-2",
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := repo.Create(context.Background(), pr); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.MarkUsed(context.Background(), pr.ID); err != nil {
		t.Fatalf("MarkUsed() error = %v", err)
	}

	found, err := repo.FindByTokenHash(context.Background(), "reset-hash-2")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if found.UsedAt == nil {
		t.Fatal("MarkUsed() did not set UsedAt")
	}

	if err := repo.MarkUsed(context.Background(), pr.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("MarkUsed() second call error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}

func TestPasswordResetRepositoryMarkUsedNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPasswordResetRepository(db)

	err := repo.MarkUsed(context.Background(), uuid.New())
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("MarkUsed() error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}

func TestPasswordResetRepositoryInvalidateUserTokens(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPasswordResetRepository(db)

	u := seedUser(t, db)
	otherU := seedUser(t, db)

	active := models.PasswordReset{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "active-reset-hash",
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	now := time.Now()
	used := models.PasswordReset{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "used-reset-hash",
		ExpiresAt: time.Now().Add(30 * time.Minute),
		UsedAt:    &now,
	}

	other := models.PasswordReset{
		ID:        uuid.New(),
		UserID:    otherU.ID,
		TokenHash: "other-reset-hash",
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	for _, pr := range []models.PasswordReset{active, used, other} {
		if err := repo.Create(context.Background(), pr); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if err := repo.InvalidateUserTokens(context.Background(), u.ID); err != nil {
		t.Fatalf("InvalidateUserTokens() error = %v", err)
	}

	activeFound, err := repo.FindByTokenHash(context.Background(), "active-reset-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if activeFound.ExpiresAt.After(time.Now()) {
		t.Fatal("InvalidateUserTokens() did not expire active token")
	}

	usedFound, err := repo.FindByTokenHash(context.Background(), "used-reset-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if usedFound.ExpiresAt.Before(time.Now()) {
		t.Fatal("InvalidateUserTokens() expired an already-used token")
	}

	otherFound, err := repo.FindByTokenHash(context.Background(), "other-reset-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if otherFound.ExpiresAt.Before(time.Now()) {
		t.Fatal("InvalidateUserTokens() expired another user's token")
	}
}
