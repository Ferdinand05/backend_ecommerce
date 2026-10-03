package productimage

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

func seedVariant(t *testing.T, db *gorm.DB, productID uuid.UUID, sku, name string) models.ProductVariant {
	t.Helper()

	variant := models.ProductVariant{
		ID:        uuid.New(),
		ProductID: productID,
		SKU:       sku,
		Name:      name,
		IsActive:  true,
	}

	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("failed to seed product variant: %v", err)
	}

	return variant
}

func seedImage(t *testing.T, db *gorm.DB, productID uuid.UUID, variantID *uuid.UUID, key string, sortOrder int) models.ProductImage {
	t.Helper()

	image := models.ProductImage{
		ID:               uuid.New(),
		ProductID:        productID,
		ProductVariantID: variantID,
		StorageKey:       key,
		SortOrder:        sortOrder,
	}

	if err := db.Create(&image).Error; err != nil {
		t.Fatalf("failed to seed product image: %v", err)
	}

	return image
}

func TestProductImageRepositoryCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "electronics", "electronics")
	product := seedProduct(t, db, category.ID, "laptop", "laptop")

	image := models.ProductImage{
		ID:         uuid.New(),
		ProductID:  product.ID,
		StorageKey: "products/" + product.ID.String() + "/" + uuid.New().String() + ".png",
	}

	if err := repo.Create(context.Background(), image); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(context.Background(), image.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.StorageKey != image.StorageKey {
		t.Fatalf("Create() = %v, want persisted image", found)
	}
}

func TestProductImageRepositoryFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "fashion", "fashion")
	product := seedProduct(t, db, category.ID, "t-shirt", "t-shirt")
	seeded := seedImage(t, db, product.ID, nil, "products/parent.png", 0)

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.ID != seeded.ID {
		t.Fatalf("FindByID() = %v, want %v", found, seeded)
	}
}

func TestProductImageRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorProductImageNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, ErrorProductImageNotFound)
	}
}

func TestProductImageRepositoryFindAllByProductIDParentOnly(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "toys", "toys")
	product := seedProduct(t, db, category.ID, "lego", "lego")
	variant := seedVariant(t, db, product.ID, "LEGO-01", "classic")

	parent := seedImage(t, db, product.ID, nil, "products/parent.png", 1)
	seedImage(t, db, product.ID, &variant.ID, "products/variant.png", 0)

	images, err := repo.FindAllByProductID(context.Background(), product.ID)
	if err != nil {
		t.Fatalf("FindAllByProductID() error = %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("FindAllByProductID() len = %d, want 1 parent image only", len(images))
	}

	if images[0].ID != parent.ID {
		t.Fatalf("FindAllByProductID()[0] = %v, want parent image", images[0].ID)
	}
}

func TestProductImageRepositoryFindAllByVariantIDOrdered(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "books", "books")
	product := seedProduct(t, db, category.ID, "clean-code", "clean-code")
	variant := seedVariant(t, db, product.ID, "BOOK-01", "hardcover")

	seedImage(t, db, product.ID, nil, "products/parent.png", 0)
	second := seedImage(t, db, product.ID, &variant.ID, "products/v2.png", 1)
	first := seedImage(t, db, product.ID, &variant.ID, "products/v1.png", 0)

	images, err := repo.FindAllByVariantID(context.Background(), product.ID, variant.ID)
	if err != nil {
		t.Fatalf("FindAllByVariantID() error = %v", err)
	}

	if len(images) != 2 {
		t.Fatalf("FindAllByVariantID() len = %d, want 2", len(images))
	}

	if images[0].ID != first.ID || images[1].ID != second.ID {
		t.Fatalf("FindAllByVariantID() order = %v, want sort_order ascending", []uuid.UUID{images[0].ID, images[1].ID})
	}
}

func TestProductImageRepositoryDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	category := seedCategory(t, db, "sports", "sports")
	product := seedProduct(t, db, category.ID, "football", "football")
	seeded := seedImage(t, db, product.ID, nil, "products/football.png", 0)

	if err := repo.Delete(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := repo.FindByID(context.Background(), seeded.ID)
	if !errors.Is(err, ErrorProductImageNotFound) {
		t.Fatalf("FindByID() after Delete error = %v, want %v", err, ErrorProductImageNotFound)
	}
}

func TestProductImageRepositoryDeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	err := repo.Delete(context.Background(), uuid.New())
	if !errors.Is(err, ErrorProductImageNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrorProductImageNotFound)
	}
}
