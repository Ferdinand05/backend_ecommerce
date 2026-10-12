package category

import (
	"context"
	"errors"
	"testing"
	"time"

	redisCache "ferdinand/ecommerce/database/redis"
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

func TestCategoryFindAllCachesOnMiss(t *testing.T) {
	repo := &fakeRepo{findAllRes: []models.Category{
		{ID: uuid.New(), Name: "electronics", Slug: "electronics"},
		{ID: uuid.New(), Name: "fashion", Slug: "fashion"},
	}}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := NewService(repo, cache)

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

	if cache.lastTTL != categoryListCacheTTL {
		t.Fatalf("ttl = %v, want %v", cache.lastTTL, categoryListCacheTTL)
	}
}

func TestCategoryFindAllCacheHitSkipsRepo(t *testing.T) {
	repo := &fakeRepo{findAllRes: []models.Category{
		{ID: uuid.New(), Name: "electronics", Slug: "electronics"},
	}}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := NewService(repo, cache)

	if _, err := svc.FindAll(context.Background()); err != nil {
		t.Fatalf("FindAll() first error = %v", err)
	}

	repo.findAllRes = nil // pastikan data hanya bisa datang dari cache.

	got, err := svc.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() second error = %v", err)
	}

	if len(got) != 1 || got[0].Name != "electronics" {
		t.Fatalf("FindAll() = %v, want cached categories", got)
	}
}

func TestCategoryFindAllRepoErrorDoesNotCache(t *testing.T) {
	repo := &fakeRepo{findAllErr: errors.New("db down")}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := NewService(repo, cache)

	if _, err := svc.FindAll(context.Background()); !errors.Is(err, repo.findAllErr) {
		t.Fatalf("FindAll() error = %v, want db down", err)
	}

	if cache.setCalls != 0 {
		t.Fatalf("set calls = %d, want 0", cache.setCalls)
	}
}

func TestCategoryFindAllRedisErrorFallsBackToRepo(t *testing.T) {
	repo := &fakeRepo{findAllRes: []models.Category{{ID: uuid.New(), Name: "electronics"}}}
	cache := &fakeCache{data: map[string][]byte{}, getErr: errors.New("redis down")}
	svc := NewService(repo, cache)

	got, err := svc.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v, want fallback success", err)
	}

	if len(got) != 1 {
		t.Fatalf("FindAll() len = %d, want 1", len(got))
	}
}

func TestCategoryFindAllCorruptCacheFallsBackToRepo(t *testing.T) {
	repo := &fakeRepo{findAllRes: []models.Category{{ID: uuid.New(), Name: "electronics"}}}
	cache := &fakeCache{data: map[string][]byte{categoryListCacheKey: []byte("{not-json")}}
	svc := NewService(repo, cache)

	got, err := svc.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if cache.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1 (cache rusak dihapus)", cache.deleteCalls)
	}

	if len(got) != 1 {
		t.Fatalf("FindAll() len = %d, want 1 dari repo", len(got))
	}
}

func TestCategoryWriteOperationsInvalidateListCache(t *testing.T) {
	repo := &fakeRepo{
		createRes: models.Category{ID: uuid.New(), Name: "electronics"},
		updateRes: models.Category{ID: uuid.New(), Name: "electronics"},
	}
	cache := &fakeCache{data: map[string][]byte{}}
	svc := NewService(repo, cache)

	if _, err := svc.Create(context.Background(), CreateCategoryRequest{Name: "electronics"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := svc.Update(context.Background(), uuid.New(), UpdateCategoryRequest{Name: "gadget"}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := svc.Delete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if cache.deleteCalls != 3 {
		t.Fatalf("delete calls = %d, want 3 (create+update+delete)", cache.deleteCalls)
	}
}
