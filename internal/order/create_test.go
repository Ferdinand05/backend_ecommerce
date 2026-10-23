package order

import (
	"context"
	"errors"
	"testing"

	cartitems "ferdinand/ecommerce/internal/cart_items"
	"ferdinand/ecommerce/internal/inventory"
	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"ferdinand/ecommerce/internal/shipping"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// --- fakes untuk unit test CreateOrder (transaksi DB tidak tersentuh) ---

type stubCartService struct {
	cart       cartitems.CartResponse
	getErr     error
	cleared    bool
	clearCalls int
}

func (s *stubCartService) GetCart(ctx context.Context, userID uuid.UUID) (cartitems.CartResponse, error) {
	return s.cart, s.getErr
}

func (s *stubCartService) ClearCart(ctx context.Context, userID uuid.UUID) error {
	s.cleared = true
	s.clearCalls++
	return nil
}

func (s *stubCartService) AddItem(ctx context.Context, userID uuid.UUID, req cartitems.AddCartItemRequest) (cartitems.CartItemResponse, error) {
	return cartitems.CartItemResponse{}, errors.New("not implemented")
}

func (s *stubCartService) IncreaseItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (cartitems.CartItemResponse, error) {
	return cartitems.CartItemResponse{}, errors.New("not implemented")
}

func (s *stubCartService) DecreaseItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (cartitems.CartItemResponse, bool, error) {
	return cartitems.CartItemResponse{}, false, errors.New("not implemented")
}

func (s *stubCartService) RemoveItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) error {
	return errors.New("not implemented")
}

type stubInventoryRepo struct {
	locked     []uuid.UUID
	reserved   map[uuid.UUID]int
	reserveErr error
}

func newStubInventoryRepo() *stubInventoryRepo {
	return &stubInventoryRepo{reserved: make(map[uuid.UUID]int)}
}

func (s *stubInventoryRepo) Create(ctx context.Context, item models.InventoryItem) error {
	return errors.New("not implemented")
}

func (s *stubInventoryRepo) FindByID(ctx context.Context, id uuid.UUID) (models.InventoryItem, error) {
	return models.InventoryItem{}, errors.New("not implemented")
}

func (s *stubInventoryRepo) FindByVariantID(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error) {
	return models.InventoryItem{}, errors.New("not implemented")
}

func (s *stubInventoryRepo) FindAll(ctx context.Context) ([]models.InventoryItem, error) {
	return nil, errors.New("not implemented")
}

func (s *stubInventoryRepo) UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) error {
	return errors.New("not implemented")
}

func (s *stubInventoryRepo) FindByVariantIDForUpdate(ctx context.Context, variantID uuid.UUID) (models.InventoryItem, error) {
	s.locked = append(s.locked, variantID)
	return models.InventoryItem{}, nil
}

func (s *stubInventoryRepo) ReserveStock(ctx context.Context, variantID uuid.UUID, qty int) error {
	if s.reserveErr != nil {
		return s.reserveErr
	}
	s.reserved[variantID] += qty
	return nil
}

func (s *stubInventoryRepo) FinalizeReservation(ctx context.Context, variantID uuid.UUID, qty int) error {
	return errors.New("not implemented")
}

func (s *stubInventoryRepo) ReleaseReservation(ctx context.Context, variantID uuid.UUID, qty int) error {
	return errors.New("not implemented")
}

type stubVariantRepo struct {
	variants []models.ProductVariant
	err      error
}

func (s *stubVariantRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ProductVariant, error) {
	return s.variants, s.err
}

func (s *stubVariantRepo) Create(ctx context.Context, variant models.ProductVariant) error {
	return errors.New("not implemented")
}

func (s *stubVariantRepo) FindAllByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductVariant, error) {
	return nil, errors.New("not implemented")
}

func (s *stubVariantRepo) FindByID(ctx context.Context, id uuid.UUID) (models.ProductVariant, error) {
	return models.ProductVariant{}, errors.New("not implemented")
}

func (s *stubVariantRepo) FindBySKU(ctx context.Context, sku string) (models.ProductVariant, error) {
	return models.ProductVariant{}, errors.New("not implemented")
}

func (s *stubVariantRepo) ExistsBySKUExceptID(ctx context.Context, sku string, id uuid.UUID) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *stubVariantRepo) Update(ctx context.Context, id uuid.UUID, variant models.ProductVariant) error {
	return errors.New("not implemented")
}

func (s *stubVariantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}

type stubShippingService struct {
	rates   []shipping.ShippingRate
	err     error
	lastReq shipping.CalculateRatesRequest
}

func (s *stubShippingService) SearchAreas(ctx context.Context, query string) ([]shipping.Area, error) {
	return nil, errors.New("not implemented")
}

func (s *stubShippingService) CalculateRates(ctx context.Context, req shipping.CalculateRatesRequest) ([]shipping.ShippingRate, error) {
	s.lastReq = req
	return s.rates, s.err
}

// --- helpers ---

