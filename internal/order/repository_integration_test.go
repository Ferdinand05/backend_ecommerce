package order

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"ferdinand/ecommerce/internal/models"

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

func seedOrder(t *testing.T, db *gorm.DB, userID uuid.UUID, status string, createdAt time.Time) models.Order {
	t.Helper()

	order := models.Order{
		ID:          uuid.New(),
		UserID:      userID,
		OrderNumber: "ORD-" + uuid.NewString(),
		Status:      status,
		Subtotal:    decimal.NewFromInt(100000),
		TotalAmount: decimal.NewFromInt(100000),
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}

	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to seed order: %v", err)
	}

	return order
}

func seedOrderItem(t *testing.T, db *gorm.DB, orderID, variantID uuid.UUID) models.OrderItem {
	t.Helper()

	item := models.OrderItem{
		ID:               uuid.New(),
		OrderID:          orderID,
		ProductVariantID: variantID,
		ProductName:      "laptop",
		VariantName:      "16GB",
		SKU:              "SKU-" + uuid.NewString(),
		Price:            decimal.NewFromInt(100000),
		Quantity:         1,
		Subtotal:         decimal.NewFromInt(100000),
	}

	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to seed order item: %v", err)
	}

	return item
}

func TestOrderRepositoryFindByIDPreloadsRelations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)
	order := seedOrder(t, db, user.ID, "PENDING", time.Now())

	seedOrderItem(t, db, order.ID, variant.ID)

	address := models.OrderAddress{
		ID:            uuid.New(),
		OrderID:       order.ID,
		RecipientName: "Budi",
		Phone:         "08123",
		Address:       "Jl. Test 1",
		City:          "Jakarta",
		Province:      "DKI Jakarta",
		PostalCode:    "10110",
	}
	if err := db.Create(&address).Error; err != nil {
		t.Fatalf("failed to seed order address: %v", err)
	}

	history := models.OrderStatusHistory{
		ID:      uuid.New(),
		OrderID: order.ID,
		Status:  "PENDING",
	}
	if err := db.Create(&history).Error; err != nil {
		t.Fatalf("failed to seed status history: %v", err)
	}

	got, err := repo.FindByID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if len(got.Items) != 1 || got.Items[0].ProductName != "laptop" {
		t.Fatalf("items = %v, want one laptop item", got.Items)
	}

	if got.Address == nil || got.Address.RecipientName != "Budi" {
		t.Fatalf("address = %v, want Budi", got.Address)
	}

	if len(got.StatusHistories) != 1 || got.StatusHistories[0].Status != "PENDING" {
		t.Fatalf("histories = %v, want one PENDING", got.StatusHistories)
	}
}

func TestOrderRepositoryFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrorOrderNotFound) {
		t.Fatalf("error = %v, want ErrorOrderNotFound", err)
	}
}

func TestOrderRepositoryFindByIDForUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	order := seedOrder(t, db, user.ID, "PENDING", time.Now())

	got, err := repo.FindByIDForUpdate(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("FindByIDForUpdate() error = %v", err)
	}

	if got.ID != order.ID || got.Status != "PENDING" {
		t.Fatalf("order = %v, want %v PENDING", got, order.ID)
	}

	if _, err := repo.FindByIDForUpdate(context.Background(), uuid.New()); !errors.Is(err, ErrorOrderNotFound) {
		t.Fatalf("error = %v, want ErrorOrderNotFound", err)
	}
}

func TestOrderRepositoryFindAllByUserIDOrdersByNewestAndPaginates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	oldest := seedOrder(t, db, user.ID, "PENDING", base.Add(-48*time.Hour))
	middle := seedOrder(t, db, user.ID, "PAID", base.Add(-24*time.Hour))
	newest := seedOrder(t, db, user.ID, "PENDING", base)

	result, err := repo.FindAllByUserID(context.Background(), user.ID, 1, 2)
	if err != nil {
		t.Fatalf("FindAllByUserID() error = %v", err)
	}

	if result.Total != 3 {
		t.Fatalf("total = %d, want 3", result.Total)
	}

	if len(result.Orders) != 2 {
		t.Fatalf("orders = %d, want 2", len(result.Orders))
	}

	if result.Orders[0].ID != newest.ID || result.Orders[1].ID != middle.ID {
		t.Fatal("orders must be sorted created_at DESC")
	}

	page2, err := repo.FindAllByUserID(context.Background(), user.ID, 2, 2)
	if err != nil {
		t.Fatalf("FindAllByUserID() page 2 error = %v", err)
	}

	if len(page2.Orders) != 1 || page2.Orders[0].ID != oldest.ID {
		t.Fatal("page 2 must contain the oldest order")
	}
}

