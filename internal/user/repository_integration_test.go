package user

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const customerRoleID = "00000000-0000-0000-0000-000000000001"

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	err := godotenv.Load("../../.env.test")
	if err != nil {
		t.Fatalf("failed to load .env.test: %v", err)
	}

	dbUser := os.Getenv("TEST_DB_USER")
	dbPassword := os.Getenv("TEST_DB_PASSWORD")
	dbHost := os.Getenv("TEST_DB_HOST")
	dbPort := os.Getenv("TEST_DB_PORT")
	dbName := os.Getenv("TEST_DB_NAME")
	dbSSLMode := os.Getenv("TEST_DB_SSLMODE")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbHost,
		dbUser,
		dbPassword,
		dbName,
		dbPort,
		dbSSLMode,
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

func newTestUser() models.User {
	return models.User{
		ID:           uuid.New(),
		RoleID:       uuid.MustParse(customerRoleID),
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "hashed-password",
		FirstName:    "Test",
		Status:       "active",
	}
}

func TestRepositoryCreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := newTestUser()

	created, err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID != user.ID {
		t.Fatalf("Create() ID = %v, want %v", created.ID, user.ID)
	}

	found, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Email != user.Email {
		t.Fatalf("FindByID() Email = %q, want %q", found.Email, user.Email)
	}
}

func TestRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorUserNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, ErrorUserNotFound)
	}
}

func TestRepositoryFindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := newTestUser()
	if _, err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByEmail(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}

	if found.ID != user.ID {
		t.Fatalf("FindByEmail() ID = %v, want %v", found.ID, user.ID)
	}

	if found.Role.ID != uuid.MustParse(customerRoleID) {
		t.Fatalf("FindByEmail() Role.ID = %v, want customer role", found.Role.ID)
	}
}

func TestRepositoryFindByEmailNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByEmail(context.Background(), "missing@example.com")
	if !errors.Is(err, ErrorUserNotFound) {
		t.Fatalf("FindByEmail() error = %v, want %v", err, ErrorUserNotFound)
	}
}

func TestRepositoryFindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := newTestUser()
	if _, err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	users, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	found := false
	for _, u := range users {
		if u.ID == user.ID {
			found = true
		}
	}

	if !found {
		t.Fatal("FindAll() did not include created user")
	}
}

func TestRepositoryMarkEmailVerified(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := newTestUser()
	if _, err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.MarkEmailVerified(context.Background(), user.ID); err != nil {
		t.Fatalf("MarkEmailVerified() error = %v", err)
	}

	found, err := repo.FindByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.EmailVerifiedAt == nil {
		t.Fatal("MarkEmailVerified() did not set EmailVerifiedAt")
	}
}

func TestRepositoryMarkEmailVerifiedUserNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	err := repo.MarkEmailVerified(context.Background(), uuid.New())
	if !errors.Is(err, ErrorUserNotFound) {
		t.Fatalf("MarkEmailVerified() error = %v, want %v", err, ErrorUserNotFound)
	}
}

func TestRepositoryUpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := newTestUser()
	if _, err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.UpdatePassword(context.Background(), user.ID, "new-hash"); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	found, err := repo.FindByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.PasswordHash != "new-hash" {
		t.Fatalf("PasswordHash = %q, want %q", found.PasswordHash, "new-hash")
	}
}

func TestRepositoryUpdatePasswordUserNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	err := repo.UpdatePassword(context.Background(), uuid.New(), "new-hash")
	if !errors.Is(err, ErrorUserNotFound) {
		t.Fatalf("UpdatePassword() error = %v, want %v", err, ErrorUserNotFound)
	}
}