func activeVariant(id uuid.UUID) models.ProductVariant {
	return models.ProductVariant{
		ID:        id,
		ProductID: uuid.New(),
		SKU:       "SKU-" + uuid.NewString(),
		Name:      "16GB",
		Price:     decimal.NewFromInt(150000),
		IsActive:  true,
		Product: models.Product{
			ID:       id, // tidak dipakai untuk banding; hanya Name
			Name:     "laptop",
			IsActive: true,
		},
	}
}

func createOrderCart(variantID uuid.UUID, qty int) cartitems.CartResponse {
	return cartitems.CartResponse{
		Items: []cartitems.CartItemResponse{
			{
				ID:               uuid.New(),
				ProductVariantID: variantID,
				Quantity:         qty,
			},
		},
	}
}

func createOrderService(cart *stubCartService, inv inventory.InventoryRepository, variants productvariant.Repository, ship *stubShippingService) *service {
	return NewService(&fakeRepo{}, nil, cart, inv, variants, ship, nil)
}

// --- tests (jalur sebelum transaksi; db nil tidak boleh disentuh) ---

func TestCreateOrderRejectsNonPositiveQuantity(t *testing.T) {
	for _, qty := range []int{0, -1} {
		cart := &stubCartService{cart: createOrderCart(uuid.New(), qty)}
		svc := createOrderService(cart, newStubInventoryRepo(), &stubVariantRepo{}, &stubShippingService{})

		_, err := svc.CreateOrder(context.Background(), uuid.New(), validCreateOrderRequest())
		if !errors.Is(err, ErrorInvalidOrderInput) {
			t.Fatalf("qty %d: err = %v, want ErrorInvalidOrderInput", qty, err)
		}
	}
}

func TestCreateOrderEmptyCart(t *testing.T) {
	cart := &stubCartService{cart: cartitems.CartResponse{}}
	svc := createOrderService(cart, newStubInventoryRepo(), &stubVariantRepo{}, &stubShippingService{})

	_, err := svc.CreateOrder(context.Background(), uuid.New(), validCreateOrderRequest())
	if !errors.Is(err, ErrorEmptyCart) {
		t.Fatalf("err = %v, want ErrorEmptyCart", err)
	}
}

func TestCreateOrderVariantUnavailable(t *testing.T) {
	variantID := uuid.New()

	tests := []struct {
		name     string
		variants []models.ProductVariant
	}{
		{name: "varian tidak ditemukan", variants: nil},
		{name: "varian nonaktif", variants: func() []models.ProductVariant {
			v := activeVariant(variantID)
			v.IsActive = false
			return []models.ProductVariant{v}
		}()},
		{name: "produk nonaktif", variants: func() []models.ProductVariant {
			v := activeVariant(variantID)
			v.Product.IsActive = false
			return []models.ProductVariant{v}
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cart := &stubCartService{cart: createOrderCart(variantID, 1)}
			svc := createOrderService(cart, newStubInventoryRepo(), &stubVariantRepo{variants: tt.variants}, &stubShippingService{})

			_, err := svc.CreateOrder(context.Background(), uuid.New(), validCreateOrderRequest())
			if !errors.Is(err, ErrorVariantUnavailable) {
				t.Fatalf("err = %v, want ErrorVariantUnavailable", err)
			}
		})
	}
}

func TestCreateOrderShippingRateNotFound(t *testing.T) {
	variantID := uuid.New()
	cart := &stubCartService{cart: createOrderCart(variantID, 2)}

	ship := &stubShippingService{rates: []shipping.ShippingRate{
		{CourierCode: "jne", CourierServiceCode: "YES", Price: 20000},
	}}

	svc := createOrderService(
		cart,
		newStubInventoryRepo(),
		&stubVariantRepo{variants: []models.ProductVariant{activeVariant(variantID)}},
		ship,
	)

	_, err := svc.CreateOrder(context.Background(), uuid.New(), validCreateOrderRequest())
	if !errors.Is(err, ErrorShippingRateNotFound) {
		t.Fatalf("err = %v, want ErrorShippingRateNotFound", err)
	}

	// ongkir dihitung ulang dengan area tujuan + item cart
	if ship.lastReq.DestinationAreaID != "IDNP3IDNC445" {
		t.Fatalf("destination area = %q, want IDNP3IDNC445", ship.lastReq.DestinationAreaID)
	}
	if len(ship.lastReq.Items) != 1 ||
		ship.lastReq.Items[0].ProductVariantID != variantID.String() ||
		ship.lastReq.Items[0].Quantity != 2 {
		t.Fatalf("rate items = %+v, want satu item qty 2", ship.lastReq.Items)
	}
}

func validCreateOrderRequest() CreateOrderRequest {
	return CreateOrderRequest{
		RecipientName:       "Budi",
		Phone:               "081234567890",
		Address:             "Jl. Mawar 1",
		City:                "Bandung",
		Province:            "Jawa Barat",
		PostalCode:          "40123",
		DestinationAreaID:   "IDNP3IDNC445",
		DestinationAreaName: "Coblong, Bandung",
		CourierCode:         "jne",
		CourierServiceCode:  "REG",
	}
}
