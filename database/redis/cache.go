package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

type cache struct {
	client *redis.Client
}

func NewCache(client *redis.Client) *cache {
	return &cache{
		client: client,
	}
}

func (c *cache) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.client.Get(ctx, key).Bytes()

	if err == redis.Nil {
		return nil, ErrCacheMiss
	}

	if err != nil {
		return nil, err
	}

	return value, nil
}

func (c *cache) Set(
	ctx context.Context,
	key string,
	value []byte,
	ttl time.Duration,
) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *cache) Delete(
	ctx context.Context,
	keys ...string,
) error {
	return c.client.Del(ctx, keys...).Err()
}
