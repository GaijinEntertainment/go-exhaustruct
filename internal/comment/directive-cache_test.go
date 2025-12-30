package comment_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/comment"
)

func Test_FileDirectivesCache_Stats(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/directives.go", nil, parser.ParseComments)
	require.NoError(t, err)

	var cache comment.FileDirectivesCache

	// Initial stats: all zeros
	{
		hits, misses, size := cache.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(0), misses)
		assert.Equal(t, uint64(0), size)
	}

	// First Get: cache miss, returns diagnostics
	{
		fd, diagnostics := cache.Get(fset, file)
		assert.NotNil(t, fd)
		assert.NotEmpty(t, diagnostics, "first Get should return diagnostics")

		hits, misses, size := cache.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(1), misses)
		assert.Equal(t, uint64(1), size)
	}

	// Second Get: cache hit, returns nil diagnostics
	{
		fd, diagnostics := cache.Get(fset, file)
		assert.NotNil(t, fd)
		assert.Nil(t, diagnostics, "cache hit should return nil diagnostics")

		hits, misses, size := cache.Stats()
		assert.Equal(t, uint64(1), hits)
		assert.Equal(t, uint64(1), misses)
		assert.Equal(t, uint64(1), size)
	}

	// Third Get: another cache hit
	{
		_, _ = cache.Get(fset, file)

		hits, misses, size := cache.Stats()
		assert.Equal(t, uint64(2), hits)
		assert.Equal(t, uint64(1), misses)
		assert.Equal(t, uint64(1), size)
	}
}

func Test_FileDirectivesCache_MultipleFiles(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	src1 := "package foo\n//exhaustruct:ignore\nvar x int\n"
	file1, err := parser.ParseFile(fset, "file1.go", src1, parser.ParseComments)
	require.NoError(t, err)

	src2 := "package foo\n//exhaustruct:enforce\nvar y int\n"
	file2, err := parser.ParseFile(fset, "file2.go", src2, parser.ParseComments)
	require.NoError(t, err)

	var cache comment.FileDirectivesCache

	// Get file1: miss
	fd1, _ := cache.Get(fset, file1)
	assert.Equal(t, comment.DirectiveIgnore, fd1.Lookup(3))

	// Get file2: miss
	fd2, _ := cache.Get(fset, file2)
	assert.Equal(t, comment.DirectiveEnforce, fd2.Lookup(3))

	// Stats: 0 hits, 2 misses, 2 size
	hits, misses, size := cache.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(2), misses)
	assert.Equal(t, uint64(2), size)

	// Get file1 again: hit
	fd1Again, diagnostics := cache.Get(fset, file1)
	assert.Equal(t, comment.DirectiveIgnore, fd1Again.Lookup(3))
	assert.Nil(t, diagnostics)

	hits, misses, size = cache.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(2), misses)
	assert.Equal(t, uint64(2), size)
}
