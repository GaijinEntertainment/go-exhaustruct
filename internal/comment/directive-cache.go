package comment

import (
	"go/ast"
	"go/token"
	"sync"
	"sync/atomic"

	"golang.org/x/tools/go/analysis"
)

// FileDirectivesCache provides thread-safe caching of parsed file directives.
type FileDirectivesCache struct {
	directives map[string]FileDirectives
	mu         sync.RWMutex

	hits   atomic.Uint64
	misses atomic.Uint64
}

const fileDirectivesCachePreallocSize = 64

// Get returns FileDirectives for a given file, parsing and caching if needed.
// Diagnostics from parsing are returned on cache miss, nil on cache hit.
func (c *FileDirectivesCache) Get(fset *token.FileSet, f *ast.File) (FileDirectives, []analysis.Diagnostic) {
	filename := fset.Position(f.Pos()).Filename

	c.mu.RLock()

	fd, ok := c.directives[filename]

	c.mu.RUnlock()

	if ok {
		c.hits.Add(1)

		return fd, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if fd, ok = c.directives[filename]; ok {
		c.hits.Add(1)

		return fd, nil
	}

	if c.directives == nil {
		c.directives = make(map[string]FileDirectives, fileDirectivesCachePreallocSize)
	}

	c.misses.Add(1)

	fd, diagnostics := NewFileDirectives(fset, f)

	c.directives[filename] = fd

	return fd, diagnostics
}

// Stats returns cache hit count, miss count, and current size.
func (c *FileDirectivesCache) Stats() (hits, misses, size uint64) {
	c.mu.RLock()

	size = uint64(len(c.directives))

	c.mu.RUnlock()

	return c.hits.Load(), c.misses.Load(), size
}
