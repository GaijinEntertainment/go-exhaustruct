package structure

import (
	"go/types"
	"sync"
	"sync/atomic"
)

// FieldsCache provides thread-safe caching of struct field metadata.
type FieldsCache struct {
	fields map[*types.Struct]Fields
	mu     sync.RWMutex

	hits   atomic.Uint64
	misses atomic.Uint64
}

const fieldsCachePreallocSize = 64

// Get returns [Fields] for a given type, creating and caching them if needed.
func (c *FieldsCache) Get(typ *types.Struct) Fields {
	c.mu.RLock()

	fields, ok := c.fields[typ]

	c.mu.RUnlock()

	if ok {
		c.hits.Add(1)

		return fields
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if fields, ok = c.fields[typ]; ok {
		c.hits.Add(1)

		return fields
	}

	if c.fields == nil {
		c.fields = make(map[*types.Struct]Fields, fieldsCachePreallocSize)
	}

	c.misses.Add(1)

	fields = NewFields(typ)
	c.fields[typ] = fields

	return fields
}

// Stats returns cache hit and miss counts.
func (c *FieldsCache) Stats() (hits, misses uint64) {
	return c.hits.Load(), c.misses.Load()
}
