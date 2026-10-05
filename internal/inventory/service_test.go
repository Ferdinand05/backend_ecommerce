package inventory

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"testing"

	"github.com/google/uuid"
)

type fakeItemRepo struct {
	createErr error
	createArg models.InventoryItem

	findByIDRes models.InventoryItem
	findByIDErr error

	findByVariantRes        models.InventoryItem
	findByVariantErr        error
	findByVariantArgVariant uuid.UUID

	findAllRes []models.InventoryItem
	findAllErr error

	updateErr      error
	updateID       uuid.UUID
	updateQuantity int

	forUpdateRes models.InventoryItem
	forUpdateErr error
}

func (f *fakeItemRepo) Create(ctx context.Context, item models.InventoryItem) error {
	f.createArg = item
	return f.createErr
}

func (f *fakeItemRepo) FindByID(ctx context.Context, id uuid.UUID) (models.InventoryItem, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeItemRepo) FindByVariantID(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error) {
	f.findByVariantArgVariant = variantID
	return f.findByVariantRes, f.findByVariantErr
}

func (f *fakeItemRepo) FindAll(ctx context.Context) ([]models.InventoryItem, error) {
	return f.findAllRes, f.findAllErr
}

func (f *fakeItemRepo) UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) error {
	f.updateID = id
	f.updateQuantity = quantity
	return f.updateErr
}

func (f *fakeItemRepo) FindByVariantIDForUpdate(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error) {
	f.findByVariantArgVariant = variantID
	return f.forUpdateRes, f.forUpdateErr
}

type fakeMovementRepo struct {
	createErr error
	createArg models.StockMovement

	findByIDRes models.StockMovement
	findByIDErr error

	findAllRes   []models.StockMovement
	findAllErr   error
	findAllInvID uuid.UUID
}

func (f *fakeMovementRepo) Create(ctx context.Context, movement models.StockMovement) error {
	f.createArg = movement
	return f.createErr
}

func (f *fakeMovementRepo) FindByID(ctx context.Context, id uuid.UUID) (models.StockMovement, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeMovementRepo) FindAllByInventoryID(ctx context.Context, inventoryID uuid.UUID) ([]models.StockMovement, error) {
	f.findAllInvID = inventoryID
	return f.findAllRes, f.findAllErr
}

type fakeVariantRepo struct {
	findByIDRes models.ProductVariant
	findByIDErr error
}

func (f *fakeVariantRepo) Create(ctx context.Context, variant models.ProductVariant) error {
	return nil
}
func (f *fakeVariantRepo) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error) {
	return nil, nil
}
func (f *fakeVariantRepo) FindByID(ctx context.Context, id uuid.UUID) (models.ProductVariant, error) {
	return f.findByIDRes, f.findByIDErr
}
func (f *fakeVariantRepo) FindBySKU(ctx context.Context, sku string) (models.ProductVariant, error) {
	return models.ProductVariant{}, nil
}
func (f *fakeVariantRepo) ExistsBySKUExceptID(ctx context.Context, sku string, id uuid.UUID) (bool, error) {
	return false, nil
}
func (f *fakeVariantRepo) Update(ctx context.Context, id uuid.UUID, variant models.ProductVariant) error {
	return nil
}
func (f *fakeVariantRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func TestServiceGetInventory(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()
	itemID := uuid.New()

	tests := []struct {
		name    string
		item    *fakeItemRepo
		variant *fakeVariantRepo
		wantQty int
		wantErr error
	}{
		{
			name:    "success",
			item:    &fakeItemRepo{findByVariantRes: models.InventoryItem{ID: itemID, ProductVariantID: variantID, Quantity: 12}},
			variant: &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: productID}},
			wantQty: 12,
		},
		{
			name:    "variant belongs to another product",
			item:    &fakeItemRepo{},
			variant: &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: uuid.New()}},
			wantErr: productvariant.ErrorProductVariantNotFound,
		},
		{
			name:    "variant not found",
			item:    &fakeItemRepo{},
			variant: &fakeVariantRepo{findByIDErr: productvariant.ErrorProductVariantNotFound},
			wantErr: productvariant.ErrorProductVariantNotFound,
		},
		{
			name:    "inventory not found",
			item:    &fakeItemRepo{findByVariantErr: ErrorInventoryNotFound},
			variant: &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: productID}},
			wantErr: ErrorInventoryNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.item, &fakeMovementRepo{}, tt.variant, nil)

			got, err := svc.GetInventory(context.Background(), productID, variantID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetInventory() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if got.Quantity != tt.wantQty || tt.item.findByVariantArgVariant != variantID {
				t.Fatalf("GetInventory() = %v, want quantity %d", got, tt.wantQty)
			}
		})
	}
}

