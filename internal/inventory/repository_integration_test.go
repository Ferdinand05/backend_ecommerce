package inventory

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

func seedVariant(t *testing.T, db *gorm.DB, productID uuid.UUID, sku, name string) models.ProductVariant {
	t.Helper()

	variant := models.ProductVariant{
		ID:        uuid.New(),
		ProductID: productID,
		SKU:       sku,
		Name:      name,
		Price:     decimal.NewFromInt(100000),
		IsActive:  true,
	}

	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("failed to seed variant: %v", err)
	}

	return variant
}

func seedInventory(t *testing.T, db *gorm.DB, variantID uuid.UUID, quantity int) models.InventoryItem {
	t.Helper()

	item := models.InventoryItem{
		ID:               uuid.New(),
		ProductVariantID: variantID,
		Quantity:         quantity,
	}

	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to seed inventory item: %v", err)
	}

	return item
}

func seedMovement(t *testing.T, db *gorm.DB, inventoryID uuid.UUID, movementType string, quantity int, createdAt time.Time) models.StockMovement {
	t.Helper()

	movement := models.StockMovement{
		ID:              uuid.New(),
		InventoryItemID: inventoryID,
		Type:            movementType,
		Quantity:        quantity,
		CreatedAt:       createdAt,
	}

	if err := db.Create(&movement).Error; err != nil {
		t.Fatalf("failed to seed stock movement: %v", err)
	}

	return movement
}

func TestInventoryRepositoryCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	category := seedCategory(t, db, "electronics", "electronics")
	product := seedProduct(t, db, category.ID, "laptop", "laptop")
	variant := seedVariant(t, db, product.ID, "LAP-001", "16GB")

	item := models.InventoryItem{
		ID:               uuid.New(),
		ProductVariantID: variant.ID,
		Quantity:         4,
	}

	if err := repo.Create(context.Background(), item); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Quantity != 4 {
		t.Fatalf("Create() = %v, want quantity 4", found)
	}
}

func TestInventoryRepositoryFindByVariantID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	category := seedCategory(t, db, "fashion", "fashion")
	product := seedProduct(t, db, category.ID, "t-shirt", "t-shirt")
	variant := seedVariant(t, db, product.ID, "TSH-M", "size M")
	seeded := seedInventory(t, db, variant.ID, 9)

	found, err := repo.FindByVariantID(context.Background(), variant.ID)
	if err != nil {
		t.Fatalf("FindByVariantID() error = %v", err)
	}

	if found.ID != seeded.ID || found.Quantity != 9 {
		t.Fatalf("FindByVariantID() = %v, want %v", found, seeded)
	}

	if found.Movements != nil {
		t.Fatal("FindByVariantID() must not preload movements")
	}
}

func TestInventoryRepositoryFindByVariantIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	_, err := repo.FindByVariantID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorInventoryNotFound) {
		t.Fatalf("FindByVariantID() error = %v, want %v", err, ErrorInventoryNotFound)
	}
}

func TestInventoryRepositoryFindAllOrdered(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	category := seedCategory(t, db, "toys", "toys")
	product := seedProduct(t, db, category.ID, "lego", "lego")
	first := seedVariant(t, db, product.ID, "LEGO-A", "first")
	second := seedVariant(t, db, product.ID, "LEGO-B", "second")

	seedInventory(t, db, second.ID, 1)
	seedInventory(t, db, first.ID, 2)

	items, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(items) < 2 {
		t.Fatalf("FindAll() len = %d, want at least 2", len(items))
	}

	for i := 1; i < len(items); i++ {
		if items[i].CreatedAt.Before(items[i-1].CreatedAt) {
			t.Fatal("FindAll() must be ordered by created_at ascending")
		}
	}
}

