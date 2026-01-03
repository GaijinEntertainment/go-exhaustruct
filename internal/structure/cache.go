package structure

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/internal/cache"
	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

const cachePreallocSize = 64

// Cache provides thread-safe caching of parsed Struct metadata.
type Cache struct {
	cache *cache.Cache[*types.Struct, *Struct]
}

// NewCache creates a new Cache with pre-allocated storage.
func NewCache() *Cache {
	return &Cache{
		cache: cache.New[*types.Struct, *Struct](cachePreallocSize),
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
	if s, ok := c.cache.Get(strct); ok {
		return s, nil
	}

	s, diags := NewStruct(fset, strct, name, pkg, pos, lookup)

	c.cache.Set(strct, s)

	return s, diags
}

// Stats returns cache hit count, miss count, and current size.
func (c *Cache) Stats() (hits, misses, size uint64) {
	return c.cache.Stats()
}
