// Package derived tests derived type behavior.
// Derived types are created with `type T U` (no `=`).
// They create new distinct types with the same underlying structure.
package derived

// Base is a local base type with unexported field.
type Base struct {
	A string
	B int
	c string // unexported
}

// BaseExcluded matches exclusion patterns.
type BaseExcluded struct {
	A string
	B int
}

// Derived types.
type Derived Base
type DerivedDerived Derived
type DerivedExcluded BaseExcluded

func shouldFailLocalDerived() {
	_ = Derived{}        // want "derived.Derived is missing fields A, B, c"
	_ = DerivedDerived{} // want "derived.DerivedDerived is missing fields A, B, c"
}

func shouldPassLocalDerivedComplete() {
	_ = Derived{A: "", B: 0, c: ""}
	_ = DerivedDerived{A: "", B: 0, c: ""}
}

func shouldPassExcludedDerived() {
	_ = DerivedExcluded{}
}
