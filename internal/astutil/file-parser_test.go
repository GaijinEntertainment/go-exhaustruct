package astutil_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/internal/astutil"
)

func TestFileParser_ParseByName(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	var (
		callbackInvoked bool
		parsedFile      *ast.File
	)

	fp.OnFileParsed(func(_ *token.FileSet, file *ast.File) []analysis.Diagnostic {
		callbackInvoked = true
		parsedFile = file

		return nil
	})

	diags := fp.ParseByName(fset, filename)

	require.Empty(t, diags)
	assert.True(t, callbackInvoked)
	require.NotNil(t, parsedFile)
	assert.Equal(t, "sample", parsedFile.Name.Name)
}

func TestFileParser_ParseByName_HasComments(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	var hasDirective bool

	fp.OnFileParsed(func(_ *token.FileSet, file *ast.File) []analysis.Diagnostic {
		for _, cg := range file.Comments {
			for _, c := range cg.List {
				if c.Text == "//exhaustruct:optional" {
					hasDirective = true

					return nil
				}
			}
		}

		return nil
	})

	diags := fp.ParseByName(fset, filename)

	require.Empty(t, diags)
	assert.True(t, hasDirective)
}

func TestFileParser_ParseByName_Nonexistent(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()

	diags := fp.ParseByName(fset, "nonexistent.go")

	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "read file")
}

func TestFileParser_ParseByName_SyntaxError(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "invalid.go")

	diags := fp.ParseByName(fset, filename)

	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "parse file")
}

func TestFileParser_ParseByName_MultipleCallbacks(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	invocations := []string{}

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		invocations = append(invocations, "first")

		return nil
	})

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		invocations = append(invocations, "second")

		return nil
	})

	diags := fp.ParseByName(fset, filename)

	require.Empty(t, diags)
	assert.Equal(t, []string{"first", "second"}, invocations)
}

func TestFileParser_ParseByName_CallbackDiagnostics(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		return []analysis.Diagnostic{{Message: "diag1"}}
	})

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		return []analysis.Diagnostic{{Message: "diag2"}, {Message: "diag3"}}
	})

	diags := fp.ParseByName(fset, filename)

	require.Len(t, diags, 3)
	assert.Equal(t, "diag1", diags[0].Message)
	assert.Equal(t, "diag2", diags[1].Message)
	assert.Equal(t, "diag3", diags[2].Message)
}

func TestFileParser_Add(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	require.NoError(t, err)

	var callbackInvoked bool

	fp.OnFileParsed(func(_ *token.FileSet, f *ast.File) []analysis.Diagnostic {
		callbackInvoked = true

		assert.Equal(t, "sample", f.Name.Name)

		return nil
	})

	diags := fp.Add(fset, file)

	require.Empty(t, diags)
	assert.True(t, callbackInvoked)
}

func TestFileParser_WithParseFlags(t *testing.T) {
	t.Parallel()

	// SkipObjectResolution without ParseComments should omit comments.
	fp := astutil.NewFileParser(astutil.WithParseFlags(parser.SkipObjectResolution))
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	var hasComments bool

	fp.OnFileParsed(func(_ *token.FileSet, file *ast.File) []analysis.Diagnostic {
		hasComments = len(file.Comments) > 0

		return nil
	})

	diags := fp.ParseByName(fset, filename)

	require.Empty(t, diags)
	assert.False(t, hasComments)
}

func TestFileParser_NoCallbacks(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	diags := fp.ParseByName(fset, filename)

	assert.Empty(t, diags)
}

func TestFileParser_ParseByName_DuplicateSkipped(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	var invocationCount int

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		invocationCount++

		return nil
	})

	diags1 := fp.ParseByName(fset, filename)
	require.Empty(t, diags1)
	assert.Equal(t, 1, invocationCount)

	diags2 := fp.ParseByName(fset, filename)
	assert.Empty(t, diags2)
	assert.Equal(t, 1, invocationCount)
}

func TestFileParser_Add_DuplicateSkipped(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	require.NoError(t, err)

	var invocationCount int

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		invocationCount++

		return nil
	})

	diags1 := fp.Add(fset, file)
	require.Empty(t, diags1)
	assert.Equal(t, 1, invocationCount)

	diags2 := fp.Add(fset, file)
	assert.Empty(t, diags2)
	assert.Equal(t, 1, invocationCount)
}

func TestFileParser_Add_SameFileTwice(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	require.NoError(t, err)

	var invocationCount int

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		invocationCount++

		return nil
	})

	diags := fp.Add(fset, file, file)

	assert.Empty(t, diags)
	assert.Equal(t, 1, invocationCount)
}

func TestFileParser_Stats(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		return nil
	})

	hits, misses, size := fp.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, uint64(0), size)

	fp.ParseByName(fset, filename)

	hits, misses, size = fp.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(1), misses)
	assert.Equal(t, uint64(1), size)

	fp.ParseByName(fset, filename)

	hits, misses, size = fp.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(1), misses)
	assert.Equal(t, uint64(1), size)

	fp.ParseByName(fset, filename)

	hits, misses, size = fp.Stats()
	assert.Equal(t, uint64(2), hits)
	assert.Equal(t, uint64(1), misses)
	assert.Equal(t, uint64(1), size)
}

func TestFileParser_ParseByName_ConcurrentSameFile(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	var callbackCount atomic.Int32

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		callbackCount.Add(1)

		return nil
	})

	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			fp.ParseByName(fset, filename)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(1), callbackCount.Load(),
		"callback should be invoked exactly once")

	_, misses, _ := fp.Stats()
	assert.Equal(t, uint64(1), misses, "file should be parsed once")
}

func TestFileParser_Add_EmptySlice(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()

	var invoked bool

	fp.OnFileParsed(func(_ *token.FileSet, _ *ast.File) []analysis.Diagnostic {
		invoked = true

		return nil
	})

	diags := fp.Add(fset)

	assert.Empty(t, diags)
	assert.False(t, invoked)
}

func TestFileParser_ParseByName_EmptyFilename(t *testing.T) {
	t.Parallel()

	fp := astutil.NewFileParser()
	fset := token.NewFileSet()

	diags := fp.ParseByName(fset, "")

	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "read file")
}
