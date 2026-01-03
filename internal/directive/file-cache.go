package directive

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/internal/cache"
)

// FileParser parses Go source files into AST with comments.
type FileParser interface {
	// ParseFile parses a Go source file and returns its AST.
	ParseFile(fset *token.FileSet, filename string) (*ast.File, error)
}

// FileCache provides thread-safe caching of parsed file directives.
type FileCache struct {
	parser FileParser
	cache  *cache.Cache[string, File]
}

const fileCachePreallocSize = 64

// NewFileCache creates a new cache with the given parser for external files.
// The parser is used by Lookup to parse files not already in the cache.
func NewFileCache(parser FileParser) *FileCache {
	return &FileCache{
		parser: parser,
		cache:  cache.New[string, File](fileCachePreallocSize),
	}
}

// Add pre-populates the cache with directives from already-parsed files.
// Returns diagnostics from directive parsing.
// Use this for files available via pass.Files in the analyzer.
func (c *FileCache) Add(fset *token.FileSet, files ...*ast.File) []analysis.Diagnostic {
	var allDiagnostics []analysis.Diagnostic

	for _, f := range files {
		filename := fset.Position(f.Pos()).Filename

		if _, ok := c.cache.Get(filename); ok {
			continue
		}

		fd, diagnostics := NewFile(fset, f)

		c.cache.Set(filename, fd)

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

	if fd, ok := c.cache.Get(pos.Filename); ok {
		return fd.Lookup(pos.Line), nil
	}

	// Cache miss - parse file and store
	file, err := c.parser.ParseFile(fset, pos.Filename)
	if err != nil {
		return nil, []analysis.Diagnostic{{
			Pos:     token.NoPos,
			Message: "failed to parse file '" + pos.Filename + "': " + err.Error(),
		}}
	}

	fd, diagnostics := NewFile(fset, file)

	c.cache.Set(pos.Filename, fd)

	return fd.Lookup(pos.Line), diagnostics
}

// Stats returns cache hit count, miss count, and current size.
func (c *FileCache) Stats() (hits, misses, size uint64) {
	return c.cache.Stats()
}
