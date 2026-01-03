package structure

import (
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

// Field represents a single struct field with its analysis metadata.
type Field struct {
	// Name is the field name.
	Name string
	// Exported indicates whether the field is exported (starts with uppercase).
	Exported bool `exhaustruct:"optional"`

	// Enforced indicates the field must be initialized even if struct is optional.
	Enforced bool `exhaustruct:"optional"`
	// Optional indicates the field can be omitted from initialization.
	Optional bool `exhaustruct:"optional"`
}

// Fields is a collection of struct fields. It contains metadata about each field
// in order of declaration. It is crucial to keep the order, since non-named init
// relies on it.
type Fields []Field

// String returns a comma-separated list of field names.
func (sf Fields) String() string {
	switch len(sf) {
	case 0:
		return ""
	case 1:
		return sf[0].Name
	}

	var b strings.Builder
	b.Grow(len(sf))
	b.WriteString(sf[0].Name)

	for _, s := range sf[1:] {
		b.WriteString(", ")
		b.WriteString(s.Name)
	}

	return b.String()
}

func newFields(fset *token.FileSet, strct *types.Struct, lookup *directive.FileCache) (Fields, []analysis.Diagnostic) {
	fields := make(Fields, 0, strct.NumFields())

	var allDiags []analysis.Diagnostic

	for f := range strct.Fields() {
		res := Field{
			Name:     f.Name(),
			Exported: f.Exported(),
		}

		if lookup != nil {
			fieldPos := fset.Position(f.Pos())
			directives, diags := lookup.Lookup(fset, fieldPos)

			allDiags = append(allDiags, diags...)

			res.Enforced = slices.Contains(directives, directive.Enforce)
			res.Optional = slices.Contains(directives, directive.Optional)
		}

		fields = append(fields, res)
	}

	return fields, allDiags
}
