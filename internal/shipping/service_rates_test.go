package shipping

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

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

func validVariants() []models.ProductVariant {
	idA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	idB := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	return []models.ProductVariant{
		{ID: idA, Name: "Kaos Polos", Price: decimal.NewFromFloat(199000.75), IsActive: true, WeightGrams: 250},
		{ID: idB, Name: "Kaos Lengan Panjang", Price: decimal.NewFromInt(299000), IsActive: true, WeightGrams: 400},
	}
}

func newRatesService(repo *fakeVariantRepo, provider *fakeProvider) *service {
	return NewService(&fakeCache{data: map[string][]byte{}}, provider, repo, "ORIGIN-AREA", "jne,sicepat")
}

func validRatesRequest() CalculateRatesRequest {
	return CalculateRatesRequest{
		DestinationAreaID: "DEST-AREA",
		Items: []ShippingRateItem{
			{ProductVariantID: "11111111-1111-1111-1111-111111111111", Quantity: 3},
		},
	}
}

func TestCalculateRatesSuccess(t *testing.T) {
	repo := &fakeVariantRepo{variants: validVariants()}
	provider := &fakeProvider{ratesRes: []ShippingRate{{CourierCode: "jne", Price: 18000}}}
	svc := newRatesService(repo, provider)

	got, err := svc.CalculateRates(context.Background(), validRatesRequest())
	if err != nil {
		t.Fatalf("CalculateRates() error = %v", err)
	}

	if len(got) != 1 || got[0].CourierCode != "jne" {
		t.Fatalf("rates = %v, want jne", got)
	}

	if repo.calls != 1 {
		t.Fatalf("repo calls = %d, want 1 (satu query)", repo.calls)
	}

	if provider.ratesCalls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.ratesCalls)
	}

	// Origin + couriers berasal dari config, bukan request.
	if provider.lastRatesReq.OriginAreaID != "ORIGIN-AREA" {
		t.Fatalf("origin = %q, want ORIGIN-AREA", provider.lastRatesReq.OriginAreaID)
	}

	if provider.lastRatesReq.Couriers != "jne,sicepat" {
		t.Fatalf("couriers = %q, want config value", provider.lastRatesReq.Couriers)
	}

	if provider.lastRatesReq.DestinationAreaID != "DEST-AREA" {
		t.Fatalf("destination = %q, want DEST-AREA", provider.lastRatesReq.DestinationAreaID)
	}

	// Item dibentuk dari DB: nama, harga, berat per unit, quantity.
	items := provider.lastRatesReq.Items
	if len(items) != 1 {
		t.Fatalf("provider items len = %d, want 1", len(items))
	}

	if items[0].Name != "Kaos Polos" {
		t.Fatalf("item name = %q, want Kaos Polos (dari DB)", items[0].Name)
	}

	// 199000.75 dibulatkan ke 199001 — bukan dipotong jadi 199000.
	if items[0].Value != 199001 {
		t.Fatalf("item value = %d, want 199001 (Round, bukan truncate)", items[0].Value)
	}

	// Berat per unit dikirim apa adanya; Biteship yang mengalikan dengan quantity.
	if items[0].Weight != 250 {
		t.Fatalf("item weight = %d, want 250 (per unit, tanpa pre-multiply)", items[0].Weight)
	}

	if items[0].Quantity != 3 {
		t.Fatalf("item quantity = %d, want 3", items[0].Quantity)
	}
}

func TestCalculateRatesMultipleItems(t *testing.T) {
	repo := &fakeVariantRepo{variants: validVariants()}
	provider := &fakeProvider{ratesRes: []ShippingRate{}}
	svc := newRatesService(repo, provider)

	req := CalculateRatesRequest{
		DestinationAreaID: "DEST-AREA",
		Items: []ShippingRateItem{
			{ProductVariantID: "11111111-1111-1111-1111-111111111111", Quantity: 2},
			{ProductVariantID: "22222222-2222-2222-2222-222222222222", Quantity: 1},
		},
	}

	if _, err := svc.CalculateRates(context.Background(), req); err != nil {
		t.Fatalf("CalculateRates() error = %v", err)
	}

	if len(provider.lastRatesReq.Items) != 2 {
		t.Fatalf("provider items len = %d, want 2", len(provider.lastRatesReq.Items))
	}
}

func TestCalculateRatesInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		req  CalculateRatesRequest
	}{
		{
			name: "destination kosong",
			req: CalculateRatesRequest{
				DestinationAreaID: "   ",
				Items:             []ShippingRateItem{{ProductVariantID: uuid.NewString(), Quantity: 1}},
			},
		},
		{
			name: "items kosong",
			req: CalculateRatesRequest{
				DestinationAreaID: "DEST-AREA",
				Items:             []ShippingRateItem{},
			},
		},
		{
			name: "variant id bukan uuid",
			req: CalculateRatesRequest{
				DestinationAreaID: "DEST-AREA",
				Items:             []ShippingRateItem{{ProductVariantID: "bukan-uuid", Quantity: 1}},
			},
		},
		{
			name: "quantity nol",
			req: CalculateRatesRequest{
				DestinationAreaID: "DEST-AREA",
				Items:             []ShippingRateItem{{ProductVariantID: uuid.NewString(), Quantity: 0}},
			},
		},
		{
			name: "variant id duplikat",
			req: CalculateRatesRequest{
				DestinationAreaID: "DEST-AREA",
				Items: []ShippingRateItem{
					{ProductVariantID: "11111111-1111-1111-1111-111111111111", Quantity: 1},
					{ProductVariantID: "11111111-1111-1111-1111-111111111111", Quantity: 2},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &fakeProvider{}
			svc := newRatesService(&fakeVariantRepo{}, provider)

			_, err := svc.CalculateRates(context.Background(), tt.req)
			if !errors.Is(err, ErrInvalidRatesInput) {
				t.Fatalf("error = %v, want ErrInvalidRatesInput", err)
			}

			if provider.ratesCalls != 0 {
				t.Fatal("provider tidak boleh dipanggil untuk input invalid")
			}
		})
	}
}

func TestCalculateRatesVariantNotAvailable(t *testing.T) {
	idA := "11111111-1111-1111-1111-111111111111"

	tests := []struct {
		name string
		repo *fakeVariantRepo
	}{
		{
			name: "variant tidak ditemukan",
			// repo tidak menemukan apa pun untuk ID yang diminta.
			repo: &fakeVariantRepo{variants: []models.ProductVariant{}},
		},
		{
			name: "variant tidak aktif",
			repo: &fakeVariantRepo{variants: []models.ProductVariant{{
				ID: uuid.MustParse(idA), Name: "Kaos", Price: decimal.NewFromInt(1000), IsActive: false, WeightGrams: 100,
			}}},
		},
		{
			name: "berat belum diset",
			repo: &fakeVariantRepo{variants: []models.ProductVariant{{
				ID: uuid.MustParse(idA), Name: "Kaos", Price: decimal.NewFromInt(1000), IsActive: true, WeightGrams: 0,
			}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &fakeProvider{}
			svc := newRatesService(tt.repo, provider)

			req := CalculateRatesRequest{
				DestinationAreaID: "DEST-AREA",
				Items:             []ShippingRateItem{{ProductVariantID: idA, Quantity: 1}},
			}

			_, err := svc.CalculateRates(context.Background(), req)
			if !errors.Is(err, ErrVariantNotAvailable) {
				t.Fatalf("error = %v, want ErrVariantNotAvailable", err)
			}

			if provider.ratesCalls != 0 {
				t.Fatal("provider tidak boleh dipanggil kalau variant tidak tersedia")
			}
		})
	}
}

func TestCalculateRatesRepoError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &fakeVariantRepo{err: repoErr}
	provider := &fakeProvider{}
	svc := newRatesService(repo, provider)

	_, err := svc.CalculateRates(context.Background(), validRatesRequest())
	if !errors.Is(err, repoErr) {
		t.Fatalf("error = %v, want pass-through repo error", err)
	}
}

func TestCalculateRatesProviderErrorPassesThrough(t *testing.T) {
	// biteship provider membungkus error dengan ErrAreaProvider;
	// service meneruskannya apa adanya.
	repo := &fakeVariantRepo{variants: validVariants()}
	provider := &fakeProvider{
		ratesErr: fmt.Errorf("%w: biteship down", ErrAreaProvider),
	}
	svc := newRatesService(repo, provider)

	_, err := svc.CalculateRates(context.Background(), validRatesRequest())
	if !errors.Is(err, ErrAreaProvider) {
		t.Fatalf("error = %v, want ErrAreaProvider", err)
	}
}
