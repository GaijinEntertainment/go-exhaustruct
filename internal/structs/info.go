package structs

import (
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

// AnonymousName is the name used for anonymous structs.
const AnonymousName = "<anonymous>"

// Info represents a struct type with its analysis metadata.
type Info struct {
	// Name is the struct type name.
	Name string
	// PackagePath is the full import path of the package where the struct is defined.
	PackagePath string
	// Position is the source location where the struct is defined.
	Position token.Position

	// Fields contains metadata for each struct field in declaration order.
	Fields Fields `exhaustruct:"optional"`

	// Enforced indicates the struct must be checked even if excluded by pattern.
	Enforced bool `exhaustruct:"optional"`
	// Ignored indicates the struct should be skipped from checking.
	Ignored bool `exhaustruct:"optional"`
	// Optional indicates all fields of this struct are treated as optional.
	Optional bool `exhaustruct:"optional"`
}

// NewInfo creates a new Info from a struct type with directive lookup.
// The fset is used to convert positions to line/column information.
// The name is the type name (use [AnonymousName] for anonymous structs).
// The pkg provides package path information.
// The pos is the type definition position.
// The lookup is used to check for directives at the struct and field positions.
// Returns the Info and any diagnostics from directive parsing.
func NewInfo(
	fset *token.FileSet,
	strct *types.Struct,
	name string,
	pkg *types.Package,
	pos token.Pos,
	lookup DirectiveLookup,
) (*Info, []analysis.Diagnostic) {
	res := Info{
		Name:        name,
		PackagePath: pkg.Path(),
		Position:    fset.Position(pos),
	}

	var allDiags []analysis.Diagnostic

	if lookup != nil {
		directives, diags := lookup.Lookup(fset, res.Position)

		allDiags = append(allDiags, diags...)

		res.Enforced = slices.Contains(directives, directive.Enforce)
		res.Ignored = slices.Contains(directives, directive.Ignore)
		res.Optional = slices.Contains(directives, directive.Optional)
	}

	fields, diags := newFields(fset, strct, lookup)

	allDiags = append(allDiags, diags...)
	res.Fields = fields

	return &res, allDiags
}

// DirectiveLookup provides position-based directive lookup.
// This interface is satisfied by [directive.FileCache].
type DirectiveLookup interface {
	// Lookup returns the directives at the given source position.
	Lookup(fset *token.FileSet, pos token.Position) (directive.Directives, []analysis.Diagnostic)
}
