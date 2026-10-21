package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	redisClient "ferdinand/ecommerce/database/redis"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/storage"
	"ferdinand/ecommerce/utils/slug"

	"github.com/google/uuid"
)

const (
	productListCacheKey = "products:all"

	// produk cukup sering berubah (stok, harga via variant, CRUD admin),
	// TTL pendek agar stale tidak terasa.
	productListCacheTTL = 15 * time.Minute
)

type Service interface {
	Create(
		ctx context.Context,
		req CreateProductRequest,
	) (models.Product, error)

	FindAll(
		ctx context.Context,
	) ([]models.Product, error)

	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (models.Product, error)

	FindBySlug(
		ctx context.Context,
		slug string,
	) (models.Product, error)

	Update(
		ctx context.Context,
		id uuid.UUID,
		req UpdateProductRequest,
	) (models.Product, error)

	Delete(
		ctx context.Context,
		id uuid.UUID,
	) error
}

type service struct {
	repo         Repository
	categoryRepo category.Repository
	cache        redisClient.Cache
	storage      storage.Provider
}

func NewService(repo Repository, categoryRepo category.Repository, cache redisClient.Cache, storage storage.Provider) *service {
	return &service{
		repo:         repo,
		categoryRepo: categoryRepo,
		cache:        cache,
		storage:      storage,
	}
}

func (s *service) Create(ctx context.Context, req CreateProductRequest) (models.Product, error) {

	generatedSlug := slug.MakeSlug(req.Name)

	exists, err := s.repo.ExistsBySlug(ctx, generatedSlug)
	if err != nil {
		return models.Product{}, err
	}

	if exists {
		return models.Product{}, ErrorProductSlugAlreadyExists
	}

	_, err = s.categoryRepo.FindByID(ctx, req.CategoryID)
	if err != nil {

		if errors.Is(err, category.ErrorCategoryNotFound) {
			return models.Product{}, category.ErrorCategoryNotFound
		}

		return models.Product{}, err
	}

	created, err := s.repo.Create(ctx, models.Product{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        generatedSlug,
		Description: req.Description,
		CategoryID:  req.CategoryID,
	})

	if err != nil {
		return models.Product{}, err
	}

	s.invalidateListCache(ctx)

	return created, nil

}

func (s *service) Update(ctx context.Context, id uuid.UUID, req UpdateProductRequest) (models.Product, error) {

	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return models.Product{}, err
	}

	_, err = s.categoryRepo.FindByID(ctx, req.CategoryID)
	if err != nil {

		if errors.Is(err, category.ErrorCategoryNotFound) {
			return models.Product{}, category.ErrorCategoryNotFound
		}

		return models.Product{}, err
	}

	generatedSlug := slug.MakeSlug(req.Name)

	if current.Slug != generatedSlug {

		exists, err := s.repo.ExistsBySlug(ctx, generatedSlug)
		if err != nil {
			return models.Product{}, err
		}

		if exists {
			return models.Product{}, ErrorProductSlugAlreadyExists
		}
	}

	updated, err := s.repo.Update(ctx, id, models.Product{
		Name:        req.Name,
		Slug:        generatedSlug,
		Description: req.Description,
		CategoryID:  req.CategoryID,
	})

	if err != nil {
		return models.Product{}, err
	}

	s.invalidateListCache(ctx)

	return updated, nil

}

func (s *service) FindAll(ctx context.Context) ([]models.Product, error) {

	// Redis bermasalah bukan berarti list produk harus gagal.
	cached, err := s.cache.Get(ctx, productListCacheKey)

	if err == nil {
		var products []models.Product

		if err := json.Unmarshal(cached, &products); err == nil {
			return products, nil
		}

		// Cache tidak valid: hapus dan ambil ulang dari database.
		slog.WarnContext(ctx, "invalid product cache",
			"key", productListCacheKey,
		)
		_ = s.cache.Delete(ctx, productListCacheKey)

	} else if !errors.Is(err, redisClient.ErrCacheMiss) {
		slog.WarnContext(ctx, "product cache unavailable",
			"error", err,
		)
	}

	products, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(products)
	if err != nil {
		return nil, fmt.Errorf("marshal products: %w", err)
	}

	if err := s.cache.Set(ctx, productListCacheKey, data, productListCacheTTL); err != nil {
		slog.WarnContext(ctx, "failed to cache products",
			"key", productListCacheKey,
			"error", err,
		)
	}

	return products, nil
}

// invalidateListCache menghapus cache list setelah ada perubahan data produk.
func (s *service) invalidateListCache(ctx context.Context) {
	if err := s.cache.Delete(ctx, productListCacheKey); err != nil {
		slog.WarnContext(ctx, "failed to invalidate product cache",
			"key", productListCacheKey,
			"error", err,
		)
	}
}

func (s *service) FindByID(ctx context.Context, ID uuid.UUID) (models.Product, error) {
	product, err := s.repo.FindByID(ctx, ID)

	if err != nil {
		return models.Product{}, err
	}

	return product, nil
}

func (s *service) FindBySlug(ctx context.Context, slug string) (models.Product, error) {
	product, err := s.repo.FindBySlug(ctx, slug)

	if err != nil {
		return models.Product{}, err
	}

	return product, nil
}

func (s *service) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	err := s.repo.Delete(ctx, id)

	if err != nil {
		return err
	}

	s.invalidateListCache(ctx)

	return nil
}
