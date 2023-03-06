package fields_test

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/tools/go/packages"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/fields"
)

func Test_HasOptionalTag(t *testing.T) {
	t.Parallel()

	assert.True(t, fields.HasOptionalTag(`exhaustruct:"optional"`))
	assert.False(t, fields.HasOptionalTag(`exhaustruct:"required"`))
}

func TestStructFields(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(StructFieldsSuite))
}

type StructFieldsSuite struct {
	suite.Suite

	scope *ast.Scope
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

func (s *StructFieldsSuite) getReferenceStructFields() fields.StructFields {
	s.T().Helper()

	obj := s.scope.Lookup("testStruct")
	s.Require().NotNil(obj)

	typ := s.pkg.TypesInfo.TypeOf(obj.Decl.(*ast.TypeSpec).Type) //nolint:forcetypeassert
	s.Require().NotNil(typ)

	return fields.NewStructFields(typ.Underlying().(*types.Struct)) //nolint:forcetypeassert
}

func (s *StructFieldsSuite) TestNewStructFields() {
	sf := s.getReferenceStructFields()

	s.Assert().Len(sf, 4)
	s.Assert().Equal(fields.StructFields{
		{
			Name:     "ExportedRequired",
			Exported: true,
			Optional: false,
		},
		{
			Name:     "unexportedRequired",
			Exported: false,
			Optional: false,
		},
		{
			Name:     "ExportedOptional",
			Exported: true,
			Optional: true,
		},
		{
			Name:     "unexportedOptional",
			Exported: false,
			Optional: true,
		},
	}, sf)
}

func (s *StructFieldsSuite) TestStructFields_String() {
	sf := s.getReferenceStructFields()

	s.Assert().Equal(
		"ExportedRequired, unexportedRequired, ExportedOptional, unexportedOptional",
		sf.String(),
	)
}

func (s *StructFieldsSuite) TestStructFields_SkippedFields_Unnamed() {
	sf := s.getReferenceStructFields()

	unnamed := s.scope.Lookup("_unnamed")
	if s.Assert().NotNil(unnamed) {
		lit := unnamed.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert
		if s.Assert().NotNil(lit) {
			s.Assert().Nil(sf.SkippedFields(lit, true))
			s.Assert().Nil(sf.SkippedFields(lit, false))
		}
	}

	unnamedIncomplete := s.scope.Lookup("_unnamedIncomplete")
	if s.Assert().NotNil(unnamedIncomplete) {
		lit := unnamedIncomplete.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert
		if s.Assert().NotNil(lit) {
			s.Assert().Equal(fields.StructFields{
				{"unexportedRequired", false, false},
				{"ExportedOptional", true, true},
				{"unexportedOptional", false, true},
			}, sf.SkippedFields(lit, true))
		}
	}
}

func (s *StructFieldsSuite) TestStructFields_SkippedFields_Named() {
	sf := s.getReferenceStructFields()

	named := s.scope.Lookup("_named")
	if s.Assert().NotNil(named) {
		lit := named.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert
		if s.Assert().NotNil(lit) {
			s.Assert().Nil(sf.SkippedFields(lit, true))
			s.Assert().Nil(sf.SkippedFields(lit, false))
		}
	}

	namedIncomplete1 := s.scope.Lookup("_namedIncomplete1")
	if s.Assert().NotNil(namedIncomplete1) {
		lit := namedIncomplete1.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert
		if s.Assert().NotNil(lit) {
			s.Assert().Nil(sf.SkippedFields(lit, true))
			s.Assert().Equal(fields.StructFields{
				{"unexportedRequired", false, false},
			}, sf.SkippedFields(lit, false))
		}
	}

	namedIncomplete2 := s.scope.Lookup("_namedIncomplete2")
	if s.Assert().NotNil(namedIncomplete2) {
		lit := namedIncomplete2.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert
		if s.Assert().NotNil(lit) {
			s.Assert().Equal(fields.StructFields{
				{"ExportedRequired", true, false},
			}, sf.SkippedFields(lit, true))
			s.Assert().Equal(fields.StructFields{
				{"ExportedRequired", true, false},
				{"unexportedRequired", false, false},
			}, sf.SkippedFields(lit, false))
		}
	}
}
