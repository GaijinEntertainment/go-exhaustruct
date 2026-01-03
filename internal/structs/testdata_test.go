package structs_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

// testdata holds parsed test fixtures.
type testdata struct {
	fset  *token.FileSet
	file  *ast.File
	pkg   *types.Package
	types map[string]*types.Struct
	cache *directive.FileCache
}

// loadTestdata parses the testdata file and type-checks it.
func loadTestdata(t *testing.T) *testdata {
	t.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/structs.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse testdata: %v", err)
	}

	conf := types.Config{} //nolint:exhaustruct // using defaults
	info := &types.Info{   //nolint:exhaustruct // only need Types and Defs
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := conf.Check("testdata", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("failed to type-check testdata: %v", err)
	}

	structs := make(map[string]*types.Struct)

	for ident, obj := range info.Defs {
		if obj == nil {
			continue
		}

		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}

		underlying := types.Unalias(tn.Type()).Underlying()

		strct, ok := underlying.(*types.Struct)
		if !ok {
			continue
		}

		structs[ident.Name] = strct
	}

	cache := directive.NewFileCache(nil)
	cache.Add(fset, file)

	return &testdata{
		fset:  fset,
		file:  file,
		pkg:   pkg,
		types: structs,
		cache: cache,
	}
}

// getStruct returns a named struct type from testdata.
func (td *testdata) getStruct(t *testing.T, name string) *types.Struct {
	t.Helper()

	s, ok := td.types[name]
	if !ok {
		t.Fatalf("struct not found in testdata: %s", name)
	}

	return s
}

// getStructPos returns the position of a named struct type from AST.
func (td *testdata) getStructPos(t *testing.T, name string) token.Pos {
	t.Helper()

	for _, decl := range td.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if ts.Name.Name == name {
				return ts.Pos()
			}
		}
	}

	t.Fatalf("struct position not found: %s", name)

	return token.NoPos
}
