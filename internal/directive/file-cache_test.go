package directive_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

var errParseFailed = errors.New("parse failed")

// mockParser implements directive.FileParser for testing.
type mockParser struct {
	files map[string]string
}

func (m *mockParser) ParseFile(fset *token.FileSet, filename string) (*ast.File, error) {
	src, ok := m.files[filename]
	if !ok {
		return nil, errParseFailed
	}

	//nolint:wrapcheck // test helper, no need to wrap
	return parser.ParseFile(fset, filename, src, parser.ParseComments)
}

func Test_FileCache_Add(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/directives.go", nil, parser.ParseComments)
	require.NoError(t, err)

	cache := directive.NewFileCache(nil)

	// Initial stats: all zeros
	hits, misses, size := cache.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, uint64(0), size)

	// First Add: returns diagnostics
	diagnostics := cache.Add(fset, file)
	assert.NotEmpty(t, diagnostics, "first Add should return diagnostics")

	_, _, size = cache.Stats()
	assert.Equal(t, uint64(1), size)

	// Second Add: already cached, returns nil
	diagnostics = cache.Add(fset, file)
	assert.Nil(t, diagnostics, "second Add should return nil")
}

func Test_FileCache_Add_MultipleFiles(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	src1 := "package foo\n//exhaustruct:ignore\nvar x int\n"
	file1, err := parser.ParseFile(fset, "file1.go", src1, parser.ParseComments)
	require.NoError(t, err)

	src2 := "package foo\n//exhaustruct:enforce\nvar y int\n"
	file2, err := parser.ParseFile(fset, "file2.go", src2, parser.ParseComments)
	require.NoError(t, err)

	cache := directive.NewFileCache(nil)

	cache.Add(fset, file1)
	cache.Add(fset, file2)

	_, _, size := cache.Stats()
	assert.Equal(t, uint64(2), size)

	// Lookup should find directives from added files
	pos1 := token.Position{Filename: "file1.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed
	d, diags := cache.Lookup(fset, pos1)
	assert.Equal(t, directive.Ignore, d)
	assert.Nil(t, diags) // cache hit, no diagnostics

	pos2 := token.Position{Filename: "file2.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed

	d, diags = cache.Lookup(fset, pos2)
	assert.Equal(t, directive.Enforce, d)
	assert.Nil(t, diags)
}

func Test_FileCache_Lookup(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	mockP := &mockParser{
		files: map[string]string{
			"test.go": "package foo\n//exhaustruct:optional\nvar x int\n",
		},
	}

	cache := directive.NewFileCache(mockP)

	pos := token.Position{Filename: "test.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed

	// First call: cache miss, parses file
	d, diags := cache.Lookup(fset, pos)
	assert.Equal(t, directive.Optional, d)
	assert.Nil(t, diags) // no invalid directives in this file

	hits, misses, _ := cache.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(1), misses)

	// Second call: cache hit
	d, diags = cache.Lookup(fset, pos)
	assert.Equal(t, directive.Optional, d)
	assert.Nil(t, diags)

	hits, misses, _ = cache.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(1), misses)
}

func Test_FileCache_Lookup_EmptyFilename(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	mockP := &mockParser{} //nolint:exhaustruct // empty mock

	cache := directive.NewFileCache(mockP)

	pos := token.Position{} //nolint:exhaustruct // testing empty filename

	d, diags := cache.Lookup(fset, pos)
	assert.Equal(t, directive.Directive(""), d)
	assert.Nil(t, diags)

	// Should not increment stats
	hits, misses, size := cache.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, uint64(0), size)
}

func Test_FileCache_Lookup_ParseError(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	mockP := &mockParser{} //nolint:exhaustruct // empty mock, will return error

	cache := directive.NewFileCache(mockP)

	pos := token.Position{Filename: "nonexistent.go", Line: 1} //nolint:exhaustruct // only Filename and Line needed

	d, diags := cache.Lookup(fset, pos)
	assert.Equal(t, directive.Directive(""), d)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "failed to parse file 'nonexistent.go'")

	// Should increment miss (attempted to parse)
	hits, misses, _ := cache.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(1), misses)
}

func Test_FileCache_Lookup_NoDirectiveAtLine(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	mockP := &mockParser{
		files: map[string]string{
			"test.go": "package foo\n//exhaustruct:optional\nvar x int\n",
		},
	}

	cache := directive.NewFileCache(mockP)

	// Line 1 has no directive
	pos := token.Position{Filename: "test.go", Line: 1} //nolint:exhaustruct // only Filename and Line needed
	d, _ := cache.Lookup(fset, pos)
	assert.Equal(t, directive.Directive(""), d)
}

func Test_FileCache_Lookup_AfterAdd(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	src := "package foo\n//exhaustruct:enforce\nvar x int\n"
	file, err := parser.ParseFile(fset, "shared.go", src, parser.ParseComments)
	require.NoError(t, err)

	mockP := &mockParser{
		files: map[string]string{
			"shared.go": src,
		},
	}

	cache := directive.NewFileCache(mockP)

	// Pre-populate with Add
	cache.Add(fset, file)

	// Lookup should hit cache
	pos := token.Position{Filename: "shared.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed
	d, diags := cache.Lookup(fset, pos)
	assert.Equal(t, directive.Enforce, d)
	assert.Nil(t, diags) // cache hit, no diagnostics

	// Should have 1 hit (from Lookup), 0 misses (Add doesn't count)
	hits, misses, _ := cache.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(0), misses)
}

func Test_FileCache_Lookup_ReturnsDiagnosticsOnMiss(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	// File with invalid directive
	mockP := &mockParser{
		files: map[string]string{
			"test.go": "package foo\n//exhaustruct:invalid\nvar x int\n",
		},
	}

	cache := directive.NewFileCache(mockP)

	pos := token.Position{Filename: "test.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed

	// First call: cache miss, parses file, returns diagnostics
	d, diags := cache.Lookup(fset, pos)
	assert.Equal(t, directive.Directive(""), d) // invalid directive is empty
	require.NotEmpty(t, diags)
	assert.Contains(t, diags[0].Message, "invalid exhaustruct directive")

	// Second call: cache hit, no diagnostics
	d, diags = cache.Lookup(fset, pos)
	assert.Equal(t, directive.Directive(""), d)
	assert.Nil(t, diags)
}
