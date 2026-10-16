package shipping

import (
	"context"
	"encoding/json"
	"errors"
	redisCache "ferdinand/ecommerce/database/redis"
	productvariant "ferdinand/ecommerce/internal/product_variant"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

const areaCacheTTL = 14 * 24 * time.Hour

type Service interface {
	SearchAreas(ctx context.Context, query string) ([]Area, error)
	CalculateRates(ctx context.Context, req CalculateRatesRequest) ([]ShippingRate, error)
}

type service struct {
	cache           redisCache.Cache
	provider        Provider
	variantRepo     productvariant.Repository
	originAreaID    string
	defaultCouriers string
}

func NewService(
	cache redisCache.Cache,
	provider Provider,
	variantRepo productvariant.Repository,
	originAreaID string,
	defaultCouriers string,
) *service {
	return &service{
		cache:           cache,
		provider:        provider,
		variantRepo:     variantRepo,
		originAreaID:    originAreaID,
		defaultCouriers: defaultCouriers,
	}
}

func (s *service) SearchAreas(
	ctx context.Context,
	query string,
) ([]Area, error) {
	query = strings.ToLower(strings.TrimSpace(query))

	if query == "" {
		return nil, ErrInvalidAreaQuery
	}

	cacheKey := "shipping:areas:" + query

	//mengambil data dari Redis.
	cached, err := s.cache.Get(ctx, cacheKey)

	if err == nil {
		var areas []Area

		if err := json.Unmarshal(cached, &areas); err == nil {
			return areas, nil
		}

		// Cache tidak valid: hapus dan ambil ulang dari provider.
		slog.WarnContext(ctx, "invalid shipping area cache",
			"key", cacheKey,
		)
		_ = s.cache.Delete(ctx, cacheKey)

	} else if !errors.Is(err, redisCache.ErrCacheMiss) {
		// Redis bermasalah bukan berarti pencarian harus gagal.
		slog.WarnContext(ctx, "shipping area cache unavailable",
			"error", err,
		)
	}

	//Cache miss atau Redis gagal: panggil Biteship.
	areas, err := s.provider.SearchAreas(ctx, query)
	if err != nil {
		return nil, err
	}

	//Simpan hasil ke Redis untuk pencarian berikutnya.
	data, err := json.Marshal(areas)
	if err != nil {
		return nil, fmt.Errorf("marshal shipping areas: %w", err)
	}

	if err := s.cache.Set(ctx, cacheKey, data, areaCacheTTL); err != nil {
		// Hasil dari Biteship tetap bisa dikembalikan.
		slog.WarnContext(ctx, "failed to cache shipping areas",
			"key", cacheKey,
			"error", err,
		)
	}

	return areas, nil
}

// CalculateRates menghitung estimasi ongkir realtime dari Biteship.
// Tarif yang dikembalikan adalah estimasi pada saat request; bukan jaminan harga.
// Saat order dibuat, backend harus memvalidasi ulang dan menyimpan snapshot
// tarif serta layanan yang dipilih — jangan mengandalkan harga dari frontend.
func (s *service) CalculateRates(
	ctx context.Context,
	req CalculateRatesRequest,
) ([]ShippingRate, error) {

	if strings.TrimSpace(req.DestinationAreaID) == "" {
		return nil, fmt.Errorf("%w: destination_area_id is required", ErrInvalidRatesInput)
	}

	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: at least one item is required", ErrInvalidRatesInput)
	}

	// Konversi product_variant_id string → UUID.
	ids := make([]uuid.UUID, 0, len(req.Items))
	qtyByID := make(map[uuid.UUID]int, len(req.Items))

	for _, item := range req.Items {
		id, err := uuid.Parse(item.ProductVariantID)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: invalid product_variant_id %q",
				ErrInvalidRatesInput,
				item.ProductVariantID,
			)
		}

		// Variant sama dua kali = total berat dobel = ongkir salah. Tolak.
		if _, duplicate := qtyByID[id]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate product_variant_id %q",
				ErrInvalidRatesInput,
				item.ProductVariantID,
			)
		}

		if item.Quantity <= 0 {
			return nil, fmt.Errorf(
				"%w: quantity for variant %q must be > 0",
				ErrInvalidRatesInput,
				item.ProductVariantID,
			)
		}

		ids = append(ids, id)
		qtyByID[id] = item.Quantity
	}

	// Ambil semua variant sekaligus — satu query.
	variants, err := s.variantRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetch variants: %w", err)
	}

	// Validasi semua variant:
	// • ditemukan, • aktif, • WeightGrams > 0.
	foundIDs := make(map[uuid.UUID]bool, len(variants))
	for _, v := range variants {
		foundIDs[v.ID] = true
	}

	for _, id := range ids {
		if !foundIDs[id] {
			return nil, fmt.Errorf(
				"%w: variant %s not found",
				ErrVariantNotAvailable,
				id,
			)
		}
	}

	// Susun items untuk Biteship.
	// Weight = berat per unit (gram); Quantity = jumlah.
	// Biteship yang mengalikan weight × quantity internal — jangan pre-multiply.
	// Harga dibulatkan ke integer rupiah; decimal.Decimal.Round(0).IntPart() aman.
	providerItems := make([]RatesProviderItem, 0, len(variants))

	// Indeks variant untuk validasi + akses O(1) per ID.
	variantMap := make(map[uuid.UUID]struct {
		name        string
		value       int64
		weightGrams int
		active      bool
	}, len(variants))

	for _, v := range variants {
		variantMap[v.ID] = struct {
			name        string
			value       int64
			weightGrams int
			active      bool
		}{
			name:        v.Name,
			value:       v.Price.Round(0).IntPart(),
			weightGrams: v.WeightGrams,
			active:      v.IsActive,
		}
	}

	for _, id := range ids {
		vd := variantMap[id]

		if !vd.active {
			return nil, fmt.Errorf(
				"%w: variant %s is not active",
				ErrVariantNotAvailable,
				id,
			)
		}

		if vd.weightGrams == 0 {
			return nil, fmt.Errorf(
				"%w: variant %s has no weight set (WeightGrams = 0)",
				ErrVariantNotAvailable,
				id,
			)
		}

		providerItems = append(providerItems, RatesProviderItem{
			Name:     vd.name,
			Value:    vd.value,
			Weight:   vd.weightGrams,
			Quantity: qtyByID[id],
		})
	}

	rates, err := s.provider.CalculateRates(ctx, RatesProviderRequest{
		OriginAreaID:      s.originAreaID,
		DestinationAreaID: req.DestinationAreaID,
		Couriers:          s.defaultCouriers,
		Items:             providerItems,
	})
	if err != nil {
		return nil, err
	}

	return rates, nil
}
