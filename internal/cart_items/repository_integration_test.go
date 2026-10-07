package cartitems

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

	if err := godotenv.Load("../../.env.test"); err != nil {
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

func seedUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()

	role := models.Role{ID: uuid.New(), Name: "customer-" + uuid.NewString()}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("failed to seed role: %v", err)
	}

	user := models.User{
		ID:           uuid.New(),
		RoleID:       role.ID,
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "hashed",
		FirstName:    "Test",
		Status:       "active",
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	return user
}

func seedVariant(t *testing.T, db *gorm.DB) models.ProductVariant {
	t.Helper()

	category := models.Category{ID: uuid.New(), Name: "electronics-" + uuid.NewString(), Slug: "electronics-" + uuid.NewString()}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("failed to seed category: %v", err)
	}

	product := models.Product{
		ID:         uuid.New(),
		CategoryID: category.ID,
		Name:       "laptop",
		Slug:       "laptop-" + uuid.NewString(),
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	variant := models.ProductVariant{
		ID:        uuid.New(),
		ProductID: product.ID,
		SKU:       "SKU-" + uuid.NewString(),
		Name:      "16GB",
		Price:     decimal.NewFromInt(15000000),
		IsActive:  true,
	}
	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("failed to seed product variant: %v", err)
	}

	return variant
}

func seedCartItem(t *testing.T, db *gorm.DB, userID, variantID uuid.UUID, quantity int, createdAt time.Time) models.CartItem {
	t.Helper()

	item := models.CartItem{
		ID:               uuid.New(),
		UserID:           userID,
		ProductVariantID: variantID,
		Quantity:         quantity,
		CreatedAt:        createdAt,
	}

	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to seed cart item: %v", err)
	}

	return item
}

func TestCartRepositoryAddQuantityCreates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)

	item, err := repo.AddQuantity(context.Background(), user.ID, variant.ID, 1)
	if err != nil {
		t.Fatalf("AddQuantity() error = %v", err)
	}

	if item.Quantity != 1 {
		t.Fatalf("AddQuantity() quantity = %d, want 1", item.Quantity)
	}

	if item.ProductVariant.SKU != variant.SKU || item.ProductVariant.Product.Name != "laptop" {
		t.Fatalf("AddQuantity() must preload variant and product, got %v", item.ProductVariant)
	}
}

func TestCartRepositoryAddQuantityUpserts(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)

	if _, err := repo.AddQuantity(context.Background(), user.ID, variant.ID, 1); err != nil {
		t.Fatalf("AddQuantity() first error = %v", err)
	}

	item, err := repo.AddQuantity(context.Background(), user.ID, variant.ID, 1)
	if err != nil {
		t.Fatalf("AddQuantity() second error = %v", err)
	}

	if item.Quantity != 2 {
		t.Fatalf("AddQuantity() quantity = %d, want 2", item.Quantity)
	}

	var count int64
	if err := db.Model(&models.CartItem{}).Where("user_id = ? AND product_variant_id = ?", user.ID, variant.ID).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}

	if count != 1 {
		t.Fatalf("rows = %d, want 1 (upsert must not duplicate)", count)
	}
}

func TestCartRepositoryIncreaseQuantity(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)
	seeded := seedCartItem(t, db, user.ID, variant.ID, 2, time.Now())

	item, err := repo.IncreaseQuantity(context.Background(), user.ID, seeded.ID, 1)
	if err != nil {
		t.Fatalf("IncreaseQuantity() error = %v", err)
	}

	if item.Quantity != 3 {
		t.Fatalf("IncreaseQuantity() quantity = %d, want 3", item.Quantity)
	}

	if item.ProductVariant.SKU != variant.SKU {
		t.Fatal("IncreaseQuantity() must preload variant")
	}
}

func TestCartRepositoryIncreaseQuantityNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	if _, err := repo.IncreaseQuantity(context.Background(), uuid.New(), uuid.New(), 1); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("IncreaseQuantity() error = %v, want %v", err, ErrorCartItemNotFound)
	}
}

