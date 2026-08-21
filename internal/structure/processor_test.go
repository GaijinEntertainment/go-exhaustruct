package structure_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v5/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v5/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v5/internal/pattern"
	"dev.gaijin.team/go/exhaustruct/v5/internal/structure"
)

func Test_Processor_Get(t *testing.T) {
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

				info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

				require.NotNil(t, info)
				assert.Equal(t, tt.structName, info.Name)
				assert.Equal(t, "testdata."+tt.structName, info.FullPath)
				assert.Len(t, info.Fields.Items, tt.wantFields)
			})
		}
	})

	t.Run("exported fields", func(t *testing.T) {
		t.Parallel()

		typeName, strct, pos := td.resolveType(t, "MixedExported")

		info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
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

				info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

				require.NotNil(t, info)
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

			info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "Required", info.Fields.Items[0].Name)
			assert.False(t, info.Fields.Items[0].Optional)

			assert.Equal(t, "Optional", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Optional)
		})

		t.Run("optional via inline comment", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithOptionalInline")

			info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "Required", info.Fields.Items[0].Name)
			assert.False(t, info.Fields.Items[0].Optional)

			assert.Equal(t, "Optional", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Optional)
		})

		t.Run("enforced field", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithEnforcedField")

			info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "Normal", info.Fields.Items[0].Name)
			assert.False(t, info.Fields.Items[0].Enforced)

			assert.Equal(t, "Enforced", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Enforced)
		})

		t.Run("mixed directives", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithMixedDirectives")

			info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
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
		info := proc.ResolveStruct(td.fset, nil, strct, token.NoPos, td.pkg)

		require.NotNil(t, info)
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

		info := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)

		// A processor given no files still answers with the directives of the
		// type's own file: the lookup that resolves it parses that file on
		// demand, so pre-populating the parser changes what it costs and never
		// what it finds.
		assert.True(t, info.Ignored, "the type's own directive has to be resolved")
	})

	t.Run("embedded fields", func(t *testing.T) {
		t.Parallel()

		t.Run("exported embedded", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithEmbedded")

			info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
			require.Len(t, info.Fields.Items, 2)

			assert.Equal(t, "Embedded", info.Fields.Items[0].Name)
			assert.True(t, info.Fields.Items[0].Exported)

			assert.Equal(t, "Own", info.Fields.Items[1].Name)
			assert.True(t, info.Fields.Items[1].Exported)
		})

		t.Run("unexported embedded", func(t *testing.T) {
			t.Parallel()

			typeName, strct, pos := td.resolveType(t, "WithUnexportedEmbedded")

			info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

			require.NotNil(t, info)
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
	info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

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

		lit := &ast.CompositeLit{
			Elts: []ast.Expr{
				&ast.BasicLit{Kind: token.INT, Value: "1"},
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
		info := td.processor.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		lit := &ast.CompositeLit{Elts: []ast.Expr{}}

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

	conf := types.Config{}
	info := &types.Info{
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

		info := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
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

		info := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
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

		info := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
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

		info := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
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

		info := proc.ResolveStruct(td.fset, typeName, strct, pos, td.pkg)

		require.NotNil(t, info)
		assert.False(t, info.PatternEnforced)
		assert.False(t, info.PatternIgnored)
		assert.False(t, info.PatternOptional)
		assert.False(t, info.AllowEmptyDecl)
	})
}

// Test_Processor_ResolveStruct_LineDirective covers declarations a //line
// directive remaps. Their directives and their origin are written in the
// physical file, so resolving either by the adjusted position finds nothing.
func Test_Processor_ResolveStruct_LineDirective(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/lined.go", nil, parser.ParseComments)
	require.NoError(t, err)

	conf := types.Config{}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := conf.Check("testdata", fset, []*ast.File{file}, info)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)
	proc := structure.NewProcessor(scanner, structure.NewOriginScanner(fp))

	scanner.ProcessFiles(fset, file)

	resolve := func(name string) *structure.Struct {
		t.Helper()

		typeName, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		require.True(t, ok)

		// The declaration reports the virtual file, which is what makes this a
		// test of the physical lookup.
		require.Equal(t, filepath.Join("testdata", "generated.tmpl"), fset.Position(typeName.Pos()).Filename)

		strct, ok := typeName.Type().Underlying().(*types.Struct)
		require.True(t, ok)

		return proc.ResolveStruct(fset, typeName, strct, typeName.Pos(), pkg)
	}

	optional := resolve("LinedOptional")
	require.NotNil(t, optional)
	assert.True(t, optional.Optional, "the directive above the declaration must apply")

	derived := resolve("LinedDerived")
	require.NotNil(t, derived)
	assert.True(t, derived.IsDerived, "the origin scan must reach the declaration")
}

// Test_Processor_ResolveStruct_PositionCollision covers types whose declaration
// positions collide. Export data is not required to carry a faithful position:
// gcimporter discards the column entirely and clamps any line past 64Ki to 1,
// so distinct declarations in a dependency collapse onto a single
// token.Position. Caching struct metadata under that position hands the wrong
// type back to the second lookup.
func Test_Processor_ResolveStruct_PositionCollision(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	// Named after a real file so directive and origin lookups can read it; the
	// declarations below are synthetic and absent from it.
	file := fset.AddFile("testdata/structs.go", -1, 10000)
	pos := file.Pos(0)

	pkg := types.NewPackage("example.com/dep", "dep")

	aaaName, aaaStruct := synthStruct(pkg, pos, "Aaa", "X")
	bbbName, bbbStruct := synthStruct(pkg, pos, "Bbb", "P", "Q", "R")

	require.Equal(t, fset.Position(aaaName.Pos()), fset.Position(bbbName.Pos()),
		"test setup: both declarations must share a position")

	fp := astutil.NewFileParser()
	proc := structure.NewProcessor(directive.NewScanner(fp), structure.NewOriginScanner(fp))

	aaa := proc.ResolveStruct(fset, aaaName, aaaStruct, pos, pkg)
	bbb := proc.ResolveStruct(fset, bbbName, bbbStruct, pos, pkg)

	require.NotNil(t, aaa)
	require.NotNil(t, bbb)

	assert.Equal(t, "Aaa", aaa.Name)
	assert.Equal(t, "example.com/dep.Aaa", aaa.FullPath)
	assert.Len(t, aaa.Fields.Items, 1)

	assert.Equal(t, "Bbb", bbb.Name)
	assert.Equal(t, "example.com/dep.Bbb", bbb.FullPath)
	assert.Len(t, bbb.Fields.Items, 3)
}

// synthStruct builds a named struct type whose TypeName and fields all sit at
// pos, mimicking what gcimporter produces when it reads export data.
func synthStruct(
	pkg *types.Package,
	pos token.Pos,
	name string,
	fieldNames ...string,
) (*types.TypeName, *types.Struct) {
	fields := make([]*types.Var, 0, len(fieldNames))

	for _, fieldName := range fieldNames {
		fields = append(fields, types.NewField(pos, pkg, fieldName, types.Typ[types.Int], false))
	}

	strct := types.NewStruct(fields, nil)

	typeName := types.NewTypeName(pos, pkg, name, nil)
	types.NewNamed(typeName, strct, nil)

	return typeName, strct
}

// Test_Struct_SkippedFields_Promoted covers composite literals that name
// promoted fields instead of the embedded field carrying them, which Go 1.27
// permits (golang/go#77245). The literals are parsed rather than compiled, so
// the case holds on toolchains that predate the feature.
func Test_Struct_SkippedFields_Promoted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		literal string
		want    string
	}{
		{"every leaf promoted", `A{b: 1, a: 2, c: 3}`, ""},
		{"embedded named directly", `A{B: b, a: 1}`, ""},
		{"promoted embedded named directly", `A{C: c, b: 1, a: 2}`, ""},
		{"nothing from the embedded subtree", `A{a: 1}`, "B"},
		{"subtree started, deeper embedded left out", `A{b: 1, a: 2}`, "C"},
		{"subtree started at the deepest leaf", `A{c: 1, a: 2}`, "b"},
		{"outer field left out too", `A{c: 1}`, "b, a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			strct := promotedFixture(t)
			lit := parseLiteral(t, tt.literal)

			assert.Equal(t, tt.want, structure.FormatFieldNames(strct.SkippedFields(lit, "example.com/dep")))
		})
	}
}

