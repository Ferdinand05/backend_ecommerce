package shipping

import (
	"context"
	"encoding/json"
	"errors"
	redisCache "ferdinand/ecommerce/database/redis"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const areaCacheTTL = 14 * 24 * time.Hour

type Service interface {
	SearchAreas(ctx context.Context, query string) ([]Area, error)
}

type service struct {
	cache    redisCache.Cache
	provider Provider
}

func NewService(cache redisCache.Cache, provider Provider) *service {
	return &service{
		cache:    cache,
		provider: provider,
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
