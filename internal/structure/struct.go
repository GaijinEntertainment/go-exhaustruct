package structure

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

// AnonymousName is the name used for anonymous structs.
const AnonymousName = "<anonymous>"

// SkippedFields returns fields that are missing from the literal but required.
// For positional literals, returns remaining fields after the last provided element,
// filtering by accessibility when externalPkg is true.
// For named literals, applies requirement rules based on directives and accessibility.
// The externalPkg flag indicates if the struct is from an external package
// (unexported fields are inaccessible and thus not required).
func (s *Struct) SkippedFields(lit *ast.CompositeLit, externalPkg bool) Fields {
	if len(lit.Elts) != 0 && !isNamedLiteral(lit) {
		return s.skippedPositional(len(lit.Elts), externalPkg)
	}

	return s.skippedNamed(lit, externalPkg)
}

// skippedPositional returns remaining fields after the given count for positional literals.
func (s *Struct) skippedPositional(count int, externalPkg bool) Fields {
	remaining := s.Fields[count:]

	if !externalPkg {
		if count >= len(s.Fields) {
			return nil
		}

		return slices.Clone(s.Fields[count:])
	}

	res := make(Fields, 0, len(remaining))

	for _, f := range remaining {
		if f.Exported {
			res = append(res, f)
		}
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

// skippedNamed returns missing required fields for named literals.
func (s *Struct) skippedNamed(lit *ast.CompositeLit, externalPkg bool) Fields {
	present := make(map[string]bool, len(lit.Elts))

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if k, ok := kv.Key.(*ast.Ident); ok {
				present[k.Name] = true
			}
		}
	}

	res := make(Fields, 0, len(s.Fields)-len(present))

	for _, f := range s.Fields {
		if !present[f.Name] && s.isFieldRequired(f, externalPkg) {
			res = append(res, f)
		}
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

// isFieldRequired returns true if the field must be present in a literal.
func (s *Struct) isFieldRequired(f Field, externalPkg bool) bool {
	if f.Enforced {
		return true
	}

	if f.Optional || s.Optional {
		return false
	}

	if externalPkg && !f.Exported {
		return false
	}

	return true
}

// isNamedLiteral returns true if the literal uses named fields.
// Panics if the literal is empty.
func isNamedLiteral(lit *ast.CompositeLit) bool {
	_, ok := lit.Elts[0].(*ast.KeyValueExpr)
	return ok
}

// Struct represents a struct type with its analysis metadata.
type Struct struct {
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

// NewStruct creates a new Struct from a struct type with directive lookup.
// The fset is used to convert positions to line/column information.
// The name is the type name (use [AnonymousName] for anonymous structs).
// The pkg provides package path information.
// The pos is the type definition position.
// The lookup is used to check for directives at the struct and field positions.
// Returns the Struct and any diagnostics from directive parsing.
func NewStruct(
	fset *token.FileSet,
	strct *types.Struct,
	name string,
	pkg *types.Package,
	pos token.Pos,
	lookup *directive.FileCache,
) (*Struct, []analysis.Diagnostic) {
	res := Struct{
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