// Test_Struct_SkippedFields_PromotedShadowed covers a direct field shadowing an
// embedded one: naming it must not count as initializing the embedded field.
func Test_Struct_SkippedFields_PromotedShadowed(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pos := fset.AddFile("testdata/structs.go", -1, 10000).Pos(0)
	pkg := types.NewPackage("example.com/dep", "dep")

	_, innerStruct, innerNamed := synthNamed(t, pkg, pos, "Inner",
		types.NewField(pos, pkg, "x", types.Typ[types.Int], false))
	require.NotNil(t, innerStruct)

	outerName, outerStruct, _ := synthNamed(t, pkg, pos, "Outer",
		types.NewField(pos, pkg, "Inner", innerNamed, true),
		types.NewField(pos, pkg, "x", types.Typ[types.Int], false))

	fp := astutil.NewFileParser()
	proc := structure.NewProcessor(directive.NewScanner(fp), structure.NewOriginScanner(fp))

	strct := proc.ResolveStruct(fset, outerName, outerStruct, pos, pkg)
	require.NotNil(t, strct)

	// Outer.x shadows Inner.x, so `x` initializes the outer field and Inner is
	// left untouched.
	assert.Equal(t, "Inner",
		structure.FormatFieldNames(strct.SkippedFields(parseLiteral(t, `Outer{x: 1}`), "example.com/dep")))
}