func TestInventoryRepositoryUpdateQuantity(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	category := seedCategory(t, db, "books", "books")
	product := seedProduct(t, db, category.ID, "clean-code", "clean-code")
	variant := seedVariant(t, db, product.ID, "BOOK-01", "hardcover")
	item := seedInventory(t, db, variant.ID, 5)

	if err := repo.UpdateQuantity(context.Background(), item.ID, 0); err != nil {
		t.Fatalf("UpdateQuantity() error = %v", err)
	}

	found, err := repo.FindByID(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Quantity != 0 {
		t.Fatalf("UpdateQuantity() = %d, want 0 (no business rule in repository)", found.Quantity)
	}
}

func TestInventoryRepositoryUpdateQuantityNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	err := repo.UpdateQuantity(context.Background(), uuid.New(), 10)
	if !errors.Is(err, ErrorInventoryNotFound) {
		t.Fatalf("UpdateQuantity() error = %v, want %v", err, ErrorInventoryNotFound)
	}
}

func TestInventoryRepositoryFindByVariantIDForUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInventoryRepository(db)

	category := seedCategory(t, db, "sports", "sports")
	product := seedProduct(t, db, category.ID, "football", "football")
	variant := seedVariant(t, db, product.ID, "BALL-01", "size 5")
	seeded := seedInventory(t, db, variant.ID, 3)

	found, err := repo.FindByVariantIDForUpdate(context.Background(), variant.ID)
	if err != nil {
		t.Fatalf("FindByVariantIDForUpdate() error = %v", err)
	}

	if found.ID != seeded.ID {
		t.Fatalf("FindByVariantIDForUpdate() = %v, want %v", found, seeded)
	}

	_, err = repo.FindByVariantIDForUpdate(context.Background(), uuid.New())
	if !errors.Is(err, ErrorInventoryNotFound) {
		t.Fatalf("FindByVariantIDForUpdate() error = %v, want %v", err, ErrorInventoryNotFound)
	}
}

func TestStockMovementRepositoryCreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStockMovementRepository(db)

	category := seedCategory(t, db, "music", "music")
	product := seedProduct(t, db, category.ID, "guitar", "guitar")
	variant := seedVariant(t, db, product.ID, "GTR-01", "acoustic")
	item := seedInventory(t, db, variant.ID, 0)

	movement := models.StockMovement{
		ID:              uuid.New(),
		InventoryItemID: item.ID,
		Type:            "RESTOCK",
		Quantity:        10,
	}

	if err := repo.Create(context.Background(), movement); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(context.Background(), movement.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Quantity != 10 || found.Type != "RESTOCK" {
		t.Fatalf("FindByID() = %v, want persisted movement", found)
	}
}

func TestStockMovementRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStockMovementRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorStockMovementNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, ErrorStockMovementNotFound)
	}
}

func TestStockMovementRepositoryFindAllByInventoryIDOrdered(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStockMovementRepository(db)

	category := seedCategory(t, db, "garden", "garden")
	product := seedProduct(t, db, category.ID, "shovel", "shovel")
	variant := seedVariant(t, db, product.ID, "SHV-01", "steel")
	item := seedInventory(t, db, variant.ID, 0)

	otherVariant := seedVariant(t, db, product.ID, "SHV-02", "plastic")
	otherItem := seedInventory(t, db, otherVariant.ID, 0)

	base := time.Now().Add(-time.Hour)
	seedMovement(t, db, item.ID, "RESTOCK", 10, base.Add(2*time.Second))
	seedMovement(t, db, item.ID, "SALE", -2, base)
	seedMovement(t, db, otherItem.ID, "RESTOCK", 5, base)

	movements, err := repo.FindAllByInventoryID(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("FindAllByInventoryID() error = %v", err)
	}

	if len(movements) != 2 {
		t.Fatalf("FindAllByInventoryID() len = %d, want 2", len(movements))
	}

	if movements[0].Type != "SALE" || movements[1].Type != "RESTOCK" {
		t.Fatalf("FindAllByInventoryID() = %v, want created_at ascending order", movements)
	}
}
