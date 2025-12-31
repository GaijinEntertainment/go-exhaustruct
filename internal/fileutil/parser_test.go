package fileutil_test

import (
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/fileutil"
)

func TestParser_ParseFile(t *testing.T) {
	t.Parallel()

	reader := fileutil.NewReader(nil)
	parser := fileutil.NewParser(reader)
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	file, err := parser.ParseFile(fset, filename)
	require.NoError(t, err)
	require.NotNil(t, file)

	assert.Equal(t, "sample", file.Name.Name)
}

func TestParser_ParseFile_HasComments(t *testing.T) {
	t.Parallel()

	reader := fileutil.NewReader(nil)
	parser := fileutil.NewParser(reader)
	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	file, err := parser.ParseFile(fset, filename)
	require.NoError(t, err)
	require.NotNil(t, file)
	require.NotEmpty(t, file.Comments, "expected comments to be parsed")

	// Check that directive comments are present
	var hasDirective bool
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if c.Text == "//exhaustruct:optional" {
				hasDirective = true
				break
			}
		}
	}

	assert.True(t, hasDirective, "expected to find //exhaustruct:optional directive")
}

func TestParser_ParseFile_Nonexistent(t *testing.T) {
	t.Parallel()

	reader := fileutil.NewReader(nil)
	parser := fileutil.NewParser(reader)
	fset := token.NewFileSet()

	_, err := parser.ParseFile(fset, "nonexistent.go")
	require.Error(t, err)
}

func TestParser_ParseFile_InvalidSyntax(t *testing.T) {
	t.Parallel()

	// Primary returns invalid Go source
	primary := func(_ string) ([]byte, error) {
		return []byte("not valid go code {{{"), nil
	}

	reader := fileutil.NewReader(primary)
	parser := fileutil.NewParser(reader)
	fset := token.NewFileSet()

	_, err := parser.ParseFile(fset, "invalid.go")
	require.Error(t, err)
}

func TestParser_ParseFile_WithPrimaryReader(t *testing.T) {
	t.Parallel()

	customSource := []byte(`// Custom source file.
package custom

// CustomStruct doc
type CustomStruct struct {
	//exhaustruct:optional
	Field string
}
`)

	primary := func(_ string) ([]byte, error) {
		return customSource, nil
	}

	reader := fileutil.NewReader(primary)
	parser := fileutil.NewParser(reader)
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "custom.go")
	require.NoError(t, err)
	require.NotNil(t, file)

	assert.Equal(t, "custom", file.Name.Name)
	assert.NotEmpty(t, file.Comments)
}