// promotedFixture builds A{B; a}, B{C; b}, C{c} — all in one package.
func promotedFixture(t *testing.T) *structure.Struct {
	t.Helper()

	fset := token.NewFileSet()
	pos := fset.AddFile("testdata/structs.go", -1, 10000).Pos(0)
	pkg := types.NewPackage("example.com/dep", "dep")

	_, _, cNamed := synthNamed(t, pkg, pos, "C",
		types.NewField(pos, pkg, "c", types.Typ[types.Int], false))

	_, _, bNamed := synthNamed(t, pkg, pos, "B",
		types.NewField(pos, pkg, "C", cNamed, true),
		types.NewField(pos, pkg, "b", types.Typ[types.Int], false))

	aName, aStruct, _ := synthNamed(t, pkg, pos, "A",
		types.NewField(pos, pkg, "B", bNamed, true),
		types.NewField(pos, pkg, "a", types.Typ[types.Int], false))

	fp := astutil.NewFileParser()
	proc := structure.NewProcessor(directive.NewScanner(fp), structure.NewOriginScanner(fp))

	strct := proc.ResolveStruct(fset, aName, aStruct, pos, pkg)
	require.NotNil(t, strct)

	return strct
}

func synthNamed(
	t *testing.T,
	pkg *types.Package,
	pos token.Pos,
	name string,
	fields ...*types.Var,
) (*types.TypeName, *types.Struct, *types.Named) {
	t.Helper()

	strct := types.NewStruct(fields, nil)
	typeName := types.NewTypeName(pos, pkg, name, nil)

	return typeName, strct, types.NewNamed(typeName, strct, nil)
}

func parseLiteral(t *testing.T, src string) *ast.CompositeLit {
	t.Helper()

	expr, err := parser.ParseExpr(src)
	require.NoError(t, err)

	lit, ok := expr.(*ast.CompositeLit)
	require.True(t, ok)

	return lit
}

// Test_Struct_SkippedFields_EmbeddedPointer covers an embedded pointer. Go does
// not promote through one into a composite literal, so the field stays ordinary
// and is reported by its own name rather than descended into.
func Test_Struct_SkippedFields_EmbeddedPointer(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pos := fset.AddFile("testdata/structs.go", -1, 10000).Pos(0)
	pkg := types.NewPackage("example.com/dep", "dep")

	_, _, cNamed := synthNamed(t, pkg, pos, "C",
		types.NewField(pos, pkg, "c", types.Typ[types.Int], false))

	aName, aStruct, _ := synthNamed(t, pkg, pos, "A",
		types.NewField(pos, pkg, "C", types.NewPointer(cNamed), true),
		types.NewField(pos, pkg, "a", types.Typ[types.Int], false))

	fp := astutil.NewFileParser()
	proc := structure.NewProcessor(directive.NewScanner(fp), structure.NewOriginScanner(fp))

	strct := proc.ResolveStruct(fset, aName, aStruct, pos, pkg)
	require.NotNil(t, strct)

	require.Nil(t, strct.Fields.Items[0].Embedded, "an embedded pointer promotes nothing")

	// `c` names nothing reachable, so C itself is what the literal left out.
	assert.Equal(t, "C",
		structure.FormatFieldNames(strct.SkippedFields(parseLiteral(t, `A{c: 1, a: 2}`), "example.com/dep")))
}

