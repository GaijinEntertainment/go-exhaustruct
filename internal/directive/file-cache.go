package directive

import (
	"go/ast"
	"go/token"
	"sync"
	"sync/atomic"

	"golang.org/x/tools/go/analysis"
)

// FileParser parses Go source files into AST with comments.
type FileParser interface {
	// ParseFile parses a Go source file and returns its AST.
	ParseFile(fset *token.FileSet, filename string) (*ast.File, error)
}

// FileCache provides thread-safe caching of parsed file directives.
type FileCache struct {
	parser     FileParser
	directives map[string]File
	mu         sync.RWMutex

	hits   atomic.Uint64
	misses atomic.Uint64
}

const fileCachePreallocSize = 64

// NewFileCache creates a new cache with the given parser for external files.
// The parser is used by Lookup to parse files not already in the cache.
func NewFileCache(parser FileParser) *FileCache {
	return &FileCache{ //nolint:exhaustruct // mu, hits, misses are zero-value initialized
		parser:     parser,
		directives: make(map[string]File, fileCachePreallocSize),
	}
}

// Add pre-populates the cache with directives from already-parsed files.
// Returns diagnostics from directive parsing.
// Use this for files available via pass.Files in the analyzer.
func (c *FileCache) Add(fset *token.FileSet, files ...*ast.File) []analysis.Diagnostic {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.directives == nil {
		c.directives = make(map[string]File, fileCachePreallocSize)
	}

	var allDiagnostics []analysis.Diagnostic

	for _, f := range files {
		filename := fset.Position(f.Pos()).Filename

		if _, ok := c.directives[filename]; ok {
			continue
		}

		fd, diagnostics := NewFile(fset, f)

		c.directives[filename] = fd

		allDiagnostics = append(allDiagnostics, diagnostics...)
	}

	return allDiagnostics
}

// Lookup returns the directives at the given source position and any diagnostics.
// If the file is not cached, it is parsed using the stored parser.
// Returns nil if the position is invalid, file cannot be parsed,
// or no directive exists at that position.
// Diagnostics are returned on cache miss (from directive parsing or parse errors).
func (c *FileCache) Lookup(fset *token.FileSet, pos token.Position) (Directives, []analysis.Diagnostic) {
	if pos.Filename == "" {
		return nil, nil
	}

	c.mu.RLock()

	fd, ok := c.directives[pos.Filename]

	c.mu.RUnlock()

	if ok {
		c.hits.Add(1)

		return fd.Lookup(pos.Line), nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if fd, ok = c.directives[pos.Filename]; ok {
		c.hits.Add(1)

		return fd.Lookup(pos.Line), nil
	}

	if c.directives == nil {
		c.directives = make(map[string]File, fileCachePreallocSize)
	}

	c.misses.Add(1)

	file, err := c.parser.ParseFile(fset, pos.Filename)
	if err != nil {
		return nil, []analysis.Diagnostic{{
			Pos:     token.NoPos,
			Message: "failed to parse file '" + pos.Filename + "': " + err.Error(),
		}}
	}

	fd, diagnostics := NewFile(fset, file)

	c.directives[pos.Filename] = fd

	return fd.Lookup(pos.Line), diagnostics
}

// Stats returns cache hit count, miss count, and current size.
func (c *FileCache) Stats() (hits, misses, size uint64) {
	c.mu.RLock()

	size = uint64(len(c.directives))

	c.mu.RUnlock()

	return c.hits.Load(), c.misses.Load(), size
}
