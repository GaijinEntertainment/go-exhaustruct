package structure

import (
	"go/token"
	"go/types"
	"sync"
	"sync/atomic"

	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

const cachePreallocSize = 64

// Cache provides thread-safe caching of parsed Struct metadata.
type Cache struct {
	structs map[*types.Struct]*Struct
	mu      sync.RWMutex `exhaustruct:"optional"`

	hits   atomic.Uint64 `exhaustruct:"optional"`
	misses atomic.Uint64 `exhaustruct:"optional"`
}

// NewCache creates a new Cache with pre-allocated storage.
func NewCache() *Cache {
	return &Cache{
		structs: make(map[*types.Struct]*Struct, cachePreallocSize),
	}
}

// Get returns Struct for a given struct type, creating and caching it if needed.
// The lookup is used to check for directives when the entry is not cached.
// Returns the Struct and any diagnostics from directive parsing.
func (c *Cache) Get(
	fset *token.FileSet,
	strct *types.Struct,
	name string,
	pkg *types.Package,
	pos token.Pos,
	lookup *directive.FileCache,
) (*Struct, []analysis.Diagnostic) {
	c.mu.RLock()

	s, ok := c.structs[strct]

	c.mu.RUnlock()

	if ok {
		c.hits.Add(1)

		return s, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if s, ok = c.structs[strct]; ok {
		c.hits.Add(1)

		return s, nil
	}

	c.misses.Add(1)

	s, diags := NewStruct(fset, strct, name, pkg, pos, lookup)

	c.structs[strct] = s

	return s, diags
}

// Stats returns cache hit and miss counts.
func (c *Cache) Stats() (hits, misses uint64) {
	return c.hits.Load(), c.misses.Load()
}
