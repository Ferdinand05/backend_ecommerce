package checkout

import (
	"context"
	"errors"
	"testing"

	cartitems "ferdinand/ecommerce/internal/cart_items"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeCartService struct {
	cart       cartitems.CartResponse
	err        error
	calls      int
	lastUserID uuid.UUID
}

func (f *fakeCartService) GetCart(ctx context.Context, userID uuid.UUID) (cartitems.CartResponse, error) {
	f.calls++
	f.lastUserID = userID
	return f.cart, f.err
}

func (f *fakeCartService) AddItem(ctx context.Context, userID uuid.UUID, req cartitems.AddCartItemRequest) (cartitems.CartItemResponse, error) {
	return cartitems.CartItemResponse{}, errors.New("not implemented")
}

func (f *fakeCartService) IncreaseItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (cartitems.CartItemResponse, error) {
	return cartitems.CartItemResponse{}, errors.New("not implemented")
}

func (f *fakeCartService) DecreaseItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (cartitems.CartItemResponse, bool, error) {
	return cartitems.CartItemResponse{}, false, errors.New("not implemented")
}

func (f *fakeCartService) RemoveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) error {
	return errors.New("not implemented")
}

type fakeInventoryRepo struct {
	item          models.InventoryItem
	err           error
	calls         int
	lastVariantID uuid.UUID
}

func (f *fakeInventoryRepo) FindByVariantID(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error) {
	f.calls++
	f.lastVariantID = variantID
	return f.item, f.err
}

func (f *fakeInventoryRepo) Create(ctx context.Context, item models.InventoryItem) error {
	return errors.New("not implemented")
}

func (f *fakeInventoryRepo) FindByID(ctx context.Context, id uuid.UUID) (models.InventoryItem, error) {
	return models.InventoryItem{}, errors.New("not implemented")
}

func (f *fakeInventoryRepo) FindAll(ctx context.Context) ([]models.InventoryItem, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeInventoryRepo) UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) error {
	return errors.New("not implemented")
}

func (f *fakeInventoryRepo) FindByVariantIDForUpdate(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error) {
	return models.InventoryItem{}, errors.New("not implemented")
}

func (f *fakeInventoryRepo) ReserveStock(ctx context.Context, variantID uuid.UUID, qty int) error {
	return errors.New("not implemented")
}

func (f *fakeInventoryRepo) FinalizeReservation(ctx context.Context, variantID uuid.UUID, qty int) error {
	return errors.New("not implemented")
}

func (f *fakeInventoryRepo) ReleaseReservation(ctx context.Context, variantID uuid.UUID, qty int) error {
	return errors.New("not implemented")
}

type fakeVariantRepo struct {
	variants []models.ProductVariant
	err      error
	calls    int
	lastIDs  []uuid.UUID
}

func (f *fakeVariantRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ProductVariant, error) {
	f.calls++
	f.lastIDs = ids
	return f.variants, f.err
}

func (f *fakeVariantRepo) Create(ctx context.Context, variant models.ProductVariant) error {
	return errors.New("not implemented")
}

func (f *fakeVariantRepo) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeVariantRepo) FindByID(ctx context.Context, id uuid.UUID) (models.ProductVariant, error) {
	return models.ProductVariant{}, errors.New("not implemented")
}

func (f *fakeVariantRepo) FindBySKU(ctx context.Context, sku string) (models.ProductVariant, error) {
	return models.ProductVariant{}, errors.New("not implemented")
}

func (f *fakeVariantRepo) ExistsBySKUExceptID(ctx context.Context, sku string, id uuid.UUID) (bool, error) {
	return false, errors.New("not implemented")
}

func (f *fakeVariantRepo) Update(ctx context.Context, id uuid.UUID, variant models.ProductVariant) error {
	return errors.New("not implemented")
}

func (f *fakeVariantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}

func cartWithSingleItem() (cartitems.CartResponse, uuid.UUID) {
	variantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	cart := cartitems.CartResponse{
		Items: []cartitems.CartItemResponse{
			{
				ID:               uuid.New(),
				ProductVariantID: variantID,
				Quantity:         2,

				// Sengaja beda dari data varian terbaru: preview wajib
				// memakai hasil query varian, bukan data lama di cart.
				SKU:         "LAMA-SKU",
				Name:        "Nama Lama",
				Price:       decimal.NewFromInt(95000),
				ProductName: "Nama Produk Lama",
			},
		},
	}

	return cart, variantID
}

