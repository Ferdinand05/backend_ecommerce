package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/order"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- setup DB (pola sama dengan package order: tx + rollback) ---

func setupWebhookDB(t *testing.T) *gorm.DB {
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
		t.Fatalf("connect: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin: %v", tx.Error)
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

// --- seeds ---

type webhookFixture struct {
	db      *gorm.DB
	order   models.Order
	payment models.Payment
	variant models.ProductVariant
}

func seedWebhookWorld(t *testing.T, db *gorm.DB, orderStatus string) webhookFixture {
	t.Helper()

	role := models.Role{ID: uuid.New(), Name: "customer-" + uuid.NewString()}
	require.NoError(t, db.Create(&role).Error)

	user := models.User{
		ID: uuid.New(), RoleID: role.ID,
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "hashed", FirstName: "Test", Status: "active",
	}
	require.NoError(t, db.Create(&user).Error)

	category := models.Category{
		ID:   uuid.New(),
		Name: "cat-" + uuid.NewString(),
		Slug: "cat-" + uuid.NewString(),
	}
	require.NoError(t, db.Create(&category).Error)

	product := models.Product{
		ID: uuid.New(), CategoryID: category.ID,
		Name: "laptop", Slug: "laptop-" + uuid.NewString(),
	}
	require.NoError(t, db.Create(&product).Error)

	variant := models.ProductVariant{
		ID: uuid.New(), ProductID: product.ID,
		SKU: "SKU-" + uuid.NewString(), Name: "16GB",
		Price: decimal.NewFromInt(60000), IsActive: true,
	}
	require.NoError(t, db.Create(&variant).Error)

	ord := models.Order{
		ID:          uuid.New(),
		UserID:      user.ID,
		OrderNumber: "ORD-" + uuid.NewString(),
		Status:      orderStatus,
		Subtotal:    decimal.NewFromInt(120000),
		TotalAmount: decimal.NewFromInt(120000),
	}
	require.NoError(t, db.Create(&ord).Error)

	item := models.OrderItem{
		ID:               uuid.New(),
		OrderID:          ord.ID,
		ProductVariantID: variant.ID,
		ProductName:      "laptop",
		VariantName:      "16GB",
		SKU:              variant.SKU,
		Price:            decimal.NewFromInt(60000),
		Quantity:         2,
		Subtotal:         decimal.NewFromInt(120000),
	}
	require.NoError(t, db.Create(&item).Error)

	// Stok terreservasi oleh create order: quantity 10, reserved 2.
	inv := models.InventoryItem{
		ID:               uuid.New(),
		ProductVariantID: variant.ID,
		Quantity:         10,
		ReservedQuantity: 2,
	}
	require.NoError(t, db.Create(&inv).Error)

	payment := models.Payment{
		ID:         uuid.New(),
		OrderID:    ord.ID,
		Provider:   models.PaymentProviderXendit,
		ExternalID: "PAY-" + uuid.NewString(),
		Status:     string(PaymentStatusPending),
		Amount:     decimal.NewFromInt(120000),
		Currency:   "IDR",
	}
	invoiceID := "inv-" + uuid.NewString()
	payment.ProviderInvoiceID = &invoiceID
	require.NoError(t, db.Create(&payment).Error)

	return webhookFixture{db: db, order: ord, payment: payment, variant: variant}
}

func (f webhookFixture) callback(status string) InvoiceCallback {
	return InvoiceCallback{
		ID:         *f.payment.ProviderInvoiceID,
		ExternalID: f.payment.ExternalID,
		Status:     status,
		Amount:     120000,
		Currency:   "IDR",
		PaidAt:     ptr(time.Now()),
	}
}

func newRealService(db *gorm.DB) Service {
	return NewService(
		db,
		order.NewRepository(db),
		NewRepository(db),
		inventory.NewInventoryRepository(db),
		inventory.NewStockMovementRepository(db),
		&fakeXendit{},
	)
}

// --- tests ---

func TestWebhookPaidFinalizesEverything(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	f := seedWebhookWorld(t, setupWebhookDB(t), "PENDING")
	svc := newRealService(f.db)

	require.NoError(t, svc.HandleInvoiceCallback(context.Background(), f.callback("PAID")))

	var p models.Payment
	require.NoError(t, f.db.First(&p, "id = ?", f.payment.ID).Error)
	require.Equal(t, "PAID", p.Status)
	require.NotNil(t, p.PaidAt)

	var ord models.Order
	require.NoError(t, f.db.First(&ord, "id = ?", f.order.ID).Error)
	require.Equal(t, "PAID", ord.Status)

	var histories []models.OrderStatusHistory
	require.NoError(t, f.db.Find(&histories, "order_id = ?", f.order.ID).Error)
	require.Len(t, histories, 1)
	require.Equal(t, "PAID", histories[0].Status)
	require.NotNil(t, histories[0].Note)
	require.Contains(t, *histories[0].Note, *f.payment.ProviderInvoiceID)

	// Stok: quantity 10-2=8, reserved 2-2=0.
	var inv models.InventoryItem
	require.NoError(t, f.db.First(&inv, "product_variant_id = ?", f.variant.ID).Error)
	require.Equal(t, 8, inv.Quantity)
	require.Equal(t, 0, inv.ReservedQuantity)

	// Jurnal SALE satu baris, referensi order.
	var movements []models.StockMovement
	require.NoError(t, f.db.Find(&movements, "inventory_item_id = ?", inv.ID).Error)
	require.Len(t, movements, 1)
	m := movements[0]
	require.Equal(t, "SALE", m.Type)
	require.Equal(t, -2, m.Quantity)
	require.NotNil(t, m.ReferenceType)
	require.Equal(t, "ORDER", *m.ReferenceType)
	require.NotNil(t, m.ReferenceID)
	require.Equal(t, f.order.ID, *m.ReferenceID)
	require.NotNil(t, m.Note)
	require.Contains(t, *m.Note, f.order.OrderNumber)
}

func TestWebhookPaidReplayIsIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	f := seedWebhookWorld(t, setupWebhookDB(t), "PENDING")
	svc := newRealService(f.db)
	cb := f.callback("PAID")

	require.NoError(t, svc.HandleInvoiceCallback(context.Background(), cb))
	require.NoError(t, svc.HandleInvoiceCallback(context.Background(), cb))

	var inv models.InventoryItem
	require.NoError(t, f.db.First(&inv, "product_variant_id = ?", f.variant.ID).Error)
	require.Equal(t, 8, inv.Quantity, "finalize tidak boleh dobel")
	require.Equal(t, 0, inv.ReservedQuantity)

	var count int64
	require.NoError(t, f.db.Model(&models.StockMovement{}).
		Where("inventory_item_id = ?", inv.ID).Count(&count).Error)
	require.EqualValues(t, 1, count, "jurnal SALE tidak boleh dobel")
}

