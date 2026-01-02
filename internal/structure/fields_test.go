package structure_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/tools/go/packages"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

// mockDirectiveLookup implements structure.DirectiveLookup for testing.
type mockDirectiveLookup struct {
	dirs directive.Directives
}

func (m *mockDirectiveLookup) Lookup(_ token.Pos) directive.Directives {
	return m.dirs
}

func Test_HasOptionalTag(t *testing.T) {
	t.Parallel()

	assert.True(t, structure.HasOptionalTag(`exhaustruct:"optional"`))
	assert.False(t, structure.HasOptionalTag(`exhaustruct:"required"`))
	assert.False(t, structure.HasOptionalTag(``))
	assert.False(t, structure.HasOptionalTag(`json:"name"`))
}

func Test_Fields_String_Empty(t *testing.T) {
	t.Parallel()

	var empty structure.Fields
	assert.Empty(t, empty.String())
}

func TestStructFields(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(StructFieldsSuite))
}

type StructFieldsSuite struct {
	suite.Suite

	// Note: ast.Scope is deprecated but there's no direct replacement for looking up
	// AST declarations by name. Loaded package still provides it.
	scope *ast.Scope //nolint:staticcheck
	pkg   *packages.Package
}

func (s *StructFieldsSuite) SetupSuite() {
	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedSyntax,
		Dir:  "testdata",
	}, "")
	s.Require().NoError(err)
	s.Require().Len(pkgs, 1)

	s.pkg = pkgs[0]
	s.Require().NotNil(s.pkg)

	s.scope = s.pkg.Syntax[0].Scope
	s.Require().NotNil(s.scope)
}

func (s *StructFieldsSuite) getStructFields() structure.Fields {
	s.T().Helper()

	obj := s.scope.Lookup("testStruct")
	s.Require().NotNil(obj)

	typ := s.pkg.TypesInfo.TypeOf(obj.Decl.(*ast.TypeSpec).Type) //nolint:forcetypeassert
	s.Require().NotNil(typ)

	return structure.NewFields(typ.Underlying().(*types.Struct)) //nolint:forcetypeassert
}

func (s *StructFieldsSuite) getLiteral(name string) *ast.CompositeLit {
	s.T().Helper()

	obj := s.scope.Lookup(name)
	s.Require().NotNil(obj, "literal %q not found", name)

	lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert
	s.Require().NotNil(lit)

	return lit
}

func (s *StructFieldsSuite) TestNewStructFields() {
	sf := s.getStructFields()

	s.Len(sf, 4)
	s.Equal(structure.Fields{
		{"ExportedRequired", true, false, false},
		{"unexportedRequired", false, false, false},
		{"ExportedOptional", true, false, true},
		{"unexportedOptional", false, false, true},
	}, sf)
}

func (s *StructFieldsSuite) TestStructFields_String() {
	sf := s.getStructFields()

	s.Equal(
		"ExportedRequired, unexportedRequired, ExportedOptional, unexportedOptional",
		sf.String(),
	)
}

func (s *StructFieldsSuite) TestSkipped_Positional_Complete() {
	sf := s.getStructFields()
	lit := s.getLiteral("_unnamed")

	s.Nil(sf.Skipped(lit, true))
	s.Nil(sf.Skipped(lit, false))
}

func (s *StructFieldsSuite) TestSkipped_Positional_Incomplete() {
	sf := s.getStructFields()
	lit := s.getLiteral("_unnamedIncomplete")

	// Positional literals return remaining fields regardless of export status
	s.Equal(structure.Fields{
		{"unexportedRequired", false, false, false},
		{"ExportedOptional", true, false, true},
		{"unexportedOptional", false, false, true},
	}, sf.Skipped(lit, true))
}

func (s *StructFieldsSuite) TestSkipped_Named_Complete() {
	sf := s.getStructFields()
	lit := s.getLiteral("_named")

	s.Nil(sf.Skipped(lit, true))
	s.Nil(sf.Skipped(lit, false))
}

func (s *StructFieldsSuite) TestSkipped_Named_MissingUnexported() {
	sf := s.getStructFields()
	lit := s.getLiteral("_namedIncomplete1")

	// onlyExported=true: unexported fields are not required
	s.Nil(sf.Skipped(lit, true))

	// onlyExported=false: unexported fields are required
	s.Equal(structure.Fields{
		{"unexportedRequired", false, false, false},
	}, sf.Skipped(lit, false))
}

