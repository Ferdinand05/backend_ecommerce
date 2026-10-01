package category

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
		os.Getenv("TEST_DB_HOST"),
		os.Getenv("TEST_DB_USER"),
		os.Getenv("TEST_DB_PASSWORD"),
		os.Getenv("TEST_DB_NAME"),
		os.Getenv("TEST_DB_PORT"),
		os.Getenv("TEST_DB_SSLMODE"),
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

func seedCategory(t *testing.T, db *gorm.DB, name, slug string) models.Category {
	t.Helper()

	category := models.Category{
		ID:   uuid.New(),
		Name: name,
		Slug: slug,
	}

	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("failed to seed category: %v", err)
	}

	return category
}

func TestCategoryRepositoryFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedCategory(t, db, "electronics", "electronics")

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Slug != seeded.Slug {
		t.Fatalf("FindByID() = %v, want %v", found, seeded)
	}
}

func TestCategoryRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorCategoryNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, ErrorCategoryNotFound)
	}
}

func TestCategoryRepositoryFindBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedCategory(t, db, "fashion", "fashion")

	found, err := repo.FindBySlug(context.Background(), "fashion")
	if err != nil {
		t.Fatalf("FindBySlug() error = %v", err)
	}

	if found.ID != seeded.ID {
		t.Fatalf("FindBySlug() = %v, want %v", found, seeded)
	}
}

func TestCategoryRepositoryFindBySlugNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindBySlug(context.Background(), "missing-category")
	if !errors.Is(err, ErrorCategoryNotFound) {
		t.Fatalf("FindBySlug() error = %v, want %v", err, ErrorCategoryNotFound)
	}
}

func TestCategoryRepositoryFindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedCategory(t, db, "books", "books")

	categories, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	found := false
	for _, c := range categories {
		if c.ID == seeded.ID {
			found = true
		}
	}

	if !found {
		t.Fatal("FindAll() did not include seeded category")
	}
}

func TestCategoryRepositoryCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	created, err := repo.Create(context.Background(), models.Category{
		ID:   uuid.New(),
		Name: "toys",
		Slug: "toys",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.Name != "toys" || created.ID == uuid.Nil {
		t.Fatalf("Create() = %v, want persisted toys category", created)
	}
}

func TestCategoryRepositoryUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedCategory(t, db, "garden", "garden")

	updated, err := repo.Update(context.Background(), seeded.ID, models.Category{
		Name: "garden-updated",
		Slug: "garden-updated",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Name != "garden-updated" {
		t.Fatalf("Update() Name = %q, want %q", updated.Name, "garden-updated")
	}

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Name != "garden-updated" {
		t.Fatalf("FindByID() Name = %q, want %q", found.Name, "garden-updated")
	}
}

func TestCategoryRepositoryUpdateNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Update(context.Background(), uuid.New(), models.Category{
		Name: "ghost",
		Slug: "ghost",
	})
	if !errors.Is(err, ErrorCategoryNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, ErrorCategoryNotFound)
	}
}

func TestCategoryRepositoryDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	seeded := seedCategory(t, db, "sports", "sports")

	if err := repo.Delete(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := repo.FindByID(context.Background(), seeded.ID)
	if !errors.Is(err, ErrorCategoryNotFound) {
		t.Fatalf("FindByID() after Delete error = %v, want %v", err, ErrorCategoryNotFound)
	}
}

func TestCategoryRepositoryDeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	err := repo.Delete(context.Background(), uuid.New())
	if !errors.Is(err, ErrorCategoryNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrorCategoryNotFound)
	}
}
