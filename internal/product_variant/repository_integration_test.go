package productvariant

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

func seedVariant(t *testing.T, db *gorm.DB, productID uuid.UUID, sku, name string, price decimal.Decimal, createdAt time.Time) models.ProductVariant {
	t.Helper()

	variant := models.ProductVariant{
		ID:        uuid.New(),
		ProductID: productID,
		SKU:       sku,
		Name:      name,
		Price:     price,
		IsActive:  true,
		CreatedAt: createdAt,
	}

	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("failed to seed product variant: %v", err)
	}

	return variant
}

func TestVariantRepositoryCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "electronics", "electronics")
	product := seedProduct(t, db, category.ID, "laptop", "laptop")

	variant := models.ProductVariant{
		ID:        uuid.New(),
		ProductID: product.ID,
		SKU:       "LAP-001",
		Name:      "16GB",
		Price:     decimal.NewFromInt(15000000),
		IsActive:  true,
	}

	if err := repo.Create(context.Background(), variant); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(context.Background(), variant.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.SKU != "LAP-001" || !found.IsActive {
		t.Fatalf("Create() = %v, want persisted active variant", found)
	}
}

func TestVariantRepositoryFindAllByProductIDOrdered(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "electronics", "electronics")
	product := seedProduct(t, db, category.ID, "laptop", "laptop")
	other := seedProduct(t, db, category.ID, "monitor", "monitor")

	base := time.Now().Add(-time.Hour)

	seedVariant(t, db, product.ID, "SKU-C", "third", decimal.NewFromInt(3), base.Add(2*time.Second))
	seedVariant(t, db, product.ID, "SKU-A", "first", decimal.NewFromInt(1), base)
	seedVariant(t, db, product.ID, "SKU-B", "second", decimal.NewFromInt(2), base.Add(time.Second))
	seedVariant(t, db, other.ID, "SKU-OTHER", "other", decimal.NewFromInt(9), base)

	variants, err := repo.FindAllByProductID(context.Background(), product.ID)
	if err != nil {
		t.Fatalf("FindAllByProductID() error = %v", err)
	}

	if len(variants) != 3 {
		t.Fatalf("FindAllByProductID() len = %d, want 3", len(variants))
	}

	want := []string{"SKU-A", "SKU-B", "SKU-C"}
	for i, v := range variants {
		if v.SKU != want[i] {
			t.Fatalf("FindAllByProductID()[%d] = %q, want %q", i, v.SKU, want[i])
		}
	}
}

func TestVariantRepositoryFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "fashion", "fashion")
	product := seedProduct(t, db, category.ID, "t-shirt", "t-shirt")
	seeded := seedVariant(t, db, product.ID, "TSH-M", "size M", decimal.NewFromInt(99000), time.Now())

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.SKU != seeded.SKU {
		t.Fatalf("FindByID() = %v, want %v", found, seeded)
	}

	if found.Product.ID != uuid.Nil {
		t.Fatal("FindByID() must not preload product")
	}
}

func TestVariantRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorProductVariantNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, ErrorProductVariantNotFound)
	}
}

func TestVariantRepositoryFindBySKU(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "books", "books")
	product := seedProduct(t, db, category.ID, "clean-code", "clean-code")
	seeded := seedVariant(t, db, product.ID, "BOOK-01", "hardcover", decimal.NewFromInt(150000), time.Now())

	found, err := repo.FindBySKU(context.Background(), "BOOK-01")
	if err != nil {
		t.Fatalf("FindBySKU() error = %v", err)
	}

	if found.ID != seeded.ID {
		t.Fatalf("FindBySKU() = %v, want %v", found, seeded)
	}
}

func TestVariantRepositoryFindBySKUNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindBySKU(context.Background(), "missing-sku")
	if !errors.Is(err, ErrorProductVariantNotFound) {
		t.Fatalf("FindBySKU() error = %v, want %v", err, ErrorProductVariantNotFound)
	}
}

func TestVariantRepositoryExistsBySKUExceptID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "toys", "toys")
	product := seedProduct(t, db, category.ID, "lego", "lego")
	seeded := seedVariant(t, db, product.ID, "LEGO-01", "classic", decimal.NewFromInt(500000), time.Now())

	exists, err := repo.ExistsBySKUExceptID(context.Background(), "LEGO-01", uuid.New())
	if err != nil {
		t.Fatalf("ExistsBySKUExceptID() error = %v", err)
	}

	if !exists {
		t.Fatal("ExistsBySKUExceptID() = false, want true for other id")
	}

	exists, err = repo.ExistsBySKUExceptID(context.Background(), "LEGO-01", seeded.ID)
	if err != nil {
		t.Fatalf("ExistsBySKUExceptID() error = %v", err)
	}

	if exists {
		t.Fatal("ExistsBySKUExceptID() = true, want false for own id")
	}

	exists, err = repo.ExistsBySKUExceptID(context.Background(), "missing-sku", uuid.New())
	if err != nil {
		t.Fatalf("ExistsBySKUExceptID() error = %v", err)
	}

	if exists {
		t.Fatal("ExistsBySKUExceptID() = true, want false for missing sku")
	}
}

func TestVariantRepositoryUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "sports", "sports")
	product := seedProduct(t, db, category.ID, "football", "football")
	seeded := seedVariant(t, db, product.ID, "BALL-01", "size 5", decimal.NewFromInt(200000), time.Now())

	err := repo.Update(context.Background(), seeded.ID, models.ProductVariant{
		SKU:   "BALL-02",
		Name:  "size 4",
		Price: decimal.NewFromInt(180000),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.SKU != "BALL-02" || found.Name != "size 4" {
		t.Fatalf("Update() = %v, want updated sku and name", found)
	}

	if !found.IsActive {
		t.Fatal("Update() must not change IsActive")
	}
}

func TestVariantRepositoryUpdateNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	err := repo.Update(context.Background(), uuid.New(), models.ProductVariant{
		SKU:  "GHOST-01",
		Name: "ghost",
	})
	if !errors.Is(err, ErrorProductVariantNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, ErrorProductVariantNotFound)
	}
}

func TestVariantRepositoryDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "music", "music")
	product := seedProduct(t, db, category.ID, "guitar", "guitar")
	seeded := seedVariant(t, db, product.ID, "GTR-01", "acoustic", decimal.NewFromInt(2500000), time.Now())

	if err := repo.Delete(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := repo.FindByID(context.Background(), seeded.ID)
	if !errors.Is(err, ErrorProductVariantNotFound) {
		t.Fatalf("FindByID() after Delete error = %v, want %v", err, ErrorProductVariantNotFound)
	}
}

func TestVariantRepositoryDeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	err := repo.Delete(context.Background(), uuid.New())
	if !errors.Is(err, ErrorProductVariantNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrorProductVariantNotFound)
	}
}