func TestCartRepositoryIncreaseQuantityOtherUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	owner := seedUser(t, db)
	intruder := seedUser(t, db)
	variant := seedVariant(t, db)
	seeded := seedCartItem(t, db, owner.ID, variant.ID, 1, time.Now())

	if _, err := repo.IncreaseQuantity(context.Background(), intruder.ID, seeded.ID, 1); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("IncreaseQuantity() error = %v, want %v", err, ErrorCartItemNotFound)
	}

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Quantity != 1 {
		t.Fatalf("quantity = %d, want untouched 1", found.Quantity)
	}
}

func TestCartRepositoryDecreaseQuantity(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)
	seeded := seedCartItem(t, db, user.ID, variant.ID, 2, time.Now())

	item, err := repo.DecreaseQuantity(context.Background(), user.ID, seeded.ID, 1)
	if err != nil {
		t.Fatalf("DecreaseQuantity() error = %v", err)
	}

	if item.Quantity != 1 {
		t.Fatalf("DecreaseQuantity() quantity = %d, want 1", item.Quantity)
	}
}

func TestCartRepositoryDecreaseQuantityNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	if _, err := repo.DecreaseQuantity(context.Background(), uuid.New(), uuid.New(), 1); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("DecreaseQuantity() error = %v, want %v", err, ErrorCartItemNotFound)
	}
}

func TestCartRepositoryFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)
	seeded := seedCartItem(t, db, user.ID, variant.ID, 2, time.Now())

	found, err := repo.FindByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Quantity != 2 {
		t.Fatalf("FindByID() quantity = %d, want 2", found.Quantity)
	}

	if found.ProductVariant.ID != uuid.Nil {
		t.Fatal("FindByID() must not preload variant")
	}
}

func TestCartRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	if _, err := repo.FindByID(context.Background(), uuid.New()); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, ErrorCartItemNotFound)
	}
}

func TestCartRepositoryFindByUserIDOrdered(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	other := seedUser(t, db)
	variantA := seedVariant(t, db)
	variantB := seedVariant(t, db)
	variantC := seedVariant(t, db)

	base := time.Now().Add(-time.Hour)
	seedCartItem(t, db, user.ID, variantC.ID, 1, base.Add(2*time.Second))
	seedCartItem(t, db, user.ID, variantA.ID, 1, base)
	seedCartItem(t, db, user.ID, variantB.ID, 1, base.Add(time.Second))
	seedCartItem(t, db, other.ID, variantA.ID, 1, base)

	items, err := repo.FindByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("FindByUserID() error = %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("FindByUserID() len = %d, want 3", len(items))
	}

	want := []uuid.UUID{variantA.ID, variantB.ID, variantC.ID}
	for i, item := range items {
		if item.ProductVariantID != want[i] {
			t.Fatalf("FindByUserID()[%d] = %v, want %v", i, item.ProductVariantID, want[i])
		}
	}

	if items[0].ProductVariant.ID == uuid.Nil {
		t.Fatal("FindByUserID() must preload variant")
	}
}

func TestCartRepositoryDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)
	seeded := seedCartItem(t, db, user.ID, variant.ID, 1, time.Now())

	if err := repo.Delete(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := repo.FindByID(context.Background(), seeded.ID); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("FindByID() after Delete error = %v, want %v", err, ErrorCartItemNotFound)
	}
}

func TestCartRepositoryDeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	if err := repo.Delete(context.Background(), uuid.New()); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrorCartItemNotFound)
	}
}

func TestCartRepositoryDeleteAllByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	other := seedUser(t, db)
	variantA := seedVariant(t, db)
	variantB := seedVariant(t, db)

	seedCartItem(t, db, user.ID, variantA.ID, 1, time.Now())
	seedCartItem(t, db, user.ID, variantB.ID, 1, time.Now())
	seedCartItem(t, db, other.ID, variantA.ID, 1, time.Now())

	if err := repo.DeleteAllByUserID(context.Background(), user.ID); err != nil {
		t.Fatalf("DeleteAllByUserID() error = %v", err)
	}

	items, err := repo.FindByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("FindByUserID() error = %v", err)
	}

	if len(items) != 0 {
		t.Fatalf("deleted user items = %d, want 0", len(items))
	}

	remaining, err := repo.FindByUserID(context.Background(), other.ID)
	if err != nil {
		t.Fatalf("FindByUserID() other error = %v", err)
	}

	if len(remaining) != 1 {
		t.Fatalf("other user items = %d, want 1", len(remaining))
	}
}
