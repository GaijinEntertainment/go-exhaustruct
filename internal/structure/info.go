package structure

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/comment"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/file"
)

type Info struct {
	valid bool `exhaustruct:"optional"`

	Name        string        `exhaustruct:"optional"`
	PackageName string        `exhaustruct:"optional"`
	PackagePath string        `exhaustruct:"optional"`
	Directives  Directives    `exhaustruct:"optional"`
	Fields      Fields        `exhaustruct:"optional"`
	Type        *types.Struct `exhaustruct:"optional"`
}

type Directives struct {
	Ignore  bool
	Enforce bool
}

// IsValid returns true if the type is a structure type and successfully parsed.
func (t Info) IsValid() bool {
	return t.valid
}

func (t Info) String() string {
	return t.PackagePath + "." + t.Name
}

func (t Info) ShortString() string {
	return t.PackageName + "." + t.Name
}

// GetInfo returns structure information for a given type if it of a structure
// type.
func GetInfo(t types.Type, fs *token.FileSet, ac *file.ASTCache) (Info, error) {
	switch typ := t.(type) {
	case *types.Named: // named type
		return namedStructInfo(typ, fs, ac)

	case *types.Struct: // anonymous type

		// TODO: Implement.
		return Info{}, nil
	}

	return Info{}, nil
}

func namedStructInfo(typ *types.Named, fs *token.FileSet, ac *file.ASTCache) (Info, error) {
	structType, ok := typ.Underlying().(*types.Struct)
	if !ok {
		return Info{}, nil
	}

	obj := typ.Obj()

	decl, err := ac.FindTypeNameGenDecl(fs, obj)
	if err != nil {
		return Info{}, err //nolint:wrapcheck
	}

	pkg := obj.Pkg()
	i := Info{
		valid: true,

		Name:        obj.Name(),
		PackageName: pkg.Name(),
		PackagePath: pkg.Path(),
		Directives:  parseDocDirectives(decl.Doc),
		Type:        structType,
	}

	return i, nil
}

func parseDocDirectives(doc *ast.CommentGroup) Directives {
	cg := make([]*ast.CommentGroup, 0, 1)
	if doc != nil {
		cg = append(cg, doc)
	}

	return Directives{
		Ignore:  comment.HasDirective(cg, comment.DirectiveIgnore),
		Enforce: comment.HasDirective(cg, comment.DirectiveEnforce),
	}
}
