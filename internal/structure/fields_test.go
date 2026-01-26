package structure_test

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/tools/go/packages"

	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

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
		Mode:       packages.NeedTypes | packages.NeedTypesInfo | packages.NeedTypesSizes | packages.NeedSyntax,
		Dir:        "testdata",
		Context:    nil,
		Logf:       nil,
		Env:        nil,
		BuildFlags: nil,
		Fset:       nil,
		ParseFile:  nil,
		Tests:      false,
		Overlay:    nil,
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
	s.assertFieldsMatch(sf, []fieldMeta{
		{"ExportedRequired", true, false},
		{"unexportedRequired", false, false},
		{"ExportedOptional", true, true},
		{"unexportedOptional", false, true},
	})
}

// fieldMeta is a helper type for comparing field properties without Type.
type fieldMeta struct {
	Name     string
	Exported bool
	Optional bool
}

// assertFieldsMatch compares Fields by Name, Exported, and Optional only.
func (s *StructFieldsSuite) assertFieldsMatch(actual structure.Fields, expected []fieldMeta) {
	s.T().Helper()
	s.Require().Len(actual, len(expected))
	for i, f := range actual {
		s.Equal(expected[i].Name, f.Name, "field %d name mismatch", i)
		s.Equal(expected[i].Exported, f.Exported, "field %d exported mismatch", i)
		s.Equal(expected[i].Optional, f.Optional, "field %d optional mismatch", i)
	}
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
	s.assertFieldsMatch(sf.Skipped(lit, true), []fieldMeta{
		{"unexportedRequired", false, false},
		{"ExportedOptional", true, true},
		{"unexportedOptional", false, true},
	})
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
	s.assertFieldsMatch(sf.Skipped(lit, false), []fieldMeta{
		{"unexportedRequired", false, false},
	})
}

func (s *StructFieldsSuite) TestSkipped_Named_MissingExported() {
	sf := s.getStructFields()
	lit := s.getLiteral("_namedIncomplete2")

	// onlyExported=true: only exported required fields reported
	s.assertFieldsMatch(sf.Skipped(lit, true), []fieldMeta{
		{"ExportedRequired", true, false},
	})

	// onlyExported=false: both exported and unexported required fields reported
	s.assertFieldsMatch(sf.Skipped(lit, false), []fieldMeta{
		{"ExportedRequired", true, false},
		{"unexportedRequired", false, false},
	})
}

func (s *StructFieldsSuite) TestSkipped_Empty() {
	sf := s.getStructFields()
	lit := s.getLiteral("_empty")

	// Empty literal: all required fields are missing
	s.assertFieldsMatch(sf.Skipped(lit, true), []fieldMeta{
		{"ExportedRequired", true, false},
	})

	s.assertFieldsMatch(sf.Skipped(lit, false), []fieldMeta{
		{"ExportedRequired", true, false},
		{"unexportedRequired", false, false},
	})
}

func Test_Fields_Skipped_EmptyStruct(t *testing.T) {
	t.Parallel()

	var emptyFields structure.Fields

	lit := &ast.CompositeLit{Elts: []ast.Expr{}, Type: nil, Lbrace: 0, Rbrace: 0, Incomplete: false} //nolint:exhaustruct

	require.Nil(t, emptyFields.Skipped(lit, true))
	require.Nil(t, emptyFields.Skipped(lit, false))
}

func Test_NewFields_EmptyStruct(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode:       packages.NeedTypes,
		Dir:        "testdata",
		Context:    nil,
		Logf:       nil,
		Env:        nil,
		BuildFlags: nil,
		Fset:       nil,
		ParseFile:  nil,
		Tests:      false,
		Overlay:    nil,
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

func Test_FieldsCache_Stats(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode:       packages.NeedTypes,
		Dir:        "testdata",
		Context:    nil,
		Logf:       nil,
		Env:        nil,
		BuildFlags: nil,
		Fset:       nil,
		ParseFile:  nil,
		Tests:      false,
		Overlay:    nil,
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
		_ = cache.Get(strct)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(1), misses)
	}

	{
		_ = cache.Get(strct)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(1), hits)
		assert.Equal(t, uint64(1), misses)
	}

	{
		_ = cache.Get(strct)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(2), hits)
		assert.Equal(t, uint64(1), misses)
	}
}
