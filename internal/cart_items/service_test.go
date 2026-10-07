package cartitems

import (
	"context"
	"errors"
	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeRepo struct {
	addQuantityRes       models.CartItem
	addQuantityErr       error
	addQuantityUserID    uuid.UUID
	addQuantityVariantID uuid.UUID
	addQuantityQty       int

	increaseRes    models.CartItem
	increaseErr    error
	increaseUserID uuid.UUID
	increaseItemID uuid.UUID
	increaseQty    int

	decreaseRes    models.CartItem
	decreaseErr    error
	decreaseUserID uuid.UUID
	decreaseItemID uuid.UUID
	decreaseQty    int

	findByIDRes models.CartItem
	findByIDErr error
	findByIDID  uuid.UUID

	findByUserIDRes    []models.CartItem
	findByUserIDErr    error
	findByUserIDUserID uuid.UUID

	deleteErr error
	deleteID  uuid.UUID

	deleteAllErr error
}

func (f *fakeRepo) AddQuantity(ctx context.Context, userID uuid.UUID, variantID uuid.UUID, quantity int) (models.CartItem, error) {
	f.addQuantityUserID = userID
	f.addQuantityVariantID = variantID
	f.addQuantityQty = quantity
	return f.addQuantityRes, f.addQuantityErr
}

func (f *fakeRepo) IncreaseQuantity(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, quantity int) (models.CartItem, error) {
	f.increaseUserID = userID
	f.increaseItemID = itemID
	f.increaseQty = quantity
	return f.increaseRes, f.increaseErr
}

func (f *fakeRepo) DecreaseQuantity(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, quantity int) (models.CartItem, error) {
	f.decreaseUserID = userID
	f.decreaseItemID = itemID
	f.decreaseQty = quantity
	return f.decreaseRes, f.decreaseErr
}

func (f *fakeRepo) FindByID(ctx context.Context, id uuid.UUID) (models.CartItem, error) {
	f.findByIDID = id
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error) {
	f.findByUserIDUserID = userID
	return f.findByUserIDRes, f.findByUserIDErr
}

func (f *fakeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	f.deleteID = id
	return f.deleteErr
}

func (f *fakeRepo) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error {
	return f.deleteAllErr
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

func (f *fakeVariantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func newItem(userID uuid.UUID, variant models.ProductVariant, quantity int) models.CartItem {
	variant.Product = models.Product{
		ID:   variant.ProductID,
		Name: "laptop",
	}

	return models.CartItem{
		ID:               uuid.New(),
		UserID:           userID,
		ProductVariantID: variant.ID,
		Quantity:         quantity,
		ProductVariant:   variant,
	}
}

func TestServiceAddItem(t *testing.T) {
	userID := uuid.New()
	variantID := uuid.New()
	activeVariant := models.ProductVariant{
		ID:       variantID,
		SKU:      "LAP-001",
		Name:     "16GB",
		Price:    decimal.NewFromInt(15000000),
		IsActive: true,
	}

	tests := []struct {
		name        string
		repo        *fakeRepo
		variantRepo *fakeVariantRepo
		want        CartItemResponse
		wantErr     error
	}{
		{
			name:        "success",
			repo:        &fakeRepo{addQuantityRes: newItem(userID, activeVariant, 1)},
			variantRepo: &fakeVariantRepo{findByIDRes: activeVariant},
			want:        CartItemResponse{Quantity: 1, SKU: "LAP-001", ProductName: "laptop"},
		},
		{
			name:        "variant not found",
			repo:        &fakeRepo{},
			variantRepo: &fakeVariantRepo{findByIDErr: productvariant.ErrorProductVariantNotFound},
			wantErr:     productvariant.ErrorProductVariantNotFound,
		},
		{
			name:        "variant inactive",
			repo:        &fakeRepo{},
			variantRepo: &fakeVariantRepo{findByIDRes: models.ProductVariant{ID: variantID, IsActive: false}},
			wantErr:     ErrorProductVariantInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, tt.variantRepo)

			got, err := svc.AddItem(context.Background(), userID, AddCartItemRequest{ProductVariantID: variantID})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("AddItem() error = %v, want %v", err, tt.wantErr)
				}

				if tt.repo.addQuantityVariantID != uuid.Nil {
					t.Fatal("AddItem() must not reach repository on validation failure")
				}

				return
			}

			if tt.repo.addQuantityUserID != userID || tt.repo.addQuantityVariantID != variantID {
				t.Fatalf("AddItem() args = (%v, %v), want (%v, %v)",
					tt.repo.addQuantityUserID, tt.repo.addQuantityVariantID, userID, variantID)
			}

			if tt.repo.addQuantityQty != 1 {
				t.Fatalf("AddItem() quantity = %d, want 1", tt.repo.addQuantityQty)
			}

			if got.SKU != tt.want.SKU || got.ProductName != tt.want.ProductName || got.Quantity != tt.want.Quantity {
				t.Fatalf("AddItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceIncreaseItem(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	variant := models.ProductVariant{ID: uuid.New(), SKU: "LAP-001", Price: decimal.NewFromInt(15000000), IsActive: true}

	tests := []struct {
		name    string
		repo    *fakeRepo
		want    int
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{
				findByIDRes: newItem(userID, variant, 2),
				increaseRes: newItem(userID, variant, 3),
			},
			want: 3,
		},
		{
			name:    "owned by another user",
			repo:    &fakeRepo{findByIDRes: newItem(uuid.New(), variant, 1)},
			wantErr: ErrorCartItemNotFound,
		},
		{
			name:    "not found",
			repo:    &fakeRepo{findByIDErr: ErrorCartItemNotFound},
			wantErr: ErrorCartItemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeVariantRepo{})

			got, err := svc.IncreaseItem(context.Background(), userID, itemID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("IncreaseItem() error = %v, want %v", err, tt.wantErr)
				}

				if tt.repo.increaseItemID != uuid.Nil {
					t.Fatal("IncreaseItem() must not reach repository on validation failure")
				}

				return
			}

			if tt.repo.increaseUserID != userID || tt.repo.increaseItemID != itemID || tt.repo.increaseQty != 1 {
				t.Fatalf("IncreaseItem() args = (%v, %v, %d), want (%v, %v, 1)",
					tt.repo.increaseUserID, tt.repo.increaseItemID, tt.repo.increaseQty, userID, itemID)
			}

			if got.Quantity != tt.want {
				t.Fatalf("IncreaseItem() quantity = %d, want %d", got.Quantity, tt.want)
			}
		})
	}
}

