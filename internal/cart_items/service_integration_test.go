package cartitems

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func newCartService(db *gorm.DB) *service {
	return NewService(NewRepository(db), productvariant.NewRepository(db))
}

func TestCartServiceFlow(t *testing.T) {
	db := setupTestDB(t)
	svc := newCartService(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)

	added, err := svc.AddItem(context.Background(), user.ID, AddCartItemRequest{ProductVariantID: variant.ID})
	if err != nil {
		t.Fatalf("AddItem() error = %v", err)
	}

	if added.Quantity != 1 || added.SKU != variant.SKU || added.ProductName != "laptop" {
		t.Fatalf("AddItem() = %v, want quantity 1 with variant and product data", added)
	}

	increased, err := svc.IncreaseItem(context.Background(), user.ID, added.ID)
	if err != nil {
		t.Fatalf("IncreaseItem() error = %v", err)
	}

	if increased.Quantity != 2 {
		t.Fatalf("IncreaseItem() quantity = %d, want 2", increased.Quantity)
	}

	_, removed, err := svc.DecreaseItem(context.Background(), user.ID, added.ID)
	if err != nil {
		t.Fatalf("DecreaseItem() error = %v", err)
	}

	if removed {
		t.Fatal("DecreaseItem() removed = true, want false while quantity > 1")
	}

	_, removed, err = svc.DecreaseItem(context.Background(), user.ID, added.ID)
	if err != nil {
		t.Fatalf("DecreaseItem() at quantity 1 error = %v", err)
	}

	if !removed {
		t.Fatal("DecreaseItem() removed = false, want true at quantity 1")
	}

	cart, err := svc.GetCart(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetCart() error = %v", err)
	}

	if len(cart.Items) != 0 {
		t.Fatalf("cart items = %d, want 0 after removal", len(cart.Items))
	}

	if _, err := svc.IncreaseItem(context.Background(), user.ID, added.ID); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("IncreaseItem() after removal error = %v, want %v", err, ErrorCartItemNotFound)
	}
}

func TestCartServiceAddItemRejectsInactiveAndMissingVariant(t *testing.T) {
	db := setupTestDB(t)
	svc := newCartService(db)

	user := seedUser(t, db)
	variant := seedVariant(t, db)

	if err := db.Model(&models.ProductVariant{}).Where("id = ?", variant.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate variant error = %v", err)
	}

	if _, err := svc.AddItem(context.Background(), user.ID, AddCartItemRequest{ProductVariantID: variant.ID}); !errors.Is(err, ErrorProductVariantInactive) {
		t.Fatalf("AddItem() inactive error = %v, want %v", err, ErrorProductVariantInactive)
	}

	if _, err := svc.AddItem(context.Background(), user.ID, AddCartItemRequest{ProductVariantID: uuid.New()}); !errors.Is(err, productvariant.ErrorProductVariantNotFound) {
		t.Fatalf("AddItem() missing variant error = %v, want %v", err, productvariant.ErrorProductVariantNotFound)
	}
}

func TestCartServiceCartsSeparatedByUser(t *testing.T) {
	db := setupTestDB(t)
	svc := newCartService(db)

	userA := seedUser(t, db)
	userB := seedUser(t, db)
	variant := seedVariant(t, db)

	itemA, err := svc.AddItem(context.Background(), userA.ID, AddCartItemRequest{ProductVariantID: variant.ID})
	if err != nil {
		t.Fatalf("AddItem() user A error = %v", err)
	}

	if _, err := svc.AddItem(context.Background(), userB.ID, AddCartItemRequest{ProductVariantID: variant.ID}); err != nil {
		t.Fatalf("AddItem() user B error = %v", err)
	}

	increased, err := svc.IncreaseItem(context.Background(), userA.ID, itemA.ID)
	if err != nil {
		t.Fatalf("IncreaseItem() user A error = %v", err)
	}

	if increased.Quantity != 2 {
		t.Fatalf("user A quantity = %d, want 2", increased.Quantity)
	}

	cartB, err := svc.GetCart(context.Background(), userB.ID)
	if err != nil {
		t.Fatalf("GetCart() user B error = %v", err)
	}

	if len(cartB.Items) != 1 || cartB.Items[0].Quantity != 1 {
		t.Fatalf("user B cart = %v, want single untouched item", cartB.Items)
	}

	if _, err := svc.IncreaseItem(context.Background(), userA.ID, cartB.Items[0].ID); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("IncreaseItem() on other user's item error = %v, want %v", err, ErrorCartItemNotFound)
	}

	if err := svc.RemoveItem(context.Background(), userA.ID, cartB.Items[0].ID); !errors.Is(err, ErrorCartItemNotFound) {
		t.Fatalf("RemoveItem() on other user's item error = %v, want %v", err, ErrorCartItemNotFound)
	}
}