// Test_Struct_SkippedFields_PromotedPatterns covers field patterns aimed at a
// promoted field. Every level of the tree is addressed by the name a literal of
// the outer type writes, so a pattern targets a promoted field by that name.
func Test_Struct_SkippedFields_PromotedPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []structure.Option
		want string
	}{
		{"no patterns", nil, "c2"},
		{
			"promoted field made optional",
			[]structure.Option{structure.WithOptional(mustList(t, `.*\.A#c2`))},
			"",
		},
		{
			"enforce on the promoted field wins over optional",
			[]structure.Option{
				structure.WithOptional(mustList(t, `.*\.A#c2`)),
				structure.WithEnforce(mustList(t, `.*\.A#c2`)),
			},
			"c2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			pos := fset.AddFile("testdata/structs.go", -1, 10000).Pos(0)
			pkg := types.NewPackage("example.com/dep", "dep")

			_, _, cNamed := synthNamed(t, pkg, pos, "C",
				types.NewField(pos, pkg, "c1", types.Typ[types.Int], false),
				types.NewField(pos, pkg, "c2", types.Typ[types.Int], false))

			_, _, bNamed := synthNamed(t, pkg, pos, "B",
				types.NewField(pos, pkg, "C", cNamed, true),
				types.NewField(pos, pkg, "b", types.Typ[types.Int], false))

			aName, aStruct, _ := synthNamed(t, pkg, pos, "A",
				types.NewField(pos, pkg, "B", bNamed, true),
				types.NewField(pos, pkg, "a", types.Typ[types.Int], false))

			fp := astutil.NewFileParser()
			proc := structure.NewProcessor(
				directive.NewScanner(fp), structure.NewOriginScanner(fp), tt.opts...,
			)

			strct := proc.ResolveStruct(fset, aName, aStruct, pos, pkg)
			require.NotNil(t, strct)

			lit := parseLiteral(t, `A{b: 1, a: 2, c1: 3}`)
			assert.Equal(t, tt.want, structure.FormatFieldNames(strct.SkippedFields(lit, "example.com/dep")))
		})
	}
}

// Test_Struct_SkippedFields_PositionalBlank covers a positional literal that
// stops short of the field list. Go rejects such a literal, so no compiling
// fixture reaches this path and the literal here is parsed rather than built.
//
// Blank fields stay in Items to keep positions aligned with the declaration:
// dropping them would shift every field after one, and a literal ending before
// the blank would look complete.
func Test_Struct_SkippedFields_PositionalBlank(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pos := fset.AddFile("testdata/structs.go", -1, 10000).Pos(0)
	pkg := types.NewPackage("example.com/dep", "dep")

	typeName, strct, _ := synthNamed(t, pkg, pos, "Padded",
		types.NewField(pos, pkg, "A", types.Typ[types.Byte], false),
		types.NewField(pos, pkg, "_", types.NewArray(types.Typ[types.Byte], 3), false),
		types.NewField(pos, pkg, "B", types.Typ[types.Byte], false))

	fp := astutil.NewFileParser()
	proc := structure.NewProcessor(directive.NewScanner(fp), structure.NewOriginScanner(fp))

	resolved := proc.ResolveStruct(fset, typeName, strct, pos, pkg)
	require.NotNil(t, resolved)

	require.Len(t, resolved.Fields.Items, 3, "the blank field holds its position")

	// Two elements supplied, so only B is left; were the blank field dropped,
	// two elements would cover the whole list and nothing would be reported.
	assert.Equal(t, "B", structure.FormatFieldNames(
		resolved.SkippedFields(parseLiteral(t, `Padded{1, x}`), "example.com/dep")))
}
