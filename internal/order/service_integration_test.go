package order

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"ferdinand/ecommerce/internal/cart_items"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"ferdinand/ecommerce/internal/shipping"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func TestOrderServiceUpdateStatusSuccess(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, db, nil, nil, nil, nil, nil)

	user := seedUser(t, db)
	order := models.Order{
		ID:          uuid.New(),
		UserID:      user.ID,
		OrderNumber: "ORD-" + uuid.NewString(),
		Status:      "PENDING",
		Subtotal:    decimal.NewFromInt(100000),
		TotalAmount: decimal.NewFromInt(100000),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to seed order: %v", err)
	}

	note := "payment confirmed"
	updated, err := svc.UpdateStatus(context.Background(), order.ID, OrderStatusPaid, &note)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	if updated.Status != "PAID" {
		t.Fatalf("status = %q, want PAID", updated.Status)
	}

	if len(updated.StatusHistories) != 1 {
		t.Fatalf("histories = %d, want 1", len(updated.StatusHistories))
	}

	if updated.StatusHistories[0].Status != "PAID" || updated.StatusHistories[0].Note == nil || *updated.StatusHistories[0].Note != note {
		t.Fatalf("history = %+v, want PAID with note", updated.StatusHistories[0])
	}

	// rantai transisi berikutnya tetap tercatat
	updated, err = svc.UpdateStatus(context.Background(), order.ID, OrderStatusProcessing, nil)
	if err != nil {
		t.Fatalf("UpdateStatus() second error = %v", err)
	}

	if updated.Status != "PROCESSING" || len(updated.StatusHistories) != 2 {
		t.Fatalf("status = %q, histories = %d, want PROCESSING with 2 histories",
			updated.Status, len(updated.StatusHistories))
	}
}

func TestOrderServiceUpdateStatusRejectsInvalidTransition(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, db, nil, nil, nil, nil, nil)

	user := seedUser(t, db)
	order := models.Order{
		ID:          uuid.New(),
		UserID:      user.ID,
		OrderNumber: "ORD-" + uuid.NewString(),
		Status:      "PENDING",
		Subtotal:    decimal.NewFromInt(100000),
		TotalAmount: decimal.NewFromInt(100000),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to seed order: %v", err)
	}

	_, err := svc.UpdateStatus(context.Background(), order.ID, OrderStatusShipped, nil)
	if !errors.Is(err, ErrorInvalidStatusTransition) {
		t.Fatalf("error = %v, want ErrorInvalidStatusTransition", err)
	}

	current, err := repo.FindByID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if current.Status != "PENDING" {
		t.Fatalf("status = %q, want PENDING (tidak boleh berubah)", current.Status)
	}

	if len(current.StatusHistories) != 0 {
		t.Fatalf("histories = %d, want 0 (tidak boleh ada history)", len(current.StatusHistories))
	}
}

func TestOrderServiceUpdateStatusNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(NewRepository(db), db, nil, nil, nil, nil, nil)

	_, err := svc.UpdateStatus(context.Background(), uuid.New(), OrderStatusPaid, nil)
	if !errors.Is(err, ErrorOrderNotFound) {
		t.Fatalf("error = %v, want ErrorOrderNotFound", err)
	}
}

func seedInventoryItem(t *testing.T, db *gorm.DB, variantID uuid.UUID, quantity int) {
	t.Helper()

	item := models.InventoryItem{
		ID:               uuid.New(),
		ProductVariantID: variantID,
		Quantity:         quantity,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to seed inventory item: %v", err)
	}
}