func TestServiceDecreaseItem(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	variant := models.ProductVariant{ID: uuid.New(), SKU: "LAP-001", Price: decimal.NewFromInt(15000000), IsActive: true}

	tests := []struct {
		name       string
		repo       *fakeRepo
		want       int
		wantRemove bool
		wantErr    error
	}{
		{
			name:       "removes item when quantity is one",
			repo:       &fakeRepo{findByIDRes: newItem(userID, variant, 1)},
			wantRemove: true,
		},
		{
			name: "decreases quantity",
			repo: &fakeRepo{
				findByIDRes: newItem(userID, variant, 3),
				decreaseRes: newItem(userID, variant, 2),
			},
			want: 2,
		},
		{
			name:    "owned by another user",
			repo:    &fakeRepo{findByIDRes: newItem(uuid.New(), variant, 1)},
			wantErr: ErrorCartItemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeVariantRepo{})

			got, removed, err := svc.DecreaseItem(context.Background(), userID, itemID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("DecreaseItem() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if removed != tt.wantRemove {
				t.Fatalf("DecreaseItem() removed = %v, want %v", removed, tt.wantRemove)
			}

			if tt.wantRemove {
				if tt.repo.deleteID != itemID {
					t.Fatalf("DecreaseItem() delete id = %v, want %v", tt.repo.deleteID, itemID)
				}

				if tt.repo.decreaseItemID != uuid.Nil {
					t.Fatal("DecreaseItem() must not update quantity when removing")
				}

				return
			}

			if tt.repo.decreaseUserID != userID || tt.repo.decreaseItemID != itemID || tt.repo.decreaseQty != 1 {
				t.Fatalf("DecreaseItem() args = (%v, %v, %d), want (%v, %v, 1)",
					tt.repo.decreaseUserID, tt.repo.decreaseItemID, tt.repo.decreaseQty, userID, itemID)
			}

			if got.Quantity != tt.want {
				t.Fatalf("DecreaseItem() quantity = %d, want %d", got.Quantity, tt.want)
			}
		})
	}
}

func TestServiceRemoveItem(t *testing.T) {
	userID := uuid.New()
	itemID := uuid.New()
	variant := models.ProductVariant{ID: uuid.New(), SKU: "LAP-001", IsActive: true}

	tests := []struct {
		name    string
		repo    *fakeRepo
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{findByIDRes: newItem(userID, variant, 2)},
		},
		{
			name:    "owned by another user",
			repo:    &fakeRepo{findByIDRes: newItem(uuid.New(), variant, 2)},
			wantErr: ErrorCartItemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeVariantRepo{})

			err := svc.RemoveItem(context.Background(), userID, itemID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("RemoveItem() error = %v, want %v", err, tt.wantErr)
				}

				if tt.repo.deleteID != uuid.Nil {
					t.Fatal("RemoveItem() must not reach repository on validation failure")
				}

				return
			}

			if tt.repo.deleteID != itemID {
				t.Fatalf("RemoveItem() delete id = %v, want %v", tt.repo.deleteID, itemID)
			}
		})
	}
}

func TestServiceGetCart(t *testing.T) {
	userID := uuid.New()
	variantA := models.ProductVariant{ID: uuid.New(), SKU: "LAP-001", Price: decimal.NewFromInt(15000000), IsActive: true}
	variantB := models.ProductVariant{ID: uuid.New(), SKU: "LAP-002", Price: decimal.NewFromInt(20000000), IsActive: true}

	dbErr := errors.New("db down")

	tests := []struct {
		name    string
		repo    *fakeRepo
		wantLen int
		wantErr error
	}{
		{
			name: "success",
			repo: &fakeRepo{
				findByUserIDRes: []models.CartItem{
					newItem(userID, variantA, 1),
					newItem(userID, variantB, 2),
				},
			},
			wantLen: 2,
		},
		{
			name:    "repo error",
			repo:    &fakeRepo{findByUserIDErr: dbErr},
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, &fakeVariantRepo{})

			got, err := svc.GetCart(context.Background(), userID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetCart() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if tt.repo.findByUserIDUserID != userID {
				t.Fatalf("GetCart() repo userID = %v, want %v", tt.repo.findByUserIDUserID, userID)
			}

			if len(got.Items) != tt.wantLen {
				t.Fatalf("GetCart() len = %d, want %d", len(got.Items), tt.wantLen)
			}

			if got.Items[0].SKU != "LAP-001" || got.Items[1].SKU != "LAP-002" {
				t.Fatalf("GetCart() = %v, %v, want ordered sku mapping", got.Items[0], got.Items[1])
			}
		})
	}
}
