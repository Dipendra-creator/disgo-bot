// Package cache provides a small key/value cache abstraction with two backends:
// Redis (production) and an in-process memory store (development / Redis-absent).
//
// Modules depend only on the Cache interface, never on a concrete backend, so
// the storage engine can change without touching feature code.
package cache

import (
	"context"
	"errors"
	"time"
)

// ErrMiss is returned by Get when the key does not exist.
var ErrMiss = errors.New("cache: key not found")

// Cache is the storage-agnostic interface used throughout the bot.
type Cache interface {
	// Get returns the value for key, or ErrMiss if absent/expired.
	Get(ctx context.Context, key string) (string, error)
	// Set stores value under key. A ttl of 0 means no expiry.
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Del removes the given keys (missing keys are ignored).
	Del(ctx context.Context, keys ...string) error
	// Exists reports whether key is present and unexpired.
	Exists(ctx context.Context, key string) (bool, error)
	// Incr atomically increments the integer stored at key and returns it.
	Incr(ctx context.Context, key string) (int64, error)
	// Ping verifies backend connectivity.
	Ping(ctx context.Context) error
	// Close releases backend resources.
	Close() error
}
