package cache_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/cache"
)

type item struct {
	id   uint64
	size uint64
}

func (i *item) Size() uint64 { return i.size }

func newCache(t *testing.T) *cache.Cache[uint64, *item] {
	t.Helper()
	return cache.New[uint64, *item](128, time.Hour)
}

func TestCache_StoreGet(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	c.Store(1, &item{id: 1, size: 10})

	got, ok := c.Get(1)
	require.True(t, ok)
	assert.Equal(t, uint64(1), got.id)

	_, ok = c.Get(999)
	assert.False(t, ok)
}

func TestCache_List(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	for i := uint64(1); i <= 5; i++ {
		c.Store(i, &item{id: i, size: 10})
	}

	values, firstKey := c.List()
	assert.Len(t, values, 5)
	assert.Equal(t, uint64(1), firstKey)
}

func TestCache_FirstKeyExposedToWrappers(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	c.Store(42, &item{id: 42, size: 1})
	c.SetFirstKey(42)

	assert.Equal(t, uint64(42), c.FirstKey())
}

func TestCache_RunStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	c := cache.New[uint64, *item](64, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after context cancel")
	}
}

func TestCache_Cleanup_DropsOldest(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	for i := uint64(1); i <= 10; i++ {
		c.Store(i, &item{id: i, size: 1})
	}

	c.Cleanup() // 20% of 10 == 2 oldest

	assert.Equal(t, uint64(8), c.Len())
	_, ok := c.Get(1)
	assert.False(t, ok, "id 1 should be evicted")
	_, ok = c.Get(2)
	assert.False(t, ok, "id 2 should be evicted")
	_, ok = c.Get(3)
	assert.True(t, ok, "id 3 must survive")
	assert.Equal(t, uint64(3), c.FirstKey())
}

func TestCache_Cleanup_NoOpWhenBelowFive(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	for i := uint64(1); i <= 4; i++ {
		c.Store(i, &item{id: i, size: 1})
	}

	c.Cleanup() // toRemove = 4/5 = 0, must not touch anything

	assert.Equal(t, uint64(4), c.Len())
	_, ok := c.Get(1)
	assert.True(t, ok)
	assert.Equal(t, uint64(1), c.FirstKey())
}

func TestCache_Cleanup_EmptyCacheNoOp(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	assert.NotPanics(t, func() { c.Cleanup() })
	assert.Equal(t, uint64(0), c.Len())
}

func TestCache_Cleanup_RemovesAllWhenSmall(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	// 5 items → toRemove = 1; after cleanup len=4, firstKey=2
	for i := uint64(1); i <= 5; i++ {
		c.Store(i, &item{id: i, size: 1})
	}
	c.Cleanup()
	assert.Equal(t, uint64(4), c.Len())
	assert.Equal(t, uint64(2), c.FirstKey())
}

func TestCache_Run_NoIntervalWaitsForCtx(t *testing.T) {
	t.Parallel()

	c := cache.New[uint64, *item](64, 0) // monitorInterval == 0 → just block on ctx
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after ctx cancel")
	}
}

func TestCache_Store_UpdateExisting(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	c.Store(7, &item{id: 7, size: 10})
	c.Store(7, &item{id: 7, size: 99}) // overwrite, len must stay 1

	assert.Equal(t, uint64(1), c.Len())
	got, ok := c.Get(7)
	require.True(t, ok)
	assert.Equal(t, uint64(99), got.size)
}

func TestCache_Cleanup_HandlesGaps(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	for _, id := range []uint64{1, 5, 17, 42, 100} {
		c.Store(id, &item{id: id, size: 1})
	}

	c.Cleanup() // 20% of 5 == 1; oldest (id=1) removed

	assert.Equal(t, uint64(4), c.Len())
	_, ok := c.Get(1)
	assert.False(t, ok)
	_, ok = c.Get(5)
	assert.True(t, ok)
	assert.Equal(t, uint64(5), c.FirstKey())
}

func TestCache_ConcurrentStore(t *testing.T) {
	t.Parallel()

	c := newCache(t)
	const writers = 10
	const perWriter = 100

	var wg sync.WaitGroup
	wg.Add(writers)
	for w := uint64(0); w < writers; w++ {
		go func(base uint64) {
			defer wg.Done()
			for i := uint64(0); i < perWriter; i++ {
				id := base*1000 + i
				c.Store(id, &item{id: id, size: 1})
			}
		}(w)
	}
	wg.Wait()

	// sync.Map iteration to count inserted items
	got, _ := c.List()
	assert.Len(t, got, writers*perWriter)
}
