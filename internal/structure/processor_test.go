package structure_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v4/internal/pattern"
	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

func Test_Processor_Get(t *testing.T) { //nolint:maintidx
	t.Parallel()

	td := loadTestdata(t)

	t.Run("basic structs", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			structName string
			wantFields int
		}{
			{"empty struct", "Empty", 0},
			{"single field", "SingleField", 1},
			{"multiple fields", "MultiField", 3},
			{"mixed exported", "MixedExported", 3},
			{"all unexported", "AllUnexported", 2},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				typeName, strct, pos := td.resolveType(t, tt.structName)

				info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

				require.NotNil(t, info)
				assert.Empty(t, diags)
				assert.Equal(t, tt.structName, info.Name)
				assert.Equal(t, "testdata."+tt.structName, info.FullPath)
				assert.Len(t, info.Fields.Items, tt.wantFields)
			})
		}
	})

	t.Run("exported fields", func(t *testing.T) {
		t.Parallel()

		typeName, strct, pos := td.resolveType(t, "MixedExported")

		info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		require.Len(t, info.Fields.Items, 3)

		assert.Equal(t, "Exported", info.Fields.Items[0].Name)
		assert.True(t, info.Fields.Items[0].Exported)

		assert.Equal(t, "unexported", info.Fields.Items[1].Name)
		assert.False(t, info.Fields.Items[1].Exported)

		assert.Equal(t, "Another", info.Fields.Items[2].Name)
		assert.True(t, info.Fields.Items[2].Exported)
	})

	t.Run("struct level directives", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			structName string
			enforced   bool
			ignored    bool
			optional   bool
		}{
			{"ignored struct", "IgnoredStruct", false, true, false},
			{"enforced struct", "EnforcedStruct", true, false, false},
			{"optional struct", "OptionalStruct", false, false, true},
			{"no directives", "MultiField", false, false, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				typeName, strct, pos := td.resolveType(t, tt.structName)

				info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

				require.NotNil(t, info)
				assert.Empty(t, diags)
				assert.Equal(t, tt.enforced, info.Enforced, "enforced mismatch")
				assert.Equal(t, tt.ignored, info.Ignored, "Ignored mismatch")
				assert.Equal(t, tt.optional, info.Optional, "optional mismatch")
			})
		}
	})

	t.Run("field level directives", func(t *testing.T) {
		t.Parallel()

		t.Run("optional via doc comment", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithOptionalDoc")

			info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "Required", info.Fields.Items[0].Name)
			assert.False(t, info.Fields.Items[0].Optional)

			assert.Equal(t, "Optional", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Optional)
		})

		t.Run("optional via inline comment", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithOptionalInline")

			info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "Required", info.Fields.Items[0].Name)
			assert.False(t, info.Fields.Items[0].Optional)

			assert.Equal(t, "Optional", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Optional)
		})

		t.Run("enforced field", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithEnforcedField")

			info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "Normal", info.Fields.Items[0].Name)
			assert.False(t, info.Fields.Items[0].Enforced)

			assert.Equal(t, "Enforced", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Enforced)
		})

		t.Run("mixed directives", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithMixedDirectives")

			info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields.Items, 3)

			assert.Equal(t, "Normal", info.Fields.Items[0].Name)
			assert.False(t, info.Fields.Items[0].Optional)
			assert.False(t, info.Fields.Items[0].Enforced)

			assert.Equal(t, "Optional", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Optional)
			assert.False(t, info.Fields.Items[1].Enforced)

			assert.Equal(t, "Enforced", info.Fields.Items[2].Name)
			assert.False(t, info.Fields.Items[2].Optional)
			assert.True(t, info.Fields.Items[2].Enforced)
		})
	})

	t.Run("anonymous struct", func(t *testing.T) {
		t.Parallel()

		// Use fresh processor to avoid cache pollution from other tests
		fp := astutil.NewFileParser()
		proc := structure.NewProcessor(
			directive.NewScanner(fp),
			structure.NewOriginScanner(fp),
		)

		// Pass underlying struct type directly to simulate anonymous struct
		typ := td.getType(t, "SingleField")
		strct := types.Unalias(typ).Underlying().(*types.Struct) //nolint:forcetypeassert

		// nil named + NoPos simulates anonymous struct
		info, diags := proc.ResolveStruct(td.fset, nil, strct, token.NoPos, td.pkg)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.Equal(t, structure.AnonymousName, info.Name)
	})

	t.Run("unpopulated processor", func(t *testing.T) {
		t.Parallel()

		// Use fresh processor without pre-populating directives
		fp := astutil.NewFileParser()
		proc := structure.NewProcessor(
			directive.NewScanner(fp),
			structure.NewOriginScanner(fp),
		)

		typeName, strct, pos := td.resolveType(t, "IgnoredStruct")

		info, diags := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
		// Without pre-populating the file parser, the directive from the source file
		// won't be found (parser only has testdata files it has been given).
		// The Ignored flag depends on the directive being parsed.
		assert.Empty(t, diags)
	})

	t.Run("embedded fields", func(t *testing.T) {
		t.Parallel()

		t.Run("exported embedded", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithEmbedded")

			info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "Embedded", info.Fields.Items[0].Name)
			assert.True(t, info.Fields.Items[0].Exported)

			assert.Equal(t, "Own", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Exported)
		})

		t.Run("unexported embedded", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithUnexportedEmbedded")

			info, diags := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "unexported", info.Fields.Items[0].Name)
			assert.False(t, info.Fields.Items[0].Exported)

			assert.Equal(t, "Own", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Exported)
		})
	})
}

