package structure

import (
	"go/ast"
	"go/types"
	"reflect"
	"strings"
)

const (
	tagName          = "exhaustruct"
	optionalTagValue = "optional"
)

// Field represents a single struct field with its metadata.
type Field struct {
	Name     string
	Type     types.Type
	Exported bool
	Optional bool
}

// Fields is a collection of struct fields. It contains metadata about each field
// in order of declaration. It is crucial to keep the order, since non-named init
// relies on it.
type Fields []*Field

// NewFields creates a new [Fields] from a given struct type.
// Fields items are listed in order they appear in the struct.
func NewFields(strct *types.Struct) Fields {
	sf := make(Fields, 0, strct.NumFields())

	for i := range strct.NumFields() {
		f := strct.Field(i)

		sf = append(sf, &Field{
			Name:     f.Name(),
			Type:     f.Type(),
			Exported: f.Exported(),
			Optional: HasOptionalTag(strct.Tag(i)),
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
