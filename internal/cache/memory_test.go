package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCacheSetGet(t *testing.T) {
	ctx := context.Background()
	c := newMemory()
	defer c.Close()

	require.NoError(t, c.Set(ctx, "k", "v", 0))

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	ok, err := c.Exists(ctx, "k")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestMemoryCacheMiss(t *testing.T) {
	c := newMemory()
	defer c.Close()

	_, err := c.Get(context.Background(), "absent")
	assert.ErrorIs(t, err, ErrMiss)
}

func TestMemoryCacheTTLExpiry(t *testing.T) {
	ctx := context.Background()
	c := newMemory()
	defer c.Close()

	require.NoError(t, c.Set(ctx, "k", "v", 20*time.Millisecond))
	time.Sleep(40 * time.Millisecond)

	_, err := c.Get(ctx, "k")
	assert.ErrorIs(t, err, ErrMiss)
}

func TestMemoryCacheIncr(t *testing.T) {
	ctx := context.Background()
	c := newMemory()
	defer c.Close()

	n, err := c.Incr(ctx, "count")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = c.Incr(ctx, "count")
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestMemoryCacheDel(t *testing.T) {
	ctx := context.Background()
	c := newMemory()
	defer c.Close()

	require.NoError(t, c.Set(ctx, "k", "v", 0))
	require.NoError(t, c.Del(ctx, "k"))

	ok, err := c.Exists(ctx, "k")
	require.NoError(t, err)
	assert.False(t, ok)
}
