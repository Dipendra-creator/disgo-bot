package cache

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// memoryCache is an in-process Cache used when Redis is disabled. It is safe for
// concurrent use and lazily evicts expired entries on access, with a background
// janitor sweeping the rest.
type memoryCache struct {
	mu     sync.RWMutex
	items  map[string]entry
	stop   chan struct{}
	closed bool
}

type entry struct {
	value     string
	expiresAt time.Time // zero == no expiry
}

func (e entry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && now.After(e.expiresAt)
}

// newMemory returns a memory cache with a background eviction loop.
func newMemory() Cache {
	c := &memoryCache{
		items: make(map[string]entry),
		stop:  make(chan struct{}),
	}
	go c.janitor()
	return c
}

func (c *memoryCache) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case now := <-ticker.C:
			c.mu.Lock()
			for k, e := range c.items {
				if e.expired(now) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		}
	}
}

func (c *memoryCache) Get(_ context.Context, key string) (string, error) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || e.expired(time.Now()) {
		return "", ErrMiss
	}
	return e.value, nil
}

func (c *memoryCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.items[key] = entry{value: value, expiresAt: exp}
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Del(_ context.Context, keys ...string) error {
	c.mu.Lock()
	for _, k := range keys {
		delete(c.items, k)
	}
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Exists(_ context.Context, key string) (bool, error) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	return ok && !e.expired(time.Now()), nil
}

func (c *memoryCache) Incr(_ context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var n int64
	if e, ok := c.items[key]; ok && !e.expired(time.Now()) {
		n, _ = strconv.ParseInt(e.value, 10, 64)
	}
	n++
	c.items[key] = entry{value: strconv.FormatInt(n, 10)}
	return n, nil
}

func (c *memoryCache) Ping(_ context.Context) error { return nil }

func (c *memoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		close(c.stop)
		c.closed = true
	}
	return nil
}
