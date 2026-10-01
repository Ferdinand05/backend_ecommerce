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

func TestRefreshTokenRepositoryCreateAndFindByTokenHash(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)

	u := seedUser(t, db)

	rt := models.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "refresh-hash-1",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	if err := repo.Create(context.Background(), rt); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByTokenHash(context.Background(), "refresh-hash-1")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if found.ID != rt.ID {
		t.Fatalf("FindByTokenHash() ID = %v, want %v", found.ID, rt.ID)
	}
}

func TestRefreshTokenRepositoryFindByTokenHashNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)

	_, err := repo.FindByTokenHash(context.Background(), "missing-hash")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("FindByTokenHash() error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}

func TestRefreshTokenRepositoryRevoke(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)

	u := seedUser(t, db)

	rt := models.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: "refresh-hash-2",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	if err := repo.Create(context.Background(), rt); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Revoke(context.Background(), rt.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	found, err := repo.FindByTokenHash(context.Background(), "refresh-hash-2")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if found.RevokedAt == nil {
		t.Fatal("Revoke() did not set RevokedAt")
	}
}

func TestRefreshTokenRepositoryRevokeAllByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)

	u := seedUser(t, db)
	otherU := seedUser(t, db)

	userTokens := []models.RefreshToken{
		{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: "user-hash-1",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		},
		{
			ID:        uuid.New(),
			UserID:    u.ID,
			TokenHash: "user-hash-2",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		},
		{
			ID:        uuid.New(),
			UserID:    otherU.ID,
			TokenHash: "other-hash",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		},
	}

	for _, rt := range userTokens {
		if err := repo.Create(context.Background(), rt); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if err := repo.RevokeAllByUserID(context.Background(), u.ID); err != nil {
		t.Fatalf("RevokeAllByUserID() error = %v", err)
	}

	for _, hash := range []string{"user-hash-1", "user-hash-2"} {
		found, err := repo.FindByTokenHash(context.Background(), hash)
		if err != nil {
			t.Fatalf("FindByTokenHash() error = %v", err)
		}
		if found.RevokedAt == nil {
			t.Fatalf("RevokeAllByUserID() did not revoke %q", hash)
		}
	}

	otherFound, err := repo.FindByTokenHash(context.Background(), "other-hash")
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}

	if otherFound.RevokedAt != nil {
		t.Fatal("RevokeAllByUserID() revoked another user's token")
	}
}
