package category

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	redisCache "ferdinand/ecommerce/database/redis"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/utils/slug"

	"github.com/google/uuid"
)

const (
	categoryListCacheKey = "categories:all"

	// kategori sangat jarang berubah, TTL boleh panjang.
	categoryListCacheTTL = 24 * time.Hour
)

type Service interface {
	FindAll(ctx context.Context) ([]models.Category, error)
	FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error)
	FindBySlug(ctx context.Context, slug string) (models.Category, error)

	Create(ctx context.Context, category CreateCategoryRequest) (models.Category, error)
	Update(ctx context.Context, ID uuid.UUID, category UpdateCategoryRequest) (models.Category, error)
	Delete(ctx context.Context, ID uuid.UUID) error
}

type service struct {
	repo  Repository
	cache redisCache.Cache
}

func NewService(repo Repository, cache redisCache.Cache) *service {
	return &service{
		repo:  repo,
		cache: cache,
	}
}

func (s *service) FindAll(ctx context.Context) ([]models.Category, error) {

	// Redis bermasalah bukan berarti list kategori harus gagal.
	cached, err := s.cache.Get(ctx, categoryListCacheKey)

	if err == nil {
		var categories []models.Category

		if err := json.Unmarshal(cached, &categories); err == nil {
			return categories, nil
		}

		// Cache tidak valid: hapus dan ambil ulang dari database.
		slog.WarnContext(ctx, "invalid category cache",
			"key", categoryListCacheKey,
		)
		_ = s.cache.Delete(ctx, categoryListCacheKey)

	} else if !errors.Is(err, redisCache.ErrCacheMiss) {
		slog.WarnContext(ctx, "category cache unavailable",
			"error", err,
		)
	}

	categories, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(categories)
	if err != nil {
		return nil, fmt.Errorf("marshal categories: %w", err)
	}

	if err := s.cache.Set(ctx, categoryListCacheKey, data, categoryListCacheTTL); err != nil {
		slog.WarnContext(ctx, "failed to cache categories",
			"key", categoryListCacheKey,
			"error", err,
		)
	}

	return categories, nil
}

// invalidateListCache menghapus cache list setelah ada perubahan data kategori.
func (s *service) invalidateListCache(ctx context.Context) {
	if err := s.cache.Delete(ctx, categoryListCacheKey); err != nil {
		slog.WarnContext(ctx, "failed to invalidate category cache",
			"key", categoryListCacheKey,
			"error", err,
		)
	}
}

func (s *service) FindByID(ctx context.Context, ID uuid.UUID) (models.Category, error) {
	category, err := s.repo.FindByID(ctx, ID)

	if err != nil {
		return models.Category{}, err
	}

	return category, nil
}

func (s *service) FindBySlug(ctx context.Context, slug string) (models.Category, error) {
	category, err := s.repo.FindBySlug(ctx, slug)

	if err != nil {
		return models.Category{}, err
	}

	return category, nil
}

func (s *service) Create(ctx context.Context, category CreateCategoryRequest) (models.Category, error) {

	request := models.Category{
		ID:          uuid.New(),
		Name:        category.Name,
		Description: category.Description,
		Slug:        slug.MakeSlug(category.Name),
	}

	created, err := s.repo.Create(ctx, request)

	if err != nil {
		return models.Category{}, err
	}

	s.invalidateListCache(ctx)

	return created, nil

}

func (s *service) Update(ctx context.Context, ID uuid.UUID, category UpdateCategoryRequest) (models.Category, error) {

	request := models.Category{
		Name:        category.Name,
		Description: category.Description,
		Slug:        slug.MakeSlug(category.Name),
	}

	updated, err := s.repo.Update(ctx, ID, request)

	if err != nil {
		return models.Category{}, err
	}

	s.invalidateListCache(ctx)

	return updated, nil

}

func (s *service) Delete(ctx context.Context, ID uuid.UUID) error {
	err := s.repo.Delete(ctx, ID)
	if err != nil {
		return err
	}

	s.invalidateListCache(ctx)

	return nil
}
