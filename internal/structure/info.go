package structure

import (
	"errors"
	"go/ast"
	"go/types"
	"unsafe"

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

type InfoParser struct {
	pkg *types.Package
	ac  *file.ASTCache
	ti  *types.Info

	IC *InfoCache
}

func NewInfoParser(ic *InfoCache, pkg *types.Package, ac *file.ASTCache, ti *types.Info) InfoParser {
	return InfoParser{
		pkg: pkg,
		ac:  ac,
		ti:  ti,
		IC:  ic,
	}
}

// LitInfo returns structure information for a given type if it of a structure
// type.
func (p InfoParser) LitInfo(lit *ast.CompositeLit) (Info, error) {
	switch typ := p.ti.TypeOf(lit).(type) {
	case *types.Named: // named type
		structType, ok := typ.Underlying().(*types.Struct)
		if !ok {
			return Info{}, nil
		}

		ident, ok := lit.Type.(*ast.Ident)
		if ok { // named type alias
			return p.IC.Get(
				uintptr(unsafe.Pointer(ident)),
				func() (Info, error) { return p.byIdent(ident, structType) },
			)
		}

		return p.IC.Get(
			uintptr(unsafe.Pointer(typ)),
			func() (Info, error) { return p.byNamedType(typ, structType) },
		)

	case *types.Struct: // anonymous type
		ident, ok := lit.Type.(*ast.Ident)
		if ok { // named type alias
			return p.IC.Get(
				uintptr(unsafe.Pointer(ident)),
				func() (Info, error) { return p.byIdent(ident, typ) },
			)
		}

		astStruct, ok := lit.Type.(*ast.StructType)
		if !ok {
			return Info{}, nil
		}

		return p.IC.Get(
			uintptr(unsafe.Pointer(astStruct)),
			func() (Info, error) { return p.byAstStruct(astStruct, typ) },
		)
	}

	return Info{}, nil
}

func (p InfoParser) byNamedType(named *types.Named, typ *types.Struct) (Info, error) {
	obj := named.Obj()

	gd, err := p.ac.FindTypeNameGenDecl(obj)
	if err != nil && !errors.Is(err, file.ErrNotFound) {
		return Info{}, err //nolint:wrapcheck
	}

	var dir Directives

	if gd != nil {
		relatedComments, err := p.ac.RelatedComments(gd)
		if err != nil && !errors.Is(err, file.ErrNotFound) {
			return Info{}, err //nolint:wrapcheck
		}

		dir = parseStructDirectives(relatedComments)
	}

	pkg := obj.Pkg()

	i := Info{
		valid: true,

		Name:        obj.Name(),
		PackageName: pkg.Name(),
		PackagePath: pkg.Path(),
		Directives:  dir,
		Type:        typ,
		Fields:      NewFields(typ),
	}

	return i, nil
}

func (p InfoParser) byIdent(ident *ast.Ident, typ *types.Struct) (Info, error) {
	gd, err := p.ac.FindIdentGenDecl(ident)
	if err != nil && !errors.Is(err, file.ErrNotFound) {
		return Info{}, err //nolint:wrapcheck
	}

	var relatedComments []*ast.CommentGroup

	if gd != nil && gd.Doc != nil {
		relatedComments = append(relatedComments, gd.Doc)
	} else {
		relatedComments, err = p.ac.RelatedComments(ident)
		if err != nil && !errors.Is(err, file.ErrNotFound) {
			return Info{}, err //nolint:wrapcheck
		}
	}

	obj := p.ti.ObjectOf(ident)
	pkg := obj.Pkg()

	i := Info{
		valid: true,

		Name:        ident.Name,
		PackageName: pkg.Name(),
		PackagePath: pkg.Path(),
		Directives:  parseStructDirectives(relatedComments),
		Type:        typ,
		Fields:      NewFields(typ),
	}

	return i, nil
}

func (p InfoParser) byAstStruct(astStruct *ast.StructType, typ *types.Struct) (Info, error) {
	relatedComments, err := p.ac.RelatedComments(astStruct)
	if err != nil && !errors.Is(err, file.ErrNotFound) {
		return Info{}, err //nolint:wrapcheck
	}

	return Info{
		valid:       true,
		Name:        "<anonymous>",
		PackageName: p.pkg.Name(),
		PackagePath: p.pkg.Path(),
		Directives:  parseStructDirectives(relatedComments),
		Type:        typ,
		Fields:      NewFields(typ),
	}, nil
}

func (p InfoParser) parseStructFields(strct *types.Struct) Fields {
	sf := make(Fields, 0, strct.NumFields())

	for i := 0; i < strct.NumFields(); i++ {
		f := strct.Field(i)

		sf = append(sf, Field{
			Name:     f.Name(),
			Exported: f.Exported(),
			Optional: HasOptionalTag(strct.Tag(i)),
		})
	}

	return sf
}

func parseStructDirectives(comments []*ast.CommentGroup) (d Directives) {
	if len(comments) == 0 {
		return d
	}

	d.Ignore = comment.HasDirective(comments, comment.DirectiveIgnore)
	d.Enforce = comment.HasDirective(comments, comment.DirectiveEnforce)

	return d
}
