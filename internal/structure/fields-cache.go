package structure

import (
	"go/types"
	"sync"
)

// FieldsCache provides thread-safe caching of struct field metadata.
type FieldsCache struct {
	fields map[*types.Struct]Fields
	mu     sync.RWMutex
}

const fieldsCachePreallocSize = 64

// Get returns [Fields] for a given type, creating and caching them if needed.
func (c *FieldsCache) Get(typ *types.Struct) Fields {
	c.mu.RLock()

	fields, ok := c.fields[typ]

	c.mu.RUnlock()

	if ok {
		return fields
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if fields, ok = c.fields[typ]; ok {
		return fields
	}

	if c.fields == nil {
		c.fields = make(map[*types.Struct]Fields, fieldsCachePreallocSize)
	}

	fields = NewFields(typ)
	c.fields[typ] = fields

	return fields
}
