// Package pointers tests literals whose element type is a pointer reached
// through a name: an alias to a pointer, or a defined pointer type. A composite
// literal may elide `&T` for those exactly as it may for a plain *T.
//
// Such a pointer type is declared, so it names the literal and carries its own
// directives, as an alias to a struct does. A plain *T declares nothing and
// leaves the struct it points at to name the literal.
package pointers

// Base is the struct behind every pointer type here.
type Base struct {
	A int
	B int
}

// PtrAlias is an alias to a pointer type.
type PtrAlias = *Base

// PtrDefined is a defined pointer type.
type PtrDefined *Base

func shouldPassComplete() {
	_ = []*Base{{A: 1, B: 2}}
	_ = []PtrAlias{{A: 1, B: 2}}
	_ = []PtrDefined{{A: 1, B: 2}}
	_ = map[string]PtrAlias{"x": {A: 1, B: 2}}
	_ = [1]PtrAlias{{A: 1, B: 2}}
}

func shouldFailPlainPointerSlice() {
	_ = []*Base{{A: 1}} // want "pointers.Base is missing field B"
}

func shouldFailPointerAliasSlice() {
	_ = []PtrAlias{{A: 1}} // want "pointers.PtrAlias is missing field B"
}

func shouldFailPointerAliasMap() {
	_ = map[string]PtrAlias{"x": {A: 1}} // want "pointers.PtrAlias is missing field B"
}

func shouldFailPointerAliasArray() {
	_ = [1]PtrAlias{{A: 1}} // want "pointers.PtrAlias is missing field B"
}

func shouldFailDefinedPointerSlice() {
	_ = []PtrDefined{{A: 1}} // want "pointers.PtrDefined is missing field B"
}

// IgnoredPtrAlias and IgnoredPtrDefined carry a directive of their own, which
// Base neither supplies nor overrides.
//
//exhaustruct:ignore
type IgnoredPtrAlias = *Base

//exhaustruct:ignore
type IgnoredPtrDefined *Base

func shouldPassIgnoredPointerTypes() {
	_ = []IgnoredPtrAlias{{A: 1}}
	_ = []IgnoredPtrDefined{{A: 1}}
}