func TestServiceListMovements(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()
	itemID := uuid.New()

	item := &fakeItemRepo{findByVariantRes: models.InventoryItem{ID: itemID, ProductVariantID: variantID}}
	movement := &fakeMovementRepo{findAllRes: []models.StockMovement{
		{ID: uuid.New(), InventoryItemID: itemID, Type: "RESTOCK", Quantity: 5},
		{ID: uuid.New(), InventoryItemID: itemID, Type: "SALE", Quantity: -2},
	}}
	variant := &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: productID}}

	svc := NewService(item, movement, variant, nil)

	got, err := svc.ListMovements(context.Background(), productID, variantID)
	if err != nil {
		t.Fatalf("ListMovements() error = %v", err)
	}

	if movement.findAllInvID != itemID {
		t.Fatalf("ListMovements() inventory id = %v, want %v", movement.findAllInvID, itemID)
	}

	if len(got) != 2 || got[1].Quantity != -2 {
		t.Fatalf("ListMovements() = %v, want 2 movements with signed quantity", got)
	}
}

func TestServiceGetMovement(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()
	itemID := uuid.New()
	movementID := uuid.New()

	tests := []struct {
		name     string
		movement *fakeMovementRepo
		wantErr  error
	}{
		{
			name:     "success",
			movement: &fakeMovementRepo{findByIDRes: models.StockMovement{ID: movementID, InventoryItemID: itemID}},
		},
		{
			name:     "movement belongs to another inventory",
			movement: &fakeMovementRepo{findByIDRes: models.StockMovement{ID: movementID, InventoryItemID: uuid.New()}},
			wantErr:  ErrorStockMovementNotFound,
		},
		{
			name:     "movement not found",
			movement: &fakeMovementRepo{findByIDErr: ErrorStockMovementNotFound},
			wantErr:  ErrorStockMovementNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &fakeItemRepo{findByVariantRes: models.InventoryItem{ID: itemID, ProductVariantID: variantID}}
			variant := &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: productID}}
			svc := NewService(item, tt.movement, variant, nil)

			got, err := svc.GetMovement(context.Background(), productID, variantID, movementID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetMovement() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if got.ID != movementID {
				t.Fatalf("GetMovement() = %v, want %v", got.ID, movementID)
			}
		})
	}
}

func TestServiceCreateMovementChecks(t *testing.T) {
	productID := uuid.New()
	variantID := uuid.New()

	tests := []struct {
		name    string
		req     CreateStockMovementRequest
		variant *fakeVariantRepo
		wantErr error
	}{
		{
			name:    "invalid quantity",
			req:     CreateStockMovementRequest{Type: StockMovementRestock, Quantity: 0},
			variant: &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: productID}},
			wantErr: ErrorInvalidMovementQuantity,
		},
		{
			name:    "invalid type",
			req:     CreateStockMovementRequest{Type: "UNKNOWN", Quantity: 1},
			variant: &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: productID}},
			wantErr: ErrorInvalidMovementType,
		},
		{
			name:    "variant belongs to another product",
			req:     CreateStockMovementRequest{Type: StockMovementRestock, Quantity: 1},
			variant: &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, ProductID: uuid.New()}},
			wantErr: productvariant.ErrorProductVariantNotFound,
		},
		{
			name:    "variant not found",
			req:     CreateStockMovementRequest{Type: StockMovementRestock, Quantity: 1},
			variant: &fakeVariantRepo{findByIDErr: productvariant.ErrorProductVariantNotFound},
			wantErr: productvariant.ErrorProductVariantNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &fakeItemRepo{}
			svc := NewService(item, &fakeMovementRepo{}, tt.variant, nil)

			_, err := svc.CreateMovement(context.Background(), productID, variantID, tt.req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateMovement() error = %v, want %v", err, tt.wantErr)
			}

			if item.updateID != uuid.Nil {
				t.Fatal("CreateMovement() must not reach repository on validation failure")
			}
		})
	}
}