func newTestService(
	cart *fakeCartService,
	inv *fakeInventoryRepo,
	repo *fakeVariantRepo,
) Service {
	return NewService(cart, inv, repo)
}

func TestPreviewSuccess(t *testing.T) {
	cart, variantID := cartWithSingleItem()

	cartSvc := &fakeCartService{cart: cart}
	invSvc := &fakeInventoryRepo{item: models.InventoryItem{Quantity: 5}}
	variantSvc := &fakeVariantRepo{variants: []models.ProductVariant{
		{
			ID:          variantID,
			Name:        "Merah, M",
			SKU:         "KAOS-MERAH-M",
			Price:       decimal.NewFromInt(88000),
			IsActive:    true,
			WeightGrams: 250,
			Product:     models.Product{Name: "Kaos Premium", IsActive: true},
		},
	}}
	svc := newTestService(cartSvc, invSvc, variantSvc)

	got, err := svc.Preview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if len(got.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(got.Items))
	}

	item := got.Items[0]

	// Identitas wajib dari query varian terbaru, bukan data cart.
	if item.ProductName != "Kaos Premium" {
		t.Fatalf("product name = %q, want dari varian repo", item.ProductName)
	}

	if item.VariantName != "Merah, M" || item.SKU != "KAOS-MERAH-M" {
		t.Fatalf("item identitas = %+v, want dari varian repo", item)
	}

	// Harga terkini × quantity: 88000 × 2 (bukan harga lama di cart).
	if !item.UnitPrice.Equal(decimal.NewFromInt(88000)) {
		t.Fatalf("unit price = %s, want 88000 (harga varian terbaru)", item.UnitPrice)
	}

	if !item.Subtotal.Equal(decimal.NewFromInt(176000)) {
		t.Fatalf("item subtotal = %s, want 176000", item.Subtotal)
	}

	if !got.Subtotal.Equal(decimal.NewFromInt(176000)) {
		t.Fatalf("total subtotal = %s, want 176000", got.Subtotal)
	}

	if item.WeightGrams != 250 {
		t.Fatalf("weight = %d, want 250 (dari variant repo)", item.WeightGrams)
	}

	if item.AvailableStock != 5 {
		t.Fatalf("available stock = %d, want 5", item.AvailableStock)
	}
}

func TestPreviewMultipleItemsSumsSubtotal(t *testing.T) {
	variantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	variantB := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	cartSvc := &fakeCartService{cart: cartitems.CartResponse{
		Items: []cartitems.CartItemResponse{
			{ProductVariantID: variantA, Quantity: 2},
			{ProductVariantID: variantB, Quantity: 1},
		},
	}}
	invSvc := &fakeInventoryRepo{item: models.InventoryItem{Quantity: 10}}
	variantSvc := &fakeVariantRepo{variants: []models.ProductVariant{
		{ID: variantA, IsActive: true, WeightGrams: 100, Price: decimal.NewFromInt(50000), Product: models.Product{IsActive: true}},
		{ID: variantB, IsActive: true, WeightGrams: 200, Price: decimal.NewFromInt(30000), Product: models.Product{IsActive: true}},
	}}
	svc := newTestService(cartSvc, invSvc, variantSvc)

	got, err := svc.Preview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	// 2×50000 + 1×30000 = 130000.
	if !got.Subtotal.Equal(decimal.NewFromInt(130000)) {
		t.Fatalf("subtotal = %s, want 130000", got.Subtotal)
	}
}

func TestPreviewEmptyCart(t *testing.T) {
	svc := newTestService(
		&fakeCartService{cart: cartitems.CartResponse{}},
		&fakeInventoryRepo{},
		&fakeVariantRepo{},
	)

	_, err := svc.Preview(context.Background(), uuid.New())
	if !errors.Is(err, ErrCartEmpty) {
		t.Fatalf("error = %v, want ErrCartEmpty", err)
	}
}