func Test_Struct_SkippedFields(t *testing.T) {
	t.Parallel()

	td := loadTestdata(t)

	typeName, strct, pos := td.resolveType(t, "LiteralTest")
	info, _ := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

	require.Len(t, info.Fields.Items, 4)
	assert.False(t, info.Fields.Items[0].Optional) // ExportedRequired
	assert.False(t, info.Fields.Items[1].Optional) // unexportedRequired
	assert.True(t, info.Fields.Items[2].Optional)  // ExportedOptional
	assert.True(t, info.Fields.Items[3].Optional)  // unexportedOptional

	// Package paths for testing external vs same-package access.
	samePkg := info.Fields.PackagePath
	externalPkg := "other/package"

	t.Run("positional complete", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_positionalComplete")
		assert.Nil(t, info.SkippedFields(lit, externalPkg))
		assert.Nil(t, info.SkippedFields(lit, samePkg))
	})

	t.Run("positional incomplete", func(t *testing.T) {
		t.Parallel()

		lit := &ast.CompositeLit{ //nolint:exhaustruct
			Elts: []ast.Expr{
				&ast.BasicLit{Kind: token.INT, Value: "1"}, //nolint:exhaustruct
			},
		}

		// Positional literals now also filter by isFieldRequired.
		skipped := info.SkippedFields(lit, samePkg)
		require.Len(t, skipped, 1)
		assert.Equal(t, "unexportedRequired", skipped[0].Name)

		// External: unexportedRequired is filtered (unexported), no required fields remain.
		assert.Nil(t, info.SkippedFields(lit, externalPkg))
	})

	t.Run("named complete", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_namedComplete")
		assert.Nil(t, info.SkippedFields(lit, externalPkg))
		assert.Nil(t, info.SkippedFields(lit, samePkg))
	})

	t.Run("named missing unexported", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_namedMissingUnexported")

		assert.Nil(t, info.SkippedFields(lit, externalPkg))

		skipped := info.SkippedFields(lit, samePkg)
		require.Len(t, skipped, 1)
		assert.Equal(t, "unexportedRequired", skipped[0].Name)
	})

	t.Run("named missing exported", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_namedMissingExported")

		skipped := info.SkippedFields(lit, externalPkg)
		require.Len(t, skipped, 1)
		assert.Equal(t, "ExportedRequired", skipped[0].Name)

		skipped = info.SkippedFields(lit, samePkg)
		require.Len(t, skipped, 2)
		assert.Equal(t, "ExportedRequired", skipped[0].Name)
		assert.Equal(t, "unexportedRequired", skipped[1].Name)
	})

	t.Run("empty literal", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_empty")

		// Empty literals use positional logic with isFieldRequired filtering.
		skipped := info.SkippedFields(lit, externalPkg)
		require.Len(t, skipped, 1)
		assert.Equal(t, "ExportedRequired", skipped[0].Name)

		skipped = info.SkippedFields(lit, samePkg)
		require.Len(t, skipped, 2)
		assert.Equal(t, "ExportedRequired", skipped[0].Name)
		assert.Equal(t, "unexportedRequired", skipped[1].Name)
	})

	t.Run("empty struct", func(t *testing.T) {
		t.Parallel()

		typeName, strct, pos := td.resolveType(t, "Empty")
		info, _ := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		lit := &ast.CompositeLit{Elts: []ast.Expr{}} //nolint:exhaustruct

		assert.Nil(t, info.SkippedFields(lit, externalPkg))
		assert.Nil(t, info.SkippedFields(lit, samePkg))
	})
}

// testdata holds parsed test fixtures.
type testdata struct {
	fset       *token.FileSet
	file       *ast.File
	pkg        *types.Package
	namedTypes map[string]types.Type
	processor  *structure.Processor
}

// loadTestdata parses the testdata file and type-checks it.
func loadTestdata(t *testing.T) *testdata {
	t.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/structs.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse testdata: %v", err)
	}

	conf := types.Config{} //nolint:exhaustruct
	info := &types.Info{   //nolint:exhaustruct
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := conf.Check("testdata", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("failed to type-check testdata: %v", err)
	}

	namedTypes := make(map[string]types.Type)

	for ident, obj := range info.Defs {
		if obj == nil {
			continue
		}

		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}

		underlying := types.Unalias(tn.Type()).Underlying()
		if _, ok := underlying.(*types.Struct); !ok {
			continue
		}

		namedTypes[ident.Name] = tn.Type()
	}

	fp := astutil.NewFileParser()
	dirScanner := directive.NewScanner(fp)
	originScanner := structure.NewOriginScanner(fp)

	dirScanner.ProcessFiles(fset, file)

	return &testdata{
		fset:       fset,
		file:       file,
		pkg:        pkg,
		namedTypes: namedTypes,
		processor:  structure.NewProcessor(dirScanner, originScanner),
	}
}

