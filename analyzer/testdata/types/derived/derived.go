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

// === Directive non-inheritance test ===
// Directives on base types do NOT propagate to derived types.

// BaseIgnored has ignore directive.
//
//exhaustruct:ignore
type BaseIgnored struct {
	X string
	Y int
}

// DerivedFromIgnored is derived from BaseIgnored.
type DerivedFromIgnored BaseIgnored

func shouldPassBaseIgnored() {
	_ = BaseIgnored{}
}

func shouldFailDerivedFromIgnored() {
	// Derived type does NOT inherit ignore directive
	_ = DerivedFromIgnored{} // want "derived.DerivedFromIgnored is missing fields X, Y"
}

// === Directive on derived type definition ===
// Directives CAN be placed on derived type definitions.

// DerivedTarget is a normal struct (not ignored).
type DerivedTarget struct {
	X string
	Y int
}

// IgnoredDerived has ignore directive on the derived type itself.
//
//exhaustruct:ignore
type IgnoredDerived DerivedTarget

func testDirectiveOnDerived() {
	// Base type NOT ignored - fails
	_ = DerivedTarget{} // want "derived.DerivedTarget is missing fields X, Y"
	// Derived type has ignore directive - passes
	_ = IgnoredDerived{}
}
