package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	"github.com/redis/go-redis/v9"
)

// redisCache is the Redis-backed Cache implementation.
type redisCache struct {
	client *redis.Client
}

// newRedis dials Redis and verifies connectivity.
func newRedis(ctx context.Context, cfg config.RedisConfig) (Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	c := &redisCache{client: client}
	if err := c.Ping(ctx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect redis %q: %w", cfg.Addr, err)
	}
	return c, nil
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	v, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrMiss
	}
	if err != nil {
		return "", fmt.Errorf("cache get: %w", err)
	}
	return v, nil
}

func (c *redisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

func (c *redisCache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache del: %w", err)
	}
	return nil
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("cache exists: %w", err)
	}
	return n > 0, nil
}

func (c *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	n, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("cache incr: %w", err)
	}
	return n, nil
}

func (c *redisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *redisCache) Close() error {
	return c.client.Close()
}
