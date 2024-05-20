package file_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestASTTypes(t *testing.T) {
	t.Parallel()

	fs := token.NewFileSet()

	file, err := parser.ParseFile(fs, "internal/testdata/types.go", nil, parser.ParseComments)
	require.NoError(t, err)

	skip := false
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		switch typ := n.(type) {
		case *ast.TypeSpec:
			fmt.Printf("TypeSpec: %#v\n\n", typ)
			if _, ok := typ.Type.(*ast.StructType); ok {
				skip = true
			}

		case *ast.StructType:
			if skip {
				skip = false
				return true
			}

			fmt.Printf("StructType: %#v\n\n", typ)
		}

		return true
	})
}
