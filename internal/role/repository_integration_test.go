package role

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

func seedRole(t *testing.T, db *gorm.DB, name string) models.Role {
	t.Helper()

	role := models.Role{
		ID:   uuid.New(),
		Name: name,
	}

	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("failed to seed role: %v", err)
	}

	return role
}

func TestRepositoryFindByName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	role, err := repo.FindByName(context.Background(), "customer")
	if err != nil {
		t.Fatalf("FindByName() error = %v", err)
	}

	if role.Name != "customer" {
		t.Fatalf("FindByName() = %v, want customer role", role)
	}
}

func TestRepositoryFindByNameNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByName(context.Background(), "missing-role")
	if !errors.Is(err, ErrorRoleNotFound) {
		t.Fatalf("FindByName() error = %v, want %v", err, ErrorRoleNotFound)
	}
}

func TestRepositoryFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedRole(t, db, "seller")

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Name != "seller" {
		t.Fatalf("FindByID() = %v, want %v", found, seeded)
	}
}

func TestRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorRoleNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, ErrorRoleNotFound)
	}
}

func TestRepositoryFindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedRole(t, db, "seller")

	roles, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	found := false
	for _, r := range roles {
		if r.ID == seeded.ID {
			found = true
		}
	}

	if !found {
		t.Fatal("FindAll() did not include seeded role")
	}
}

func TestRepositoryUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedRole(t, db, "seller")

	updated, err := repo.Update(context.Background(), models.Role{
		ID:   seeded.ID,
		Name: "seller-updated",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Name != "seller-updated" {
		t.Fatalf("Update() Name = %q, want %q", updated.Name, "seller-updated")
	}

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Name != "seller-updated" {
		t.Fatalf("FindByID() Name = %q, want %q", found.Name, "seller-updated")
	}
}

func TestRepositoryDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedRole(t, db, "seller")

	if err := repo.Delete(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := repo.FindByID(context.Background(), seeded.ID)
	if !errors.Is(err, ErrorRoleNotFound) {
		t.Fatalf("FindByID() after Delete error = %v, want %v", err, ErrorRoleNotFound)
	}
}
