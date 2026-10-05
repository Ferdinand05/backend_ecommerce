package inventory

import (
	"context"
	"errors"
	"testing"

	productvariant "ferdinand/ecommerce/internal/product_variant"

	"gorm.io/gorm"
)

func newInventoryService(db *gorm.DB) *service {
	return NewService(
		NewInventoryRepository(db),
		NewStockMovementRepository(db),
		productvariant.NewRepository(db),
		db,
	)
}

func TestInventoryServiceMovementFlow(t *testing.T) {
	db := setupTestDB(t)
	svc := newInventoryService(db)

	category := seedCategory(t, db, "electronics", "electronics")
	product := seedProduct(t, db, category.ID, "laptop", "laptop")
	variant := seedVariant(t, db, product.ID, "LAP-001", "16GB")
	seedInventory(t, db, variant.ID, 5)

	ctx := context.Background()

	restock, err := svc.CreateMovement(ctx, product.ID, variant.ID, CreateStockMovementRequest{
		Type:     StockMovementRestock,
		Quantity: 10,
	})
	if err != nil {
		t.Fatalf("CreateMovement() restock error = %v", err)
	}

	if restock.Quantity != 10 {
		t.Fatalf("restock movement quantity = %d, want 10", restock.Quantity)
	}

	inventory, err := svc.GetInventory(ctx, product.ID, variant.ID)
	if err != nil {
		t.Fatalf("GetInventory() error = %v", err)
	}

	if inventory.Quantity != 15 {
		t.Fatalf("inventory quantity = %d, want 15", inventory.Quantity)
	}

	sale, err := svc.CreateMovement(ctx, product.ID, variant.ID, CreateStockMovementRequest{
		Type:     StockMovementSale,
		Quantity: 4,
	})
	if err != nil {
		t.Fatalf("CreateMovement() sale error = %v", err)
	}

	if sale.Quantity != -4 {
		t.Fatalf("sale movement quantity = %d, want -4", sale.Quantity)
	}

	inventory, err = svc.GetInventory(ctx, product.ID, variant.ID)
	if err != nil {
		t.Fatalf("GetInventory() after sale error = %v", err)
	}

	if inventory.Quantity != 11 {
		t.Fatalf("inventory quantity = %d, want 11", inventory.Quantity)
	}

	movements, err := svc.ListMovements(ctx, product.ID, variant.ID)
	if err != nil {
		t.Fatalf("ListMovements() error = %v", err)
	}

	if len(movements) != 2 || movements[0].Quantity != 10 || movements[1].Quantity != -4 {
		t.Fatalf("ListMovements() = %v, want signed ledger in order", movements)
	}

	detail, err := svc.GetMovement(ctx, product.ID, variant.ID, sale.ID)
	if err != nil {
		t.Fatalf("GetMovement() error = %v", err)
	}

	if detail.ID != sale.ID {
		t.Fatalf("GetMovement() = %v, want %v", detail.ID, sale.ID)
	}
}

func TestInventoryServiceInsufficientStock(t *testing.T) {
	db := setupTestDB(t)
	svc := newInventoryService(db)

	category := seedCategory(t, db, "fashion", "fashion")
	product := seedProduct(t, db, category.ID, "t-shirt", "t-shirt")
	variant := seedVariant(t, db, product.ID, "TSH-M", "size M")
	seedInventory(t, db, variant.ID, 3)

	ctx := context.Background()

	_, err := svc.CreateMovement(ctx, product.ID, variant.ID, CreateStockMovementRequest{
		Type:     StockMovementSale,
		Quantity: 4,
	})
	if !errors.Is(err, ErrorInsufficientStock) {
		t.Fatalf("CreateMovement() error = %v, want %v", err, ErrorInsufficientStock)
	}

	inventory, err := svc.GetInventory(ctx, product.ID, variant.ID)
	if err != nil {
		t.Fatalf("GetInventory() error = %v", err)
	}

	if inventory.Quantity != 3 {
		t.Fatalf("inventory quantity = %d, want unchanged 3", inventory.Quantity)
	}

	movements, err := svc.ListMovements(ctx, product.ID, variant.ID)
	if err != nil {
		t.Fatalf("ListMovements() error = %v", err)
	}

	if len(movements) != 0 {
		t.Fatalf("movements = %d, want 0 (failed sale must not leave a movement)", len(movements))
	}
}

func TestInventoryServiceMovementDirectionByType(t *testing.T) {
	db := setupTestDB(t)
	svc := newInventoryService(db)

	category := seedCategory(t, db, "toys", "toys")
	product := seedProduct(t, db, category.ID, "lego", "lego")
	variant := seedVariant(t, db, product.ID, "LEGO-01", "classic")
	seedInventory(t, db, variant.ID, 10)

	ctx := context.Background()

	cases := []struct {
		movementType StockMovementType
		quantity     int
		want         int
	}{
		{StockMovementRestock, 5, 15},
		{StockMovementReturn, 1, 16},
		{StockMovementOrderCancel, 2, 18},
		{StockMovementAdjustment, 1, 19},
		{StockMovementSale, 4, 15},
		{StockMovementDamage, 2, 13},
	}

	for _, tc := range cases {
		_, err := svc.CreateMovement(ctx, product.ID, variant.ID, CreateStockMovementRequest{
			Type:     tc.movementType,
			Quantity: tc.quantity,
		})
		if err != nil {
			t.Fatalf("CreateMovement() %s error = %v", tc.movementType, err)
		}

		inventory, err := svc.GetInventory(ctx, product.ID, variant.ID)
		if err != nil {
			t.Fatalf("GetInventory() error = %v", err)
		}

		if inventory.Quantity != tc.want {
			t.Fatalf("after %s quantity = %d, want %d", tc.movementType, inventory.Quantity, tc.want)
		}
	}
}
