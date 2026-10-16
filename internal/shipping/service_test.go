package shipping

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	redisCache "ferdinand/ecommerce/database/redis"
)

type fakeCache struct {
	data        map[string][]byte
	getErr      error
	setErr      error
	deleteErr   error
	getCalls    int
	setCalls    int
	deleteCalls int
	lastTTL     time.Duration
}

func (f *fakeCache) Get(ctx context.Context, key string) ([]byte, error) {
	f.getCalls++

	if f.getErr != nil {
		return nil, f.getErr
	}

	value, ok := f.data[key]
	if !ok {
		return nil, redisCache.ErrCacheMiss
	}

	return value, nil
}

func (f *fakeCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.setCalls++
	f.lastTTL = ttl

	if f.setErr != nil {
		return f.setErr
	}

	f.data[key] = value
	return nil
}

func (f *fakeCache) Delete(ctx context.Context, keys ...string) error {
	f.deleteCalls++

	if f.deleteErr != nil {
		return f.deleteErr
	}

	for _, key := range keys {
		delete(f.data, key)
	}

	return nil
}

type fakeProvider struct {
	searchAreasRes []Area
	searchAreasErr error
	calls          int

	ratesRes     []ShippingRate
	ratesErr     error
	ratesCalls   int
	lastRatesReq RatesProviderRequest
}

func (f *fakeProvider) SearchAreas(ctx context.Context, query string) ([]Area, error) {
	f.calls++
	return f.searchAreasRes, f.searchAreasErr
}

func (f *fakeProvider) CalculateRates(ctx context.Context, req RatesProviderRequest) ([]ShippingRate, error) {
	f.ratesCalls++
	f.lastRatesReq = req
	return f.ratesRes, f.ratesErr
}

func TestSearchAreasEmptyQuery(t *testing.T) {
	provider := &fakeProvider{}
	svc := NewService(&fakeCache{data: map[string][]byte{}}, provider, &fakeVariantRepo{}, "ORIGIN", "jne")

	_, err := svc.SearchAreas(context.Background(), "   ")
	if !errors.Is(err, ErrInvalidAreaQuery) {
		t.Fatalf("error = %v, want ErrInvalidAreaQuery", err)
	}

	if provider.calls != 0 {
		t.Fatal("provider must not be called for empty query")
	}
}

func TestSearchAreasCacheMiss(t *testing.T) {
	provider := &fakeProvider{searchAreasRes: []Area{{ID: "IDJA", Name: "Jakarta", PostalCode: 10110}}}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := NewService(cache, provider, &fakeVariantRepo{}, "ORIGIN", "jne")

	got, err := svc.SearchAreas(context.Background(), "  Jakarta  ")
	if err != nil {
		t.Fatalf("SearchAreas() error = %v", err)
	}

	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}

	if cache.setCalls != 1 {
		t.Fatalf("set calls = %d, want 1", cache.setCalls)
	}

	if cache.lastTTL != areaCacheTTL {
		t.Fatalf("ttl = %v, want %v", cache.lastTTL, areaCacheTTL)
	}

	if len(got) != 1 || got[0].Name != "Jakarta" {
		t.Fatalf("areas = %v, want Jakarta", got)
	}
}

func TestSearchAreasCacheHitSkipsProvider(t *testing.T) {
	provider := &fakeProvider{}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := NewService(cache, provider, &fakeVariantRepo{}, "ORIGIN", "jne")

	if _, err := svc.SearchAreas(context.Background(), "Jakarta"); err != nil {
		t.Fatalf("SearchAreas() first error = %v", err)
	}

	// pemanggilan pertama mengisi cache; provider tidak boleh dipanggil lagi.
	if _, err := svc.SearchAreas(context.Background(), "jakarta"); err != nil {
		t.Fatalf("SearchAreas() second error = %v", err)
	}

	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1 (cache hit)", provider.calls)
	}
}

func TestSearchAreasCorruptCacheFallsBackToProvider(t *testing.T) {
	provider := &fakeProvider{searchAreasRes: []Area{{ID: "IDBD", Name: "Bandung"}}}
	cache := &fakeCache{data: map[string][]byte{"shipping:areas:bandung": []byte("{not-json")}}
	svc := NewService(cache, provider, &fakeVariantRepo{}, "ORIGIN", "jne")

	got, err := svc.SearchAreas(context.Background(), "bandung")
	if err != nil {
		t.Fatalf("SearchAreas() error = %v", err)
	}

	if cache.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1 (cache rusak dihapus)", cache.deleteCalls)
	}

	if len(got) != 1 || got[0].Name != "Bandung" {
		t.Fatalf("areas = %v, want Bandung dari provider", got)
	}
}

func TestSearchAreasRedisErrorFallsBackToProvider(t *testing.T) {
	provider := &fakeProvider{searchAreasRes: []Area{{ID: "IDSB", Name: "Surabaya"}}}
	cache := &fakeCache{data: map[string][]byte{}, getErr: errors.New("redis down")}
	svc := NewService(cache, provider, &fakeVariantRepo{}, "ORIGIN", "jne")

	got, err := svc.SearchAreas(context.Background(), "surabaya")
	if err != nil {
		t.Fatalf("SearchAreas() error = %v, want fallback success", err)
	}

	if len(got) != 1 {
		t.Fatalf("areas len = %d, want 1", len(got))
	}
}

func TestSearchAreasSetFailureStillSucceeds(t *testing.T) {
	provider := &fakeProvider{searchAreasRes: []Area{{ID: "IDMD", Name: "Medan"}}}
	cache := &fakeCache{data: map[string][]byte{}, setErr: errors.New("redis read-only")}
	svc := NewService(cache, provider, &fakeVariantRepo{}, "ORIGIN", "jne")

	got, err := svc.SearchAreas(context.Background(), "medan")
	if err != nil {
		t.Fatalf("SearchAreas() error = %v, want success meski set gagal", err)
	}

	if len(got) != 1 {
		t.Fatalf("areas len = %d, want 1", len(got))
	}
}

func TestSearchAreasProviderError(t *testing.T) {
	// provider asli (biteship) membungkus error dengan ErrAreaProvider;
	// service meneruskannya apa adanya.
	provider := &fakeProvider{
		searchAreasErr: fmt.Errorf("%w: biteship down", ErrAreaProvider),
	}
	svc := NewService(&fakeCache{data: map[string][]byte{}}, provider, &fakeVariantRepo{}, "ORIGIN", "jne")

	_, err := svc.SearchAreas(context.Background(), "jakarta")
	if !errors.Is(err, ErrAreaProvider) {
		t.Fatalf("error = %v, want ErrAreaProvider", err)
	}
}

func TestSearchAreasUnknownProviderErrorPassesThrough(t *testing.T) {
	providerErr := errors.New("mystery")
	provider := &fakeProvider{searchAreasErr: providerErr}
	svc := NewService(&fakeCache{data: map[string][]byte{}}, provider, &fakeVariantRepo{}, "ORIGIN", "jne")

	_, err := svc.SearchAreas(context.Background(), "jakarta")
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v, want pass-through", err)
	}
}
