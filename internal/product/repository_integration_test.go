package product

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
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

func seedProduct(t *testing.T, db *gorm.DB, categoryID uuid.UUID, name, slug string) models.Product {
	t.Helper()

	product := models.Product{
		ID:         uuid.New(),
		CategoryID: categoryID,
		Name:       name,
		Slug:       slug,
	}

	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	return product
}

func seedVariant(t *testing.T, db *gorm.DB, productID uuid.UUID, sku, name string, price decimal.Decimal) models.ProductVariant {
	t.Helper()

	variant := models.ProductVariant{
		ID:        uuid.New(),
		ProductID: productID,
		SKU:       sku,
		Name:      name,
		Price:     price,
	}

	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("failed to seed product variant: %v", err)
	}

	return variant
}

func TestProductRepositoryFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "electronics", "electronics")
	seeded := seedProduct(t, db, category.ID, "laptop", "laptop")
	seedVariant(t, db, seeded.ID, "LAP-001", "16GB", decimal.NewFromInt(15000000))

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Slug != seeded.Slug {
		t.Fatalf("FindByID() = %v, want %v", found, seeded)
	}

	if found.Category.ID != category.ID {
		t.Fatalf("FindByID() category = %v, want %v", found.Category.ID, category.ID)
	}

	if len(found.Variants) != 1 || found.Variants[0].SKU != "LAP-001" {
		t.Fatalf("FindByID() variants = %v, want 1 variant with SKU LAP-001", found.Variants)
	}
}

func TestProductRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorProductNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, ErrorProductNotFound)
	}
}

func TestProductRepositoryFindBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "fashion", "fashion")
	seeded := seedProduct(t, db, category.ID, "t-shirt", "t-shirt")
	seedVariant(t, db, seeded.ID, "TSH-M", "size M", decimal.NewFromInt(99000))

	found, err := repo.FindBySlug(context.Background(), "t-shirt")
	if err != nil {
		t.Fatalf("FindBySlug() error = %v", err)
	}

	if found.ID != seeded.ID {
		t.Fatalf("FindBySlug() = %v, want %v", found, seeded)
	}

	if found.Category.ID != category.ID {
		t.Fatalf("FindBySlug() category = %v, want %v", found.Category.ID, category.ID)
	}

	if len(found.Variants) != 1 || found.Variants[0].SKU != "TSH-M" {
		t.Fatalf("FindBySlug() variants = %v, want 1 variant with SKU TSH-M", found.Variants)
	}
}

func TestProductRepositoryFindBySlugNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindBySlug(context.Background(), "missing-product")
	if !errors.Is(err, ErrorProductNotFound) {
		t.Fatalf("FindBySlug() error = %v, want %v", err, ErrorProductNotFound)
	}
}

func TestProductRepositoryExistsBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "books", "books")
	seedProduct(t, db, category.ID, "clean-code", "clean-code")

	exists, err := repo.ExistsBySlug(context.Background(), "clean-code")
	if err != nil {
		t.Fatalf("ExistsBySlug() error = %v", err)
	}

	if !exists {
		t.Fatal("ExistsBySlug() = false, want true for seeded slug")
	}

	exists, err = repo.ExistsBySlug(context.Background(), "missing-product")
	if err != nil {
		t.Fatalf("ExistsBySlug() error = %v", err)
	}

	if exists {
		t.Fatal("ExistsBySlug() = true, want false for missing slug")
	}
}

func TestProductRepositoryFindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "toys", "toys")
	seeded := seedProduct(t, db, category.ID, "lego", "lego")

	products, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	found := false
	for _, p := range products {
		if p.ID == seeded.ID {
			found = true

			if p.Category.ID != category.ID {
				t.Fatalf("FindAll() category = %v, want %v", p.Category.ID, category.ID)
			}
		}
	}

	if !found {
		t.Fatal("FindAll() did not include seeded product")
	}
}

func TestProductRepositoryCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "sports", "sports")

	created, err := repo.Create(context.Background(), models.Product{
		ID:         uuid.New(),
		CategoryID: category.ID,
		Name:       "football",
		Slug:       "football",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.Name != "football" || created.ID == uuid.Nil {
		t.Fatalf("Create() = %v, want persisted football product", created)
	}

	if !created.IsActive {
		t.Fatal("Create() IsActive = false, want default true")
	}
}

func TestProductRepositoryUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "garden", "garden")
	seeded := seedProduct(t, db, category.ID, "shovel", "shovel")

	updated, err := repo.Update(context.Background(), seeded.ID, models.Product{
		Name: "shovel-pro",
		Slug: "shovel-pro",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Name != "shovel-pro" {
		t.Fatalf("Update() Name = %q, want %q", updated.Name, "shovel-pro")
	}

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Name != "shovel-pro" {
		t.Fatalf("FindByID() Name = %q, want %q", found.Name, "shovel-pro")
	}
}

func TestProductRepositoryUpdateNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Update(context.Background(), uuid.New(), models.Product{
		Name: "ghost",
		Slug: "ghost",
	})
	if !errors.Is(err, ErrorProductNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, ErrorProductNotFound)
	}
}

func TestProductRepositoryDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "music", "music")
	seeded := seedProduct(t, db, category.ID, "guitar", "guitar")

	if err := repo.Delete(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := repo.FindByID(context.Background(), seeded.ID)
	if !errors.Is(err, ErrorProductNotFound) {
		t.Fatalf("FindByID() after Delete error = %v, want %v", err, ErrorProductNotFound)
	}
}

func TestProductRepositoryDeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	err := repo.Delete(context.Background(), uuid.New())
	if !errors.Is(err, ErrorProductNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrorProductNotFound)
	}
}
