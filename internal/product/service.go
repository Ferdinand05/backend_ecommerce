package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	redisClient "ferdinand/ecommerce/database/redis"
	"ferdinand/ecommerce/internal/category"
	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/storage"
	"ferdinand/ecommerce/utils/slug"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	// Cache list produk (aktif & non-aktif). Filter, sorting, dan
	// pagination dilakukan di memori service, jadi satu key cukup.
	productListCacheKey = "products:all:v3"

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
		filter ProductFilter,
	) (ProductListResult, error)

	// FindPublicList list produk untuk publik: hanya produk aktif,
	// dengan pencarian, filter kategori, sorting, dan pagination.
	FindPublicList(
		ctx context.Context,
		filter ProductFilter,
	) (ProductListResult, error)

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

// FindAll list produk untuk admin: semua produk (aktif & non-aktif),
// dengan filter opsional dan pagination. Lewat cache supaya admin selalu
// melihat data terkini.
func (s *service) FindAll(ctx context.Context, filter ProductFilter) (ProductListResult, error) {
	filter = filter.Normalize()

	products, err := s.repo.FindAll(ctx)
	if err != nil {
		return ProductListResult{}, err
	}

	return paginateProducts(products, filter), nil
}

// FindPublicList list produk untuk publik: hanya produk aktif, dengan
// pencarian, filter kategori, sorting, dan pagination.
func (s *service) FindPublicList(ctx context.Context, filter ProductFilter) (ProductListResult, error) {
	filter = filter.Normalize()

	active := true
	filter.IsActive = &active

	products, err := s.findProducts(ctx)
	if err != nil {
		return ProductListResult{}, err
	}

	return paginateProducts(products, filter), nil
}

// paginateProducts menyaring, mengurutkan, dan memotong list produk.
func paginateProducts(products []models.Product, filter ProductFilter) ProductListResult {
	matched := make([]models.Product, 0, len(products))

	for _, p := range products {
		if filter.IsActive != nil && p.IsActive != *filter.IsActive {
			continue
		}

		if filter.CategoryID != nil && p.CategoryID != *filter.CategoryID {
			continue
		}

		if filter.Q != "" && !productMatches(p, filter.Q) {
			continue
		}

		matched = append(matched, p)
	}

	sortProducts(matched, filter.Sort)

	total := int64(len(matched))

	start := (filter.Page - 1) * filter.PageSize
	if start > len(matched) {
		start = len(matched)
	}

	end := start + filter.PageSize
	if end > len(matched) {
		end = len(matched)
	}

	return ProductListResult{
		Products: matched[start:end],
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
}

// productMatches mencocokkan keyword ke nama/deskripsi produk.
func productMatches(p models.Product, q string) bool {
	needle := strings.ToLower(q)

	if strings.Contains(strings.ToLower(p.Name), needle) {
		return true
	}

	if p.Description != nil && strings.Contains(strings.ToLower(*p.Description), needle) {
		return true
	}

	return false
}

// sortProducts mengurutkan list produk sesuai parameter sort.
func sortProducts(products []models.Product, sortBy string) {
	switch sortBy {
	case "oldest":
		sort.SliceStable(products, func(i, j int) bool {
			return products[i].CreatedAt.Before(products[j].CreatedAt)
		})
	case "name":
		sort.SliceStable(products, func(i, j int) bool {
			return strings.ToLower(products[i].Name) < strings.ToLower(products[j].Name)
		})
	case "name_desc":
		sort.SliceStable(products, func(i, j int) bool {
			return strings.ToLower(products[i].Name) > strings.ToLower(products[j].Name)
		})
	case "price_asc":
		sort.SliceStable(products, func(i, j int) bool {
			return minVariantPrice(products[i]).Cmp(minVariantPrice(products[j])) < 0
		})
	case "price_desc":
		sort.SliceStable(products, func(i, j int) bool {
			return minVariantPrice(products[i]).Cmp(minVariantPrice(products[j])) > 0
		})
	default:
		sort.SliceStable(products, func(i, j int) bool {
			return products[i].CreatedAt.After(products[j].CreatedAt)
		})
	}
}

// minVariantPrice harga terendah di antara varian aktif produk.
func minVariantPrice(p models.Product) decimal.Decimal {
	lowest := decimal.Zero
	found := false

	for _, v := range p.Variants {
		if !v.IsActive {
			continue
		}

		if !found || v.Price.LessThan(lowest) {
			lowest = v.Price
			found = true
		}
	}

	return lowest
}

// findProducts mengambil seluruh produk, memakai cache Redis bila tersedia.
func (s *service) findProducts(ctx context.Context) ([]models.Product, error) {
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

	// Produk non-aktif tidak boleh bocor lewat endpoint publik.
	if !product.IsActive {
		return models.Product{}, ErrorProductNotFound
	}

	return product, nil
}

func (s *service) FindBySlug(ctx context.Context, slug string) (models.Product, error) {
	product, err := s.repo.FindBySlug(ctx, slug)

	if err != nil {
		return models.Product{}, err
	}

	if !product.IsActive {
		return models.Product{}, ErrorProductNotFound
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