func (s *StructFieldsSuite) TestSkipped_Named_MissingExported() {
	sf := s.getStructFields()
	lit := s.getLiteral("_namedIncomplete2")

	// onlyExported=true: only exported required fields reported
	s.Equal(structure.Fields{
		{"ExportedRequired", true, false, false},
	}, sf.Skipped(lit, true))

	// onlyExported=false: both exported and unexported required fields reported
	s.Equal(structure.Fields{
		{"ExportedRequired", true, false, false},
		{"unexportedRequired", false, false, false},
	}, sf.Skipped(lit, false))
}

func (s *StructFieldsSuite) TestSkipped_Empty() {
	sf := s.getStructFields()
	lit := s.getLiteral("_empty")

	// Empty literal: all required fields are missing
	s.Equal(structure.Fields{
		{"ExportedRequired", true, false, false},
	}, sf.Skipped(lit, true))

	s.Equal(structure.Fields{
		{"ExportedRequired", true, false, false},
		{"unexportedRequired", false, false, false},
	}, sf.Skipped(lit, false))
}

func Test_Fields_Skipped_EmptyStruct(t *testing.T) {
	t.Parallel()

	var emptyFields structure.Fields

	lit := &ast.CompositeLit{Elts: []ast.Expr{}} //nolint:exhaustruct

	require.Nil(t, emptyFields.Skipped(lit, true))
	require.Nil(t, emptyFields.Skipped(lit, false))
}

func Test_NewFields_EmptyStruct(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode: packages.NeedTypes,
		Dir:  "testdata",
	}, "")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	obj := pkg.Types.Scope().Lookup("emptyStruct")
	require.NotNil(t, obj)

	strct := obj.Type().Underlying().(*types.Struct) //nolint:forcetypeassert

	fields := structure.NewFields(strct)
	assert.Empty(t, fields)
	assert.Empty(t, fields.String())
}

func Test_NewFieldsWithDirectives(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode: packages.NeedTypes,
		Dir:  "testdata",
	}, "")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	obj := pkg.Types.Scope().Lookup("testStruct")
	require.NotNil(t, obj)

	strct := obj.Type().Underlying().(*types.Struct) //nolint:forcetypeassert

	t.Run("nil lookup", func(t *testing.T) {
		t.Parallel()

		// With nil lookup, should behave same as NewFields
		fields := structure.NewFieldsWithDirectives(strct, nil)
		assert.Equal(t, structure.Fields{
			{"ExportedRequired", true, false, false},
			{"unexportedRequired", false, false, false},
			{"ExportedOptional", true, false, true},
			{"unexportedOptional", false, false, true},
		}, fields)
	})

	t.Run("directive makes field optional", func(t *testing.T) {
		t.Parallel()

		// Lookup that returns optional for all positions
		lookup := &mockDirectiveLookup{dirs: directive.Directives{directive.Optional}}
		fields := structure.NewFieldsWithDirectives(strct, lookup)

		// All fields should be optional now
		for _, f := range fields {
			assert.True(t, f.Optional, "field %s should be optional", f.Name)
		}
	})

	t.Run("directive does not override tag", func(t *testing.T) {
		t.Parallel()

		// Lookup that returns no directive
		lookup := &mockDirectiveLookup{dirs: nil}
		fields := structure.NewFieldsWithDirectives(strct, lookup)

		// Tag-based optionality should still work
		assert.Equal(t, structure.Fields{
			{"ExportedRequired", true, false, false},
			{"unexportedRequired", false, false, false},
			{"ExportedOptional", true, false, true},    // still optional via tag
			{"unexportedOptional", false, false, true}, // still optional via tag
		}, fields)
	})
}

func Test_FieldsCache_Stats(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode: packages.NeedTypes,
		Dir:  "testdata",
	}, "")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]
	obj := pkg.Types.Scope().Lookup("testStruct")
	require.NotNil(t, obj)

	strct := obj.Type().Underlying().(*types.Struct) //nolint:forcetypeassert

	var cache structure.FieldsCache

	{
		hits, misses := cache.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(0), misses)
	}

	{
		_ = cache.Get(strct, nil)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(1), misses)
	}

	{
		_ = cache.Get(strct, nil)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(1), hits)
		assert.Equal(t, uint64(1), misses)
	}

	{
		_ = cache.Get(strct, nil)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(2), hits)
		assert.Equal(t, uint64(1), misses)
	}
}