func (td *testdata) getType(t *testing.T, name string) types.Type {
	t.Helper()

	typ, ok := td.namedTypes[name]
	if !ok {
		t.Fatalf("type not found in testdata: %s", name)
	}

	return typ
}

// resolveType extracts Named, Struct, and position for the new Processor API.
func (td *testdata) resolveType(t *testing.T, name string) (*types.TypeName, *types.Struct, token.Pos) {
	t.Helper()

	typ := td.getType(t, name)

	named, ok := typ.(*types.Named)
	if !ok {
		t.Fatalf("type %s is not *types.Named", name)
	}

	strct, ok := named.Underlying().(*types.Struct)
	if !ok {
		t.Fatalf("type %s underlying is not *types.Struct", name)
	}

	typeName := named.Obj()

	return typeName, strct, typeName.Pos()
}

func (td *testdata) getLiteral(t *testing.T, name string) *ast.CompositeLit {
	t.Helper()

	obj := td.file.Scope.Lookup(name)
	if obj == nil {
		t.Fatalf("literal %q not found", name)
	}

	vs, ok := obj.Decl.(*ast.ValueSpec)
	if !ok {
		t.Fatalf("literal %q is not a ValueSpec", name)
	}

	if len(vs.Values) == 0 {
		t.Fatalf("literal %q has no values", name)
	}

	lit, ok := vs.Values[0].(*ast.CompositeLit)
	if !ok {
		t.Fatalf("literal %q is not a CompositeLit", name)
	}

	return lit
}

func mustList(t *testing.T, patterns ...string) pattern.List {
	t.Helper()

	list, err := pattern.NewList(patterns...)
	require.NoError(t, err)

	return list
}

func Test_Processor_WithPatterns(t *testing.T) {
	t.Parallel()

	td := loadTestdata(t)

	t.Run("WithEnforce", func(t *testing.T) {
		t.Parallel()

		fp := astutil.NewFileParser()
		proc := structure.NewProcessor(
			directive.NewScanner(fp),
			structure.NewOriginScanner(fp),
			structure.WithEnforce(mustList(t, `testdata\.MultiField`)),
		)

		typeName, strct, pos := td.resolveType(t, "MultiField")

		info, diags := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.True(t, info.PatternEnforced)
		assert.False(t, info.PatternIgnored)
		assert.False(t, info.PatternOptional)
	})

	t.Run("WithIgnore", func(t *testing.T) {
		t.Parallel()

		fp := astutil.NewFileParser()
		proc := structure.NewProcessor(
			directive.NewScanner(fp),
			structure.NewOriginScanner(fp),
			structure.WithIgnore(mustList(t, `testdata\.MultiField`)),
		)

		typeName, strct, pos := td.resolveType(t, "MultiField")

		info, diags := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.False(t, info.PatternEnforced)
		assert.True(t, info.PatternIgnored)
		assert.False(t, info.PatternOptional)
	})

	t.Run("WithOptional", func(t *testing.T) {
		t.Parallel()

		fp := astutil.NewFileParser()
		proc := structure.NewProcessor(
			directive.NewScanner(fp),
			structure.NewOriginScanner(fp),
			structure.WithOptional(mustList(t, `testdata\.MultiField`)),
		)

		typeName, strct, pos := td.resolveType(t, "MultiField")

		info, diags := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.False(t, info.PatternEnforced)
		assert.False(t, info.PatternIgnored)
		assert.True(t, info.PatternOptional)
	})

	t.Run("WithAllowEmpty", func(t *testing.T) {
		t.Parallel()

		fp := astutil.NewFileParser()
		proc := structure.NewProcessor(
			directive.NewScanner(fp),
			structure.NewOriginScanner(fp),
			structure.WithAllowEmpty(mustList(t, `testdata\.MultiField`)),
		)

		typeName, strct, pos := td.resolveType(t, "MultiField")

		info, diags := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.True(t, info.AllowEmptyDecl)
	})

	t.Run("non-matching patterns", func(t *testing.T) {
		t.Parallel()

		fp := astutil.NewFileParser()
		proc := structure.NewProcessor(
			directive.NewScanner(fp),
			structure.NewOriginScanner(fp),
			structure.WithEnforce(mustList(t, `other\.Type`)),
			structure.WithIgnore(mustList(t, `other\.Type`)),
			structure.WithOptional(mustList(t, `other\.Type`)),
			structure.WithAllowEmpty(mustList(t, `other\.Type`)),
		)

		typeName, strct, pos := td.resolveType(t, "MultiField")

		info, diags := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.False(t, info.PatternEnforced)
		assert.False(t, info.PatternIgnored)
		assert.False(t, info.PatternOptional)
		assert.False(t, info.AllowEmptyDecl)
	})
}
