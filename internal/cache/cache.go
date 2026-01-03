// Package cache provides a generic thread-safe cache with hit/miss tracking.
package cache

import (
	"sync"
	"sync/atomic"
)

// Cache is a thread-safe generic cache with hit/miss statistics.
type Cache[K comparable, V any] struct {
	data   map[K]V
	mu     sync.RWMutex  `exhaustruct:"optional"`
	hits   atomic.Uint64 `exhaustruct:"optional"`
	misses atomic.Uint64 `exhaustruct:"optional"`
}

// New creates a cache with pre-allocated capacity.
func New[K comparable, V any](size int) *Cache[K, V] {
	return &Cache[K, V]{
		data: make(map[K]V, size),
	}
}

// Get returns cached value if exists, records hit.
func (c *Cache[K, V]) Get(key K) (v V, ok bool) {
	c.mu.RLock()

	v, ok = c.data[key]

	c.mu.RUnlock()

	if ok {
		c.hits.Add(1)
	}

	return v, ok
}

// Set stores value, records miss.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()

	c.data[key] = value
	c.misses.Add(1)

	c.mu.Unlock()
}

// GetOrSet uses double-check locking pattern.
// Returns cached value if exists, otherwise computes, stores and returns new value.
func (c *Cache[K, V]) GetOrSet(key K, compute func() V) V {
	c.mu.RLock()

	if v, ok := c.data[key]; ok {
		c.mu.RUnlock()
		c.hits.Add(1)

		return v
	}

	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if v, ok := c.data[key]; ok {
		c.hits.Add(1)

		return v
	}

	c.misses.Add(1)

	v := compute()

	c.data[key] = v

	return v
}

// Stats returns hit count, miss count, and current size.
func (c *Cache[K, V]) Stats() (hits, misses, size uint64) {
	c.mu.RLock()

	size = uint64(len(c.data))

	c.mu.RUnlock()

	return c.hits.Load(), c.misses.Load(), size
}