func TestPreviewVariantNotAvailable(t *testing.T) {
	_, variantID := cartWithSingleItem()

	tests := []struct {
		name     string
		variants []models.ProductVariant
	}{
		{
			name:     "varian hilang dari database",
			variants: []models.ProductVariant{},
		},
		{
			name: "varian nonaktif",
			variants: []models.ProductVariant{
				{ID: variantID, IsActive: false, WeightGrams: 250},
			},
		},
		{
			name: "produk induk nonaktif",
			variants: []models.ProductVariant{
				{ID: variantID, IsActive: true, WeightGrams: 250, Product: models.Product{IsActive: false}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(
				&fakeCartService{cart: mustCart()},
				&fakeInventoryRepo{item: models.InventoryItem{Quantity: 5}},
				&fakeVariantRepo{variants: tt.variants},
			)

			_, err := svc.Preview(context.Background(), uuid.New())
			if !errors.Is(err, ErrVariantNotAvailable) {
				t.Fatalf("error = %v, want ErrVariantNotAvailable", err)
			}
		})
	}
}

func TestPreviewAvailableStockSubtractsReserved(t *testing.T) {
	cart, variantID := cartWithSingleItem()

	svc := newTestService(
		&fakeCartService{cart: cart},
		&fakeInventoryRepo{item: models.InventoryItem{Quantity: 5, ReservedQuantity: 3}},
		&fakeVariantRepo{variants: []models.ProductVariant{
			{ID: variantID, IsActive: true, WeightGrams: 250, Product: models.Product{IsActive: true}},
		}},
	)

	got, err := svc.Preview(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	// 5 fisik - 3 reservasi = 2 tersedia; qty cart 2 masih muat.
	if got.Items[0].AvailableStock != 2 {
		t.Fatalf("available stock = %d, want 2 (fisik - reservasi)", got.Items[0].AvailableStock)
	}
}

func TestPreviewFailsWhenReservedEatsStock(t *testing.T) {
	_, variantID := cartWithSingleItem()

	svc := newTestService(
		&fakeCartService{cart: mustCart()},
		&fakeInventoryRepo{item: models.InventoryItem{Quantity: 5, ReservedQuantity: 4}},
		&fakeVariantRepo{variants: []models.ProductVariant{
			{ID: variantID, IsActive: true, WeightGrams: 250, Product: models.Product{IsActive: true}},
		}},
	)

	// Fisik 5 cukup, tapi tersedia hanya 1 < qty 2.
	_, err := svc.Preview(context.Background(), uuid.New())
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("error = %v, want ErrInsufficientStock", err)
	}
}

func TestPreviewInsufficientStock(t *testing.T) {
	_, variantID := cartWithSingleItem()

	tests := []struct {
		name string
		inv  *fakeInventoryRepo
	}{
		{
			name: "stok kurang",
			inv:  &fakeInventoryRepo{item: models.InventoryItem{Quantity: 1}},
		},
		{
			name: "inventory tidak ditemukan",
			inv:  &fakeInventoryRepo{err: inventory.ErrorInventoryNotFound},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(
				&fakeCartService{cart: mustCart()},
				tt.inv,
				&fakeVariantRepo{variants: []models.ProductVariant{
					{ID: variantID, IsActive: true, WeightGrams: 250, Product: models.Product{IsActive: true}},
				}},
			)

			_, err := svc.Preview(context.Background(), uuid.New())
			if !errors.Is(err, ErrInsufficientStock) {
				t.Fatalf("error = %v, want ErrInsufficientStock", err)
			}
		})
	}
}

func TestPreviewDatabaseFailurePassesThrough(t *testing.T) {
	dbErr := errors.New("db down")

	tests := []struct {
		name string
		svc  Service
	}{
		{
			name: "cart gagal",
			svc: newTestService(
				&fakeCartService{err: dbErr},
				&fakeInventoryRepo{},
				&fakeVariantRepo{},
			),
		},
		{
			name: "variant repo gagal",
			svc: newTestService(
				&fakeCartService{cart: mustCart()},
				&fakeInventoryRepo{},
				&fakeVariantRepo{err: dbErr},
			),
		},
		{
			name: "inventory gagal",
			svc: newTestService(
				&fakeCartService{cart: mustCart()},
				&fakeInventoryRepo{err: dbErr},
				&fakeVariantRepo{variants: []models.ProductVariant{
					{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), IsActive: true, Product: models.Product{IsActive: true}},
				}},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.Preview(context.Background(), uuid.New())
			if !errors.Is(err, dbErr) {
				t.Fatalf("error = %v, want pass-through db error", err)
			}
		})
	}
}

func mustCart() cartitems.CartResponse {
	cart, _ := cartWithSingleItem()
	return cart
}
