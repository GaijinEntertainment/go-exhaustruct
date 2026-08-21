package structure

import (
	"go/ast"
	"go/token"
	"strings"
)

const AnonymousName = "<anonymous>"

// BlankFieldName is the name go/types gives a blank struct field.
const BlankFieldName = "_"

// outermost marks the promotion walk's first level, where a field owns itself
// rather than inheriting an owner from the level above.
const outermost = -1

// fieldInfo contains raw field data independent of type name.
type fieldInfo struct {
	// name is the name of the field.
	name string
	// exported indicates if the field is exported.
	exported bool //exhaustruct:optional
	// enforced indicates if the field is enforced via directive.
	enforced bool //exhaustruct:optional
	// optional indicates if the field is optional via directive.
	optional bool //exhaustruct:optional
	// embedded holds the fields an embedded field promotes, nil otherwise.
	embedded *structFields //exhaustruct:optional
}

// structFields contains field information for a struct, independent of type name.
type structFields struct {
	// packagePath is the package path where fields are declared.
	packagePath string
	// fields is the list of fields in declaration order.
	fields []fieldInfo
}

type Struct struct {
	Name        string
	FullPath    string
	PackageName string

	Position token.Position //exhaustruct:optional
	Fields   Fields         //exhaustruct:optional

	Enforced bool //exhaustruct:optional
	Ignored  bool //exhaustruct:optional
	Optional bool //exhaustruct:optional

	PatternEnforced bool //exhaustruct:optional
	PatternIgnored  bool //exhaustruct:optional
	PatternOptional bool //exhaustruct:optional

	AllowEmptyDecl bool //exhaustruct:optional

	// Detected via OriginScanner AST inspection before types.Unalias.
	IsAlias   bool //exhaustruct:optional
	IsDerived bool //exhaustruct:optional
}

// PackagePath returns the package path of the struct type.
func (s *Struct) PackagePath() string {
	if idx := strings.LastIndex(s.FullPath, "."); idx >= 0 {
		return s.FullPath[:idx]
	}

	return s.FullPath
}

// IsEnforced returns true if struct is enforced via directive or pattern.
func (s *Struct) IsEnforced() bool {
	return s.Enforced || s.PatternEnforced
}

// IsIgnored returns true if struct is ignored via directive or pattern.
func (s *Struct) IsIgnored() bool {
	return s.Ignored || s.PatternIgnored
}

// IsOptional returns true if struct is optional via directive or pattern.
func (s *Struct) IsOptional() bool {
	return s.Optional || s.PatternOptional
}

// SkippedFields returns missing required fields for a composite literal.
// callerPkgPath is used to determine if unexported fields are accessible.
// For positional literals: returns fields after the last provided element.
// For named literals: returns fields not present in the literal.
func (s *Struct) SkippedFields(lit *ast.CompositeLit, callerPkgPath string) []Field {
	if isNamedLiteral(lit) {
		return s.skippedNamed(lit, callerPkgPath)
	}

	return s.skippedPositional(len(lit.Elts), s.Fields.PackagePath != callerPkgPath)
}

// isNamedLiteral checks if a composite literal uses named fields. It treats
// empty literals as not named, since positional literals checks are simpler.
func isNamedLiteral(lit *ast.CompositeLit) bool {
	if len(lit.Elts) == 0 {
		return false
	}

	_, ok := lit.Elts[0].(*ast.KeyValueExpr)

	return ok
}

func (s *Struct) skippedPositional(count int, externalPkg bool) []Field {
	items := s.Fields.Items

	if count >= len(items) {
		return nil
	}

	remaining := items[count:]
	missing := make([]Field, 0, len(remaining))

	for _, f := range remaining {
		if s.isFieldRequired(f, externalPkg) {
			missing = append(missing, f)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return missing
}

func (s *Struct) skippedNamed(lit *ast.CompositeLit, callerPkgPath string) []Field {
	present := make(map[string]bool, len(lit.Elts))

	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if k, ok := kv.Key.(*ast.Ident); ok {
				present[k.Name] = true
			}
		}
	}

	missing := s.skippedIn(&s.Fields, present, callerPkgPath)

	if len(missing) == 0 {
		return nil
	}

	return missing
}

// skippedIn returns the required fields of fs that keys leaves uninitialized.
//
// Since Go 1.27 a composite literal may name a promoted field in place of the
// embedded field carrying it (golang/go#77245), so keys is flat: a name may
// belong to fs directly or to any struct embedded within it. Naming both a
// promoted field and its enclosing embedded field does not compile, so an
// embedded field is either named directly, partially filled through promoted
// names — in which case its own missing fields are reported, each nameable from
// the same literal — or absent altogether and reported as one missing field.
func (s *Struct) skippedIn(fs *Fields, keys map[string]bool, callerPkgPath string) []Field {
	externalPkg := fs.PackagePath != callerPkgPath
	promoted := fs.promotedKeys(keys)

	// Capacity is a hint, not a contract: keys may name fields absent from
	// Items, since unexported fields are pruned for types whose fields come from
	// another package. Clamp so a mismatch can never turn into a makeslice panic.
	missing := make([]Field, 0, max(len(fs.Items)-len(keys), 0))

	for i, f := range fs.Items {
		if keys[f.Name] {
			continue
		}

		// A blank field has no name to write, so a keyed literal can never
		// initialize it and reporting it yields a finding nobody can act on.
		// Positional literals do supply a value for it, which skippedPositional
		// accounts for by keeping blank fields in Items.
		if f.Name == BlankFieldName {
			continue
		}

		if !s.isFieldRequired(f, externalPkg) {
			continue
		}

		if len(promoted[i]) > 0 {
			missing = append(missing, s.skippedIn(f.Embedded, promoted[i], callerPkgPath)...)

			continue
		}

		missing = append(missing, f)
	}

	return missing
}

