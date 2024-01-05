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

	scope      *ast.Scope
	infoParser structure.InfoParser

	packagePath string
	packageName string
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
	})

	s.Require().NoError(err)
	s.Require().Len(pkgs, 1)
	s.Require().NotNil(pkgs[0])
	s.Require().NotNil(pkgs[0].Syntax[0].Scope)

	ac := &file.ASTCache{Mode: parser.ParseComments, FS: pkgs[0].Fset}
	for _, f := range pkgs[0].Syntax {
		ac.AddFiles(f)
	}

	s.packagePath = `github.com/GaijinEntertainment/go-exhaustruct/v3/internal/structure/testdata`
	s.packageName = `testdata`

	s.infoParser = structure.NewInfoParser(&structure.InfoCache{}, pkgs[0].Types, ac, pkgs[0].TypesInfo)
	s.scope = pkgs[0].Syntax[0].Scope
}

func (s *InfoSuite) Test_NamedTypeStruct() {
	{
		obj := s.scope.Lookup("_NamedTestTypeVariable")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".NamedTestType", info.String())
		s.Equal(s.packageName+".NamedTestType", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: true, Enforce: false}, info.Directives)
	}

	{
		obj := s.scope.Lookup("_NamedTestType2Variable")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".NamedTestType2", info.String())
		s.Equal(s.packageName+".NamedTestType2", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: false, Enforce: true}, info.Directives)
	}

	{
		obj := s.scope.Lookup("_NamedTestType3Variable")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".NamedTestType3", info.String())
		s.Equal(s.packageName+".NamedTestType3", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: true, Enforce: false}, info.Directives)
	}
}

func (s *InfoSuite) Test_AliasTypeStruct() {
	{
		obj := s.scope.Lookup("_AliasTestTypeVariable")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".AliasTestType", info.String())
		s.Equal(s.packageName+".AliasTestType", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: false, Enforce: true}, info.Directives)
	}

	{
		obj := s.scope.Lookup("_AliasImportedTestTypeVariable")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".AliasImportedTestType", info.String())
		s.Equal(s.packageName+".AliasImportedTestType", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: true, Enforce: false}, info.Directives)
	}
}

func (s *InfoSuite) Test_AnonymousTypeStruct() {
	{
		obj := s.scope.Lookup("_AnonymousAliasTestTypeVariable")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".AnonymousAliasTestType", info.String())
		s.Equal(s.packageName+".AnonymousAliasTestType", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: true, Enforce: false}, info.Directives)
	}

	{
		obj := s.scope.Lookup("_AnonymousAliasEmptyTestTypeVariable")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".AnonymousAliasEmptyTestType", info.String())
		s.Equal(s.packageName+".AnonymousAliasEmptyTestType", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: false, Enforce: true}, info.Directives)
	}

	{
		obj := s.scope.Lookup("_AnonymousTestTypeVariable")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".<anonymous>", info.String()) //nolint:goconst
		s.Equal(s.packageName+".<anonymous>", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: true, Enforce: false}, info.Directives)
	}

	{
		obj := s.scope.Lookup("_AnonymousTestTypeVariable2")

		s.Require().NotNil(obj)
		lit := obj.Decl.(*ast.ValueSpec).Values[0].(*ast.CompositeLit) //nolint:forcetypeassert

		info, err := s.infoParser.LitInfo(lit)
		s.NoError(err) //nolint:testifylint
		s.True(info.IsValid())

		s.Equal(s.packagePath+".<anonymous>", info.String())
		s.Equal(s.packageName+".<anonymous>", info.ShortString())
		s.NotNil(info.Type)
		s.Equal(structure.Directives{Ignore: false, Enforce: true}, info.Directives)
	}
}