func TestOrderServiceCreateOrderSuccess(t *testing.T) {
	db := setupTestDB(t)

	user := seedUser(t, db)
	variant := seedVariant(t, db)
	seedInventoryItem(t, db, variant.ID, 10)

	// Data cart sengaja basi: snapshot wajib dari hasil query varian.
	cart := &stubCartService{cart: cartitems.CartResponse{
		Items: []cartitems.CartItemResponse{
			{
				ID:               uuid.New(),
				ProductVariantID: variant.ID,
				Quantity:         2,
				SKU:              "LAMA-SKU",
				Name:             "Nama Lama",
				Price:            decimal.NewFromInt(95000),
				ProductName:      "Produk Lama",
			},
		},
	}}

	ship := &stubShippingService{rates: []shipping.ShippingRate{
		{CourierCode: "jne", CourierName: "JNE", CourierServiceCode: "REG", CourierServiceName: "Regular Service", Price: 20000},
		{CourierCode: "sicepat", CourierName: "SiCepat", CourierServiceCode: "REG", CourierServiceName: "Regular", Price: 15000},
	}}

	svc := NewService(
		NewRepository(db),
		db,
		cart,
		inventory.NewInventoryRepository(db),
		productvariant.NewRepository(db),
		ship,
		inventory.NewStockMovementRepository(db),
	)

	created, err := svc.CreateOrder(context.Background(), user.ID, validCreateOrderRequest())
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	// Format nomor order: ORD-YYYYMMDD-XXXXXXXX
	if !regexp.MustCompile(`^ORD-\d{8}-[0-9A-F]{8}$`).MatchString(created.OrderNumber) {
		t.Fatalf("order number = %q, want ORD-YYYYMMDD-XXXXXXXX", created.OrderNumber)
	}

	if created.Status != "PENDING" {
		t.Fatalf("status = %q, want PENDING", created.Status)
	}

	if created.PaymentExpiresAt == nil {
		t.Fatal("payment_expires_at = nil, want ~30 menit")
	}
	until := time.Until(*created.PaymentExpiresAt)
	if until < 29*time.Minute || until > 31*time.Minute {
		t.Fatalf("payment_expires_at dalam %v, want ~30 menit", until)
	}

	if !created.Subtotal.Equal(decimal.NewFromInt(30000000)) {
		t.Fatalf("subtotal = %s, want 30000000", created.Subtotal)
	}
	if !created.ShippingCost.Equal(decimal.NewFromInt(20000)) {
		t.Fatalf("shipping_cost = %s, want 20000", created.ShippingCost)
	}
	if !created.TotalAmount.Equal(decimal.NewFromInt(30020000)) {
		t.Fatalf("total = %s, want 30020000", created.TotalAmount)
	}

	// Snapshot kurir dari rate yang dihitung ulang, bukan frontend.
	if created.CourierCode != "jne" || created.CourierName != "JNE" ||
		created.CourierServiceCode != "REG" || created.CourierServiceName != "Regular Service" {
		t.Fatalf("kurir = %s/%s/%s/%s, want jne/JNE/REG/Regular Service",
			created.CourierCode, created.CourierName, created.CourierServiceCode, created.CourierServiceName)
	}

	// Item order di-snapshot dari varian terbaru.
	if len(created.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(created.Items))
	}
	item := created.Items[0]
	if item.ProductVariantID != variant.ID ||
		item.ProductName != "laptop" ||
		item.VariantName != "16GB" ||
		item.SKU != variant.SKU ||
		!item.Price.Equal(variant.Price) ||
		item.Quantity != 2 ||
		!item.Subtotal.Equal(decimal.NewFromInt(30000000)) {
		t.Fatalf("item snapshot = %+v, tidak sesuai varian terbaru", item)
	}

	if created.Address == nil ||
		created.Address.RecipientName != "Budi" ||
		created.Address.BiteshipAreaID == nil ||
		*created.Address.BiteshipAreaID != "IDNP3IDNC445" {
		t.Fatalf("address = %+v, tidak sesuai request", created.Address)
	}

	if len(created.StatusHistories) != 1 || created.StatusHistories[0].Status != "PENDING" {
		t.Fatalf("histories = %+v, want satu entry PENDING", created.StatusHistories)
	}

	// Stok terreservasi di DB.
	var inv models.InventoryItem
	if err := db.Where("product_variant_id = ?", variant.ID).First(&inv).Error; err != nil {
		t.Fatalf("load inventory: %v", err)
	}
	if inv.ReservedQuantity != 2 {
		t.Fatalf("reserved_quantity = %d, want 2", inv.ReservedQuantity)
	}

	// Order, item, alamat, history, jurnal RESERVE tersimpan.
	var itemCount, addressCount, historyCount int64
	db.Model(&models.OrderItem{}).Where("order_id = ?", created.ID).Count(&itemCount)
	db.Model(&models.OrderAddress{}).Where("order_id = ?", created.ID).Count(&addressCount)
	db.Model(&models.OrderStatusHistory{}).Where("order_id = ?", created.ID).Count(&historyCount)
	if itemCount != 1 || addressCount != 1 || historyCount != 1 {
		t.Fatalf("counts item/address/history = %d/%d/%d, want 1/1/1", itemCount, addressCount, historyCount)
	}

	var movements []models.StockMovement
	if err := db.Where("inventory_item_id = ?", inv.ID).Find(&movements).Error; err != nil {
		t.Fatalf("load movements: %v", err)
	}
	if len(movements) != 1 {
		t.Fatalf("movements = %d, want 1 RESERVE", len(movements))
	}
	if movements[0].Type != "RESERVE" || movements[0].Quantity != -2 {
		t.Fatalf("movement = %s/%d, want RESERVE/-2", movements[0].Type, movements[0].Quantity)
	}
	if movements[0].ReferenceID == nil || *movements[0].ReferenceID != created.ID {
		t.Fatalf("reference_id = %v, want order id", movements[0].ReferenceID)
	}

	// Cart dikosongkan setelah commit.
	if !cart.cleared {
		t.Fatal("cart harus dikosongkan setelah order dibuat")
	}
}

