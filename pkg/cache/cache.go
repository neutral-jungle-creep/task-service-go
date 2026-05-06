// Package cache provides a generic in-memory cache with memory-pressure
// driven eviction. Mirrors the original task-specific implementation:
// sync.Map storage, atomic counters, range-based cleanup that assumes
// auto-increment keys.
package cache

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Sized is implemented by values that can report their approximate in-memory
// footprint.
type Sized interface {
	Size() uint64
}

// Ordered constrains keys to integer-like types — required by the cleanup
// policy that drops the lowest keys first (intended for auto-increment ids).
type Ordered interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~int8 | ~int16 | ~int32 | ~int64 | ~int | ~uint
}

// Cache is a thread-safe in-memory cache keyed by an Ordered type.
type Cache[K Ordered, V Sized] struct {
	items                 sync.Map
	len                   atomic.Uint64
	cleanupStartMB        uint64
	memoryMonitorInterval time.Duration
	firstKey              atomic.Uint64
}

// New constructs a Cache. Both knobs are required; callers (or wrappers)
// should apply defaults before calling New.
func New[K Ordered, V Sized](memoryLimitMB int, memoryMonitorInterval time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		cleanupStartMB:        uint64(float32(memoryLimitMB) * 0.9), // start eviction when 90% of the memory limit is reached
		memoryMonitorInterval: memoryMonitorInterval,
	}
	c.firstKey.Store(1)
	return c
}

// Store inserts or replaces the value for key.
func (c *Cache[K, V]) Store(key K, value V) {
	c.items.Store(key, value)
}

// Get returns the value for key and a boolean indicating whether it was found.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	var zero V
	val, ok := c.items.Load(key)
	if !ok {
		return zero, false
	}
	v, ok := val.(V)
	if !ok {
		return zero, false
	}
	return v, true
}

// List returns a snapshot of all values plus the smallest key currently kept.
func (c *Cache[K, V]) List() ([]V, uint64) {
	out := make([]V, 0, c.len.Load())
	c.items.Range(func(_, value any) bool {
		v, ok := value.(V)
		if ok {
			out = append(out, v)
		}
		return true
	})
	return out, c.firstKey.Load()
}

// FirstKey returns the smallest key currently kept in the cache.
func (c *Cache[K, V]) FirstKey() uint64 {
	return c.firstKey.Load()
}

// SetFirstKey lets wrappers update the smallest known key after pre-filling
// the cache from an external source.
func (c *Cache[K, V]) SetFirstKey(key uint64) {
	c.firstKey.Store(key)
}

// CleanupStartMB returns the threshold (in MB) at which Run triggers cleanup.
// Wrappers can use it to size their own pre-fill budget.
func (c *Cache[K, V]) CleanupStartMB() uint64 {
	return c.cleanupStartMB
}

// Run starts the memory monitor; it blocks until ctx is cancelled. Intended
// to be launched as a background job and shut down via context cancellation.
func (c *Cache[K, V]) Run(ctx context.Context) error {
	if c.memoryMonitorInterval <= 0 {
		<-ctx.Done()
		return nil
	}

	ticker := time.NewTicker(c.memoryMonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			if memStats.Alloc > c.cleanupStartMB*1024*1024 {
				c.Cleanup()
			}
		}
	}
}

// Cleanup drops the oldest 1/5 of the entries assuming auto-increment keys.
// Behaviour mirrors the original task-specific cleanup — it walks keys from
// firstKey and stops at the first missing one.
func (c *Cache[K, V]) Cleanup() {
	cleanupCount := c.len.Load() / 5 // drop the oldest 20% of cached records
	firstStoredKey := c.firstKey.Load()
	var newFirstStoredKey uint64

	for key := firstStoredKey; key < cleanupCount; key++ { // works only for auto-increment ids without gaps
		if _, ok := c.items.Load(K(key)); ok {
			c.items.Delete(K(key))
			continue
		}
		newFirstStoredKey = key
		break
	}

	if newFirstStoredKey == 0 {
		newFirstStoredKey = cleanupCount
	}
	c.firstKey.Store(newFirstStoredKey)
}
