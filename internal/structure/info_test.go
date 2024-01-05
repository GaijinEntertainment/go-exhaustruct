package structure_test

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/tools/go/packages"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/file"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/structure"
)

type InfoSuite struct {
	suite.Suite

	scope *ast.Scope
	pkg   *packages.Package
}

func TestInfo(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(InfoSuite))
}

func (s *InfoSuite) SetupSuite() {
	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedDeps,
		Dir: "testdata",
	}, "./...")
	s.Require().NoError(err)
	s.Require().Len(pkgs, 1)

	s.pkg = pkgs[0]
	s.Require().NotNil(s.pkg)

	s.scope = s.pkg.Syntax[0].Scope
	s.Require().NotNil(s.scope)
}

func (s *InfoSuite) Test_NamedStruct() {
	ac := &file.ASTCache{Mode: parser.ParseComments}

	obj := s.scope.Lookup("_namedTypeVariable")

	s.Require().NotNil(obj)
	lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

	typ := s.pkg.TypesInfo.TypeOf(lit)

	info, err := structure.GetInfo(typ, s.pkg.Fset, ac)
	s.NoError(err) //nolint:testifylint
	s.True(info.IsValid())

	s.Equal("github.com/GaijinEntertainment/go-exhaustruct/v3/internal/structure/testdata.NamedTestType",
		info.String(),
	)
	s.Equal("testdata.NamedTestType", info.ShortString())
	s.NotNil(info.Type)
	s.Equal(structure.Directives{Ignore: true, Enforce: false}, info.Directives)
}

func (s *InfoSuite) Test_AliasStruct() {
	ac := &file.ASTCache{Mode: parser.ParseComments}

	obj := s.scope.Lookup("_aliasNamedTestTypeVariable")

	s.Require().NotNil(obj)
	lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

	typ := s.pkg.TypesInfo.TypeOf(lit)

	info, err := structure.GetInfo(typ, s.pkg.Fset, ac)
	s.NoError(err) //nolint:testifylint
	s.True(info.IsValid())

	s.Equal("github.com/GaijinEntertainment/go-exhaustruct/v3/internal/structure/testdata.AliasNamedTestType",
		info.String(),
	)
	s.Equal("testdata.AliasNamedTestType", info.ShortString())
	s.NotNil(info.Type)
	s.Equal(structure.Directives{Ignore: false, Enforce: true}, info.Directives)
}