// promotedKeys groups the keys that reach a field of fs through an embedded one,
// by the index of that embedded field. A key naming a field of fs directly
// belongs to no group: Go promotion resolves a name to its shallowest
// occurrence, so a direct field shadows an embedded one of the same name.
//
// Nothing is grouped for a struct that embeds nothing, which is the common case
// and costs one pass over the fields.
func (fs *Fields) promotedKeys(keys map[string]bool) map[int]map[string]bool {
	owners := fs.owners
	if owners == nil {
		// Fields built outside the processor carry no precomputed map.
		owners = fs.computeOwners()
	}

	// Empty means nothing is promoted, so every key names a field directly.
	if len(owners) == 0 {
		return nil
	}

	var promoted map[int]map[string]bool

	for key := range keys {
		owner, ok := owners[key]
		if !ok || fs.Items[owner].Name == key {
			continue
		}

		if promoted == nil {
			promoted = make(map[int]map[string]bool, len(fs.Items))
		}

		if promoted[owner] == nil {
			promoted[owner] = make(map[string]bool, len(keys))
		}

		promoted[owner][key] = true
	}

	return promoted
}

// embeddedLevel is one embedded struct reached while widening the promotion
// walk, together with the field of the outermost struct that owns it.
type embeddedLevel struct {
	fields *Fields
	owner  int
}

// computeOwners maps every name a literal of fs may write to the index of the
// field that initializes it, and is empty when fs embeds nothing.
//
// The empty result is a value rather than nil so that a type embedding nothing
// answers every literal without walking its fields again.
func (fs *Fields) computeOwners() map[string]int {
	next := fs.embeddedLevels(outermost)
	if next == nil {
		return map[string]int{}
	}

	owners := make(map[string]int, len(fs.Items))

	for i, f := range fs.Items {
		if _, seen := owners[f.Name]; !seen {
			owners[f.Name] = i
		}
	}

	// Widen one level at a time and keep the first index recorded for a name,
	// which is Go's shallowest-wins rule. Two names equally deep are ambiguous
	// and cannot appear in a literal at all, so either index answers.
	for len(next) > 0 {
		current := next

		next = nil

		for _, level := range current {
			for _, f := range level.fields.Items {
				if _, seen := owners[f.Name]; !seen {
					owners[f.Name] = level.owner
				}
			}

			next = append(next, level.fields.embeddedLevels(level.owner)...)
		}
	}

	return owners
}

// embeddedLevels lists the structs embedded directly in fs, each attributed to
// owner. At the outermost level a field owns itself, which is the index the
// walk carries down.
func (fs *Fields) embeddedLevels(owner int) []embeddedLevel {
	var levels []embeddedLevel

	for i, f := range fs.Items {
		if f.Embedded == nil {
			continue
		}

		attributed := owner
		if owner == outermost {
			attributed = i
		}

		levels = append(levels, embeddedLevel{fields: f.Embedded, owner: attributed})
	}

	return levels
}

func (s *Struct) isFieldRequired(f Field, externalPkg bool) bool {
	// explicit field directives win over everything
	if f.Enforced {
		return true
	}

	if f.Optional {
		return false
	}

	// field-level patterns apply only when they specifically target the field —
	// i.e., the pattern matches the field path but not the struct path. A broad
	// pattern that also matches the struct is handled via s.IsEnforced/IsOptional
	// and must not silently promote unrelated fields to required/optional.
	if f.PatternEnforced && !s.PatternEnforced {
		return true
	}

	if f.PatternOptional && !s.PatternOptional {
		return false
	}

	// optionality can be inherited from the structure settings
	if s.IsOptional() {
		return false
	}

	// unexported fields are only required for same-package usage
	if externalPkg && !f.Exported {
		return false
	}

	return true
}

type Field struct {
	Name     string
	Exported bool //exhaustruct:optional
	Enforced bool //exhaustruct:optional
	Optional bool //exhaustruct:optional

	PatternEnforced bool //exhaustruct:optional
	PatternOptional bool //exhaustruct:optional

	// Embedded holds the fields promoted by an embedded struct field, which a
	// literal may name in place of this field since Go 1.27. Nil for every
	// other field, including embedded pointers — a literal cannot reach through
	// those.
	Embedded *Fields //exhaustruct:optional
}

// Fields is a collection of struct fields with shared package metadata.
// Items are in declaration order (required for positional literals).
type Fields struct {
	PackagePath string
	Items       []Field

	// owners is the promotion map of Items, computed once when the type's
	// metadata is built and shared by every literal of the type. Nil until then,
	// and nil for good when nothing is embedded.
	owners map[string]int //exhaustruct:optional
}

func FormatFieldNames(fields []Field) string {
	switch len(fields) {
	case 0:
		return ""
	case 1:
		return fields[0].Name
	}

	var b strings.Builder

	b.Grow(len(fields))
	b.WriteString(fields[0].Name)

	for _, s := range fields[1:] {
		b.WriteString(", ")
		b.WriteString(s.Name)
	}

	return b.String()
}
