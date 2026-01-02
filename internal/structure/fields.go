package structure

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

const (
	tagName          = "exhaustruct"
	optionalTagValue = "optional"
)

// Field represents a single struct field with its analysis metadata.
type Field struct {
	// Name is the field name.
	Name string
	// Exported indicates whether the field is exported (starts with uppercase).
	Exported bool

	// Enforced indicates the field must be initialized even if struct is optional.
	Enforced bool
	// Optional indicates the field can be omitted from initialization.
	Optional bool
}

// Fields is a collection of struct fields. It contains metadata about each field
// in order of declaration. It is crucial to keep the order, since non-named init
// relies on it.
type Fields []*Field

// DirectiveLookup provides position-based directive lookup for struct fields.
// Implementations resolve the source position to check for comment directives.
type DirectiveLookup interface {
	// Lookup returns the directives at the given source position.
	Lookup(pos token.Pos) directive.Directives
}

// NewFields creates a new [Fields] from a given struct type.
// Fields items are listed in order they appear in the struct.
// Optional fields are determined only by struct tags.
func NewFields(strct *types.Struct) Fields {
	return NewFieldsWithDirectives(strct, nil)
}

// NewFieldsWithDirectives creates a new [Fields] from a given struct type,
// using both struct tags and comment directives to determine optionality.
// The lookup is used to check for //exhaustruct:optional directives at field positions.
func NewFieldsWithDirectives(strct *types.Struct, lookup DirectiveLookup) Fields {
	sf := make(Fields, 0, strct.NumFields())

	for i := range strct.NumFields() {
		f := strct.Field(i)

		optional := HasOptionalTag(strct.Tag(i))
		if !optional && lookup != nil {
			optional = lookup.Lookup(f.Pos()).Contains(directive.Optional)
		}

		sf = append(sf, &Field{ //nolint:exhaustruct // Enforced is computed later
			Name:     f.Name(),
			Exported: f.Exported(),
			Optional: optional,
		})
	}

	return sf
}

// HasOptionalTag checks if the given struct tag contains exhaustruct:"optional".
func HasOptionalTag(tags string) bool {
	return reflect.StructTag(tags).Get(tagName) == optionalTagValue
}

// String returns a comma-separated list of field names.
func (sf Fields) String() string {
	b := strings.Builder{}

	for _, f := range sf {
		if b.Len() != 0 {
			b.WriteString(", ")
		}

		b.WriteString(f.Name)
	}

	return b.String()
}

// Skipped returns a list of fields that are not present in the given
// literal, but expected to.
func (sf Fields) Skipped(lit *ast.CompositeLit, onlyExported bool) Fields {
	if len(lit.Elts) != 0 && !isNamedLiteral(lit) {
		if len(lit.Elts) == len(sf) {
			return nil
		}

		return sf[len(lit.Elts):]
	}

	present := presentNamedFields(lit)
	res := make(Fields, 0, len(sf)-len(present))

	for _, f := range sf {
		if present[f.Name] || f.Optional || (!f.Exported && onlyExported) {
			continue
		}

		res = append(res, f)
	}

	if len(res) == 0 {
		return nil
	}

	return res
}

// presentNamedFields returns a map of field names that are present in the literal.
func presentNamedFields(lit *ast.CompositeLit) map[string]bool {
	m := make(map[string]bool, len(lit.Elts))

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		k, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		m[k.Name] = true
	}

	return m
}

// isNamedLiteral returns true if the given literal uses named fields.
//
// The logic is based on the principle that a literal is either named or positional,
// therefore if the first element is a [ast.KeyValueExpr], it is named.
//
// Method will panic if the given literal is empty.
func isNamedLiteral(lit *ast.CompositeLit) bool {
	_, ok := lit.Elts[0].(*ast.KeyValueExpr)
	return ok
}
