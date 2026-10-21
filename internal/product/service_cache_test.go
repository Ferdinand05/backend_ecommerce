package product

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	redisClient "ferdinand/ecommerce/database/redis"
	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
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
		return nil, redisClient.ErrCacheMiss
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

func newTestService(repo *fakeRepo, cache *fakeCache) *service {
	return NewService(repo, &fakeCategoryRepo{}, cache, nil)
}

func sampleProducts() []models.Product {
	description := "fast"
	mouseDescription := "clicky"

	return []models.Product{
		{ID: uuid.New(), Name: "laptop", Slug: "laptop", Description: &description},
		{ID: uuid.New(), Name: "mouse", Slug: "mouse", Description: &mouseDescription},
	}
}

func TestFindAllCachesOnMiss(t *testing.T) {
	repo := &fakeRepo{findAllRes: sampleProducts()}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := newTestService(repo, cache)

	got, err := svc.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("FindAll() len = %d, want 2", len(got))
	}

	if cache.getCalls != 1 || cache.setCalls != 1 {
		t.Fatalf("cache calls = (get %d, set %d), want (1, 1)", cache.getCalls, cache.setCalls)
	}

	if cache.lastTTL != productListCacheTTL {
		t.Fatalf("ttl = %v, want %v", cache.lastTTL, productListCacheTTL)
	}
}

func TestFindAllCacheHitSkipsRepo(t *testing.T) {
	repo := &fakeRepo{findAllRes: sampleProducts()}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := newTestService(repo, cache)

	// isi cache lewat pemanggilan pertama.
	if _, err := svc.FindAll(context.Background()); err != nil {
		t.Fatalf("FindAll() first error = %v", err)
	}

	repo.findAllRes = nil // pastikan data hanya bisa datang dari cache.

	got, err := svc.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() second error = %v", err)
	}

	if repo.findAllCalls != 1 {
		t.Fatalf("repo calls = %d, want 1 (cache hit tidak boleh ke repo)", repo.findAllCalls)
	}

	if len(got) != 2 || got[0].Name != "laptop" {
		t.Fatalf("FindAll() = %v, want cached products", got)
	}
}

func TestFindAllRepoErrorDoesNotCache(t *testing.T) {
	repo := &fakeRepo{findAllErr: errors.New("db down")}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := newTestService(repo, cache)

	if _, err := svc.FindAll(context.Background()); !errors.Is(err, repo.findAllErr) {
		t.Fatalf("FindAll() error = %v, want db down", err)
	}

	if cache.setCalls != 0 {
		t.Fatalf("set calls = %d, want 0", cache.setCalls)
	}
}

func TestFindAllRedisErrorFallsBackToRepo(t *testing.T) {
	repo := &fakeRepo{findAllRes: sampleProducts()}
	cache := &fakeCache{data: map[string][]byte{}, getErr: errors.New("redis down")}
	svc := newTestService(repo, cache)

	got, err := svc.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v, want fallback success", err)
	}

	if len(got) != 2 {
		t.Fatalf("FindAll() len = %d, want 2", len(got))
	}
}

func TestFindAllCorruptCacheFallsBackToRepo(t *testing.T) {
	repo := &fakeRepo{findAllRes: sampleProducts()}
	cache := &fakeCache{data: map[string][]byte{productListCacheKey: []byte("{not-json")}}
	svc := newTestService(repo, cache)

	got, err := svc.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if cache.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1 (cache rusak dihapus)", cache.deleteCalls)
	}

	if len(got) != 2 {
		t.Fatalf("FindAll() len = %d, want 2 dari repo", len(got))
	}
}

func TestFindAllSetFailureStillSucceeds(t *testing.T) {
	repo := &fakeRepo{findAllRes: sampleProducts()}
	cache := &fakeCache{data: map[string][]byte{}, setErr: errors.New("redis read-only")}
	svc := newTestService(repo, cache)

	got, err := svc.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v, want success meski set gagal", err)
	}

	if len(got) != 2 {
		t.Fatalf("FindAll() len = %d, want 2", len(got))
	}
}

func TestWriteOperationsInvalidateListCache(t *testing.T) {
	categoryRepo := &fakeCategoryRepo{}
	cache := &fakeCache{data: map[string][]byte{}}

	svc := NewService(
		&fakeRepo{createRes: models.Product{ID: uuid.New()}, updateRes: models.Product{ID: uuid.New()}},
		categoryRepo,
		cache,
		nil,
	)

	if _, err := svc.Create(context.Background(), CreateProductRequest{
		Name:       "laptop",
		CategoryID: uuid.New(),
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := svc.Update(context.Background(), uuid.New(), UpdateProductRequest{
		Name:       "laptop pro",
		CategoryID: uuid.New(),
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := svc.Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if cache.deleteCalls != 3 {
		t.Fatalf("delete calls = %d, want 3 (create+update+delete)", cache.deleteCalls)
	}
}

func TestProductCacheRoundTripKeepsData(t *testing.T) {
	products := sampleProducts()

	data, err := json.Marshal(products)
	if err != nil {
		t.Fatalf("marshal error = %v", err)
	}

	var decoded []models.Product
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error = %v", err)
	}

	if len(decoded) != 2 || decoded[0].Slug != "laptop" || decoded[1].Slug != "mouse" {
		t.Fatalf("round trip = %v, want original products", decoded)
	}
}