func TestWebhookExpiredKeepsOrderPending(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	f := seedWebhookWorld(t, setupWebhookDB(t), "PENDING")
	svc := newRealService(f.db)

	require.NoError(t, svc.HandleInvoiceCallback(context.Background(), f.callback("EXPIRED")))

	var p models.Payment
	require.NoError(t, f.db.First(&p, "id = ?", f.payment.ID).Error)
	require.Equal(t, "EXPIRED", p.Status)

	var ord models.Order
	require.NoError(t, f.db.First(&ord, "id = ?", f.order.ID).Error)
	require.Equal(t, "PENDING", ord.Status)

	// Stok tidak tersentuh.
	var inv models.InventoryItem
	require.NoError(t, f.db.First(&inv, "product_variant_id = ?", f.variant.ID).Error)
	require.Equal(t, 10, inv.Quantity)
	require.Equal(t, 2, inv.ReservedQuantity)
}

func TestWebhookPaidAfterCancelledRecordsPayment(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	f := seedWebhookWorld(t, setupWebhookDB(t), "CANCELLED")
	svc := newRealService(f.db)

	require.NoError(t, svc.HandleInvoiceCallback(context.Background(), f.callback("PAID")))

	var p models.Payment
	require.NoError(t, f.db.First(&p, "id = ?", f.payment.ID).Error)
	require.Equal(t, "PAID", p.Status, "payment dicatat sebagai jejak audit")

	var ord models.Order
	require.NoError(t, f.db.First(&ord, "id = ?", f.order.ID).Error)
	require.Equal(t, "CANCELLED", ord.Status, "order tidak ikut PAID")

	var count int64
	require.NoError(t, f.db.Model(&models.StockMovement{}).Count(&count).Error)
	require.EqualValues(t, 0, count, "order tidak PAID = tidak ada jurnal SALE")
}

func TestWebhookInsufficientStockRollsBack(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	f := seedWebhookWorld(t, setupWebhookDB(t), "PENDING")

	// Stok tidak cukup untuk finalisasi: reserved di-nol-kan di luar proses.
	require.NoError(t, f.db.Model(&models.InventoryItem{}).
		Where("product_variant_id = ?", f.variant.ID).
		Update("reserved_quantity", 0).Error)

	svc := newRealService(f.db)
	err := svc.HandleInvoiceCallback(context.Background(), f.callback("PAID"))
	require.Error(t, err)
	require.True(t, errors.Is(err, inventory.ErrorInsufficientStock))

	// Rollback: payment & order tetap belum berubah.
	var p models.Payment
	require.NoError(t, f.db.First(&p, "id = ?", f.payment.ID).Error)
	require.Equal(t, "PENDING", p.Status)

	var count int64
	require.NoError(t, f.db.Model(&models.StockMovement{}).Count(&count).Error)
	require.EqualValues(t, 0, count, "rollback = tidak ada jurnal")

	var ord models.Order
	require.NoError(t, f.db.First(&ord, "id = ?", f.order.ID).Error)
	require.Equal(t, "PENDING", ord.Status)
}
