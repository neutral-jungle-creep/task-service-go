// Package cache provides a generic in-memory cache with memory-pressure
// driven eviction. sync.Map storage, atomic counters, sorted cleanup that
// drops the oldest keys regardless of gaps.
package cache

import (
	"context"
	"runtime"
	"sort"
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
// policy that drops the lowest keys first.
type Ordered interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~int8 | ~int16 | ~int32 | ~int64 | ~int | ~uint
}

// Cache is a thread-safe in-memory cache keyed by an Ordered type.
//
// snapshotMu serialises List/Cleanup so callers that read both items and
// firstKey see a consistent pair. Store/Get remain lock-free on the
// underlying sync.Map.
type Cache[K Ordered, V Sized] struct {
	snapshotMu            sync.RWMutex
	items                 sync.Map
	len                   atomic.Int64 // signed so Add(-1) is straightforward; never goes negative under snapshotMu
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
	c.firstKey.Store(1) // "expected first id" sentinel until Store / SetFirstKey adjusts it
	return c
}

// Store inserts or replaces the value for key. Tracks len so cleanup can
// reason about how many entries to drop, and keeps firstKey pointing at the
// smallest stored key.
func (c *Cache[K, V]) Store(key K, value V) {
	if _, loaded := c.items.LoadOrStore(key, value); !loaded {
		newLen := c.len.Add(1)
		first := c.firstKey.Load()
		if newLen == 1 || uint64(key) < first {
			c.firstKey.Store(uint64(key))
		}
		return
	}
	c.items.Store(key, value)
}

// Delete removes the entry for key (no-op if missing). Holds snapshotMu so
// concurrent List sees the cache either before or after the deletion.
func (c *Cache[K, V]) Delete(key K) {
	c.snapshotMu.Lock()
	defer c.snapshotMu.Unlock()

	if _, ok := c.items.LoadAndDelete(key); !ok {
		return
	}
	c.len.Add(-1)

	// If the deleted key was firstKey, scan for the new smallest.
	if uint64(key) == c.firstKey.Load() {
		var next uint64
		c.items.Range(func(k, _ any) bool {
			cur, ok := k.(K)
			if !ok {
				return true
			}
			if next == 0 || uint64(cur) < next {
				next = uint64(cur)
			}
			return true
		})
		c.firstKey.Store(next)
	}
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

// List returns a snapshot of all values plus the smallest key currently
// kept, atomically with respect to Cleanup.
func (c *Cache[K, V]) List() ([]V, uint64) {
	c.snapshotMu.RLock()
	defer c.snapshotMu.RUnlock()

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

// Len returns the current number of cached entries.
// The counter is signed internally but never goes negative — Delete only
// decrements when LoadAndDelete succeeded, and snapshotMu serialises mutations.
func (c *Cache[K, V]) Len() uint64 {
	v := c.len.Load()
	if v < 0 {
		return 0
	}
	return uint64(v)
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

// Cleanup drops the oldest 1/5 of the entries. Works for any key set —
// including non-contiguous ids — by sorting current keys ascending and
// removing the first N. Holds snapshotMu so concurrent List sees either the
// pre-cleanup or the post-cleanup state, never a partial view.
func (c *Cache[K, V]) Cleanup() {
	c.snapshotMu.Lock()
	defer c.snapshotMu.Unlock()

	toRemove := int(c.len.Load() / 5) // drop the oldest 20% of cached records
	if toRemove <= 0 {
		return
	}

	keys := make([]K, 0, c.len.Load())
	c.items.Range(func(k, _ any) bool {
		key, ok := k.(K)
		if ok {
			keys = append(keys, key)
		}
		return true
	})
	if len(keys) == 0 {
		return
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	if toRemove > len(keys) {
		toRemove = len(keys)
	}
	for i := 0; i < toRemove; i++ {
		c.items.Delete(keys[i])
		c.len.Add(-1)
	}

	if toRemove < len(keys) {
		c.firstKey.Store(uint64(keys[toRemove]))
	} else {
		c.firstKey.Store(0)
	}
}
