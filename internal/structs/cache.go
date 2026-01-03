package structs

import (
	"go/token"
	"go/types"
	"sync"
	"sync/atomic"

	"golang.org/x/tools/go/analysis"
)

// Cache provides thread-safe caching of parsed struct Info.
type Cache struct {
	infos map[*types.Struct]*Info
	mu    sync.RWMutex

	hits   atomic.Uint64
	misses atomic.Uint64
}

const cachePreallocSize = 64

// Get returns Info for a given struct type, creating and caching it if needed.
// The lookup is used to check for directives when the entry is not cached.
// Returns the Info and any diagnostics from directive parsing.
func (c *Cache) Get(
	fset *token.FileSet,
	strct *types.Struct,
	name string,
	pkg *types.Package,
	pos token.Pos,
	lookup DirectiveLookup,
) (*Info, []analysis.Diagnostic) {
	c.mu.RLock()

	info, ok := c.infos[strct]

	c.mu.RUnlock()

	if ok {
		c.hits.Add(1)

		return info, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if info, ok = c.infos[strct]; ok {
		c.hits.Add(1)

		return info, nil
	}

	if c.infos == nil {
		c.infos = make(map[*types.Struct]*Info, cachePreallocSize)
	}

	c.misses.Add(1)

	info, diags := NewInfo(fset, strct, name, pkg, pos, lookup)

	c.infos[strct] = info

	return info, diags
}

// Stats returns cache hit and miss counts.
func (c *Cache) Stats() (hits, misses uint64) {
	return c.hits.Load(), c.misses.Load()
}