func TestOrderServiceCreateOrderInsufficientStock(t *testing.T) {
	db := setupTestDB(t)

	user := seedUser(t, db)
	variant := seedVariant(t, db)
	seedInventoryItem(t, db, variant.ID, 1) // stok < qty cart (2)

	cart := &stubCartService{cart: createOrderCart(variant.ID, 2)}
	ship := &stubShippingService{rates: []shipping.ShippingRate{
		{CourierCode: "jne", CourierName: "JNE", CourierServiceCode: "REG", CourierServiceName: "Regular Service", Price: 20000},
	}}

	svc := NewService(
		NewRepository(db),
		db,
		cart,
		inventory.NewInventoryRepository(db),
		productvariant.NewRepository(db),
		ship,
		inventory.NewStockMovementRepository(db),
	)

	_, err := svc.CreateOrder(context.Background(), user.ID, validCreateOrderRequest())
	if !errors.Is(err, inventory.ErrorInsufficientStock) {
		t.Fatalf("err = %v, want ErrorInsufficientStock", err)
	}

	var orderCount int64
	db.Model(&models.Order{}).Where("user_id = ?", user.ID).Count(&orderCount)
	if orderCount != 0 {
		t.Fatalf("orders = %d, want 0 (rollback)", orderCount)
	}

	var inv models.InventoryItem
	if err := db.Where("product_variant_id = ?", variant.ID).First(&inv).Error; err != nil {
		t.Fatalf("load inventory: %v", err)
	}
	if inv.ReservedQuantity != 0 {
		t.Fatalf("reserved_quantity = %d, want 0 (rollback)", inv.ReservedQuantity)
	}

	if cart.cleared {
		t.Fatal("cart tidak boleh dikosongkan saat order gagal")
	}
}

func TestOrderServiceFindStatusHistories(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)
	seedInventoryItem(t, db, variant.ID, 10)

	cart := &stubCartService{cart: createOrderCart(variant.ID, 1)}
	ship := &stubShippingService{rates: []shipping.ShippingRate{
		{CourierCode: "jne", CourierName: "JNE", CourierServiceCode: "REG", CourierServiceName: "Regular", Price: 20000},
	}}

	svc := NewService(
		repo,
		db,
		cart,
		inventory.NewInventoryRepository(db),
		productvariant.NewRepository(db),
		ship,
		inventory.NewStockMovementRepository(db),
	)

	created, err := svc.CreateOrder(context.Background(), user.ID, validCreateOrderRequest())
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	histories, err := svc.FindStatusHistories(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindStatusHistories() error = %v", err)
	}
	if len(histories) != 1 || histories[0].Status != "PENDING" {
		t.Fatalf("histories = %+v, want satu entry PENDING", histories)
	}

	if _, err := svc.FindStatusHistories(context.Background(), uuid.New()); !errors.Is(err, ErrorOrderNotFound) {
		t.Fatalf("err = %v, want ErrorOrderNotFound", err)
	}
}
