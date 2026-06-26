package cache

import (
	"context"

	"github.com/dipu-sharma/disgo-bot/internal/config"
	"go.uber.org/zap"
)

// New selects a cache backend based on configuration. When Redis is enabled it
// connects to it; otherwise (or if the connection fails) it logs a warning and
// returns the in-memory backend so the bot remains usable in development.
func New(ctx context.Context, cfg config.RedisConfig, log *zap.Logger) Cache {
	if !cfg.Enabled {
		log.Warn("redis disabled, using in-memory cache (not suitable for production)")
		return newMemory()
	}
	c, err := newRedis(ctx, cfg)
	if err != nil {
		log.Error("redis connection failed, falling back to in-memory cache", zap.Error(err))
		return newMemory()
	}
	log.Info("connected to redis cache", zap.String("addr", cfg.Addr))
	return c
}