func TestOrderRepositoryFindAllWithFilters(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	userA := seedUser(t, db)
	userB := seedUser(t, db)
	base := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	seedOrder(t, db, userA.ID, "PENDING", base.Add(-48*time.Hour))
	paidA := seedOrder(t, db, userA.ID, "PAID", base.Add(-24*time.Hour))
	paidB := seedOrder(t, db, userB.ID, "PAID", base)

	ctx := context.Background()

	// filter status
	result, err := repo.FindAll(ctx, OrderFilter{Status: statusPtr("PAID"), Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("FindAll(status) error = %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("status filter total = %d, want 2", result.Total)
	}

	// filter user_id
	result, err = repo.FindAll(ctx, OrderFilter{UserID: &userA.ID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("FindAll(user) error = %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("user filter total = %d, want 2", result.Total)
	}

	// kombinasi status + user_id
	result, err = repo.FindAll(ctx, OrderFilter{UserID: &userB.ID, Status: statusPtr("PAID"), Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("FindAll(user+status) error = %v", err)
	}
	if result.Total != 1 || result.Orders[0].ID != paidB.ID {
		t.Fatalf("user+status filter = %d orders, want only paidB", result.Total)
	}

	// prefix order_number
	prefix := paidA.OrderNumber[:len(paidA.OrderNumber)-4]
	result, err = repo.FindAll(ctx, OrderFilter{OrderNumber: &prefix, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("FindAll(order_number) error = %v", err)
	}
	if result.Total < 1 {
		t.Fatalf("order_number prefix total = %d, want at least 1", result.Total)
	}

	// rentang tanggal (end eksklusif)
	start := base.Add(-24 * time.Hour)
	end := base.Add(time.Hour)
	result, err = repo.FindAll(ctx, OrderFilter{StartDate: &start, EndDate: &end, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("FindAll(dates) error = %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("date range total = %d, want 2", result.Total)
	}
}

func TestOrderRepositoryUpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	order := seedOrder(t, db, user.ID, "PENDING", time.Now())

	if err := repo.UpdateStatus(context.Background(), order.ID, "PAID"); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	got, err := repo.FindByID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if got.Status != "PAID" {
		t.Fatalf("status = %q, want PAID", got.Status)
	}

	if err := repo.UpdateStatus(context.Background(), uuid.New(), "PAID"); !errors.Is(err, ErrorOrderNotFound) {
		t.Fatalf("error = %v, want ErrorOrderNotFound", err)
	}
}

func TestOrderRepositoryCreateWithRelations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)

	order := models.Order{
		ID:          uuid.New(),
		UserID:      user.ID,
		OrderNumber: "ORD-" + uuid.NewString(),
		Status:      "PENDING",
		Subtotal:    decimal.NewFromInt(15000000),
		TotalAmount: decimal.NewFromInt(15000000),
	}

	if err := repo.Create(context.Background(), &order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	item := models.OrderItem{
		ID:               uuid.New(),
		OrderID:          order.ID,
		ProductVariantID: variant.ID,
		ProductName:      "laptop",
		VariantName:      "16GB",
		SKU:              variant.SKU,
		Price:            decimal.NewFromInt(15000000),
		Quantity:         1,
		Subtotal:         decimal.NewFromInt(15000000),
	}
	if err := repo.CreateItem(context.Background(), &item); err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}

	address := models.OrderAddress{
		ID:            uuid.New(),
		OrderID:       order.ID,
		RecipientName: "Siti",
		Phone:         "08123",
		Address:       "Jl. Test 2",
		City:          "Bandung",
		Province:      "Jawa Barat",
		PostalCode:    "40111",
	}
	if err := repo.CreateAddress(context.Background(), &address); err != nil {
		t.Fatalf("CreateAddress() error = %v", err)
	}

	history := models.OrderStatusHistory{
		ID:      uuid.New(),
		OrderID: order.ID,
		Status:  "PENDING",
	}
	if err := repo.CreateStatusHistory(context.Background(), &history); err != nil {
		t.Fatalf("CreateStatusHistory() error = %v", err)
	}

	got, err := repo.FindByID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if len(got.Items) != 1 || got.Address == nil || len(got.StatusHistories) != 1 {
		t.Fatalf("relations = (%d items, %v address, %d histories), want 1/1/1",
			len(got.Items), got.Address != nil, len(got.StatusHistories))
	}
}

func statusPtr(s string) *OrderStatus {
	status := OrderStatus(s)
	return &status
}
