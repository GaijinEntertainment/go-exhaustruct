package fields

import (
	"go/ast"
	"go/types"
	"reflect"
)

const (
	TagName          = "exhaustruct"
	OptionalTagValue = "optional"
)

type StructField struct {
	Name     string
	Exported bool
	Optional bool
}

type StructFields []StructField

// NewStructFields creates a new [StructFields] from a given struct type.
// StructFields items are listed in order they appear in the struct.
func NewStructFields(strct *types.Struct) StructFields {
	sf := make(StructFields, strct.NumFields())

	for i := 0; i < strct.NumFields(); i++ {
		f := strct.Field(i)

		sf[i] = StructField{
			Name:     f.Name(),
			Exported: f.Exported(),
			Optional: HasOptionalTag(strct.Tag(i)),
		}
	}

	return sf
}

func HasOptionalTag(tags string) bool {
	return reflect.StructTag(tags).Get(TagName) == OptionalTagValue
}

// String returns a comma-separated list of field names.
func (sf StructFields) String() (res string) {
	for i := 0; i < len(sf); i++ {
		if res != "" {
			res += ", "
		}

		res += sf[i].Name
	}

	return res
}

// SkippedFields returns a list of fields that are not present in the given
// literal, but expected to.
//
//revive:disable-next-line:cyclomatic
func (sf StructFields) SkippedFields(lit *ast.CompositeLit, onlyExported bool) (res StructFields) {
	if len(lit.Elts) != 0 && !isNamedLiteral(lit) {
		if len(lit.Elts) == len(sf) {
			return nil
		}

		return sf[len(lit.Elts):]
	}

	pm := sf.positionsMap()

	for i := 0; i < len(lit.Elts); i++ {
		kv, ok := lit.Elts[i].(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		k, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		delete(pm, k.Name)
	}

	for i := 0; i < len(sf); i++ {
		if _, ok := pm[sf[i].Name]; !ok || (!sf[i].Exported && onlyExported) || sf[i].Optional {
			continue
		}

		res = append(res, sf[i])
	}

	return res
}

func (sf StructFields) positionsMap() map[string]int {
	m := make(map[string]int, len(sf))

	for i := 0; i < len(sf); i++ {
		m[sf[i].Name] = i
	}

	return m
}

// isNamedLiteral returns true if the given literal is unnamed.
//
// The logic is basing on the principle that literal is named or unnamed,
// therefore is literal's first element is a [ast.KeyValueExpr], it is named.
//
// Method will panic if the given literal is empty.
func isNamedLiteral(lit *ast.CompositeLit) bool {
	if _, ok := lit.Elts[0].(*ast.KeyValueExpr); !ok {
		return false
	}

	return true
}
