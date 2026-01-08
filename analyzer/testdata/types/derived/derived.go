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

// === Directive non-inheritance tests ===
// Directives on base types do NOT propagate to derived types.
// Each type definition has its own directive scope.

// BaseIgnored has ignore directive.
//
//exhaustruct:ignore
type BaseIgnored struct {
	X string
	Y int
}

// DerivedFromIgnored is derived from BaseIgnored.
// Should NOT inherit the ignore directive.
type DerivedFromIgnored BaseIgnored

// BaseEnforced has enforce directive.
//
//exhaustruct:enforce
type BaseEnforced struct {
	X string
	Y int
}

// DerivedFromEnforced is derived from BaseEnforced.
// Should NOT inherit the enforce directive.
type DerivedFromEnforced BaseEnforced

func shouldPassBaseIgnored() {
	// Base type has //exhaustruct:ignore, so it's skipped
	_ = BaseIgnored{}
}

func shouldFailDerivedFromIgnored() {
	// Derived type does NOT inherit ignore directive
	_ = DerivedFromIgnored{} // want "derived.DerivedFromIgnored is missing fields X, Y"
}

func shouldPassDerivedFromIgnoredWithLiteralIgnore() {
	// Literal-level ignore still works
	//exhaustruct:ignore
	_ = DerivedFromIgnored{}
}

func shouldFailBaseEnforced() {
	// Base type has //exhaustruct:enforce
	_ = BaseEnforced{} // want "derived.BaseEnforced is missing fields X, Y"
}

func shouldFailDerivedFromEnforced() {
	// Derived type is checked by default (implicit mode), NOT because of inherited enforce
	_ = DerivedFromEnforced{} // want "derived.DerivedFromEnforced is missing fields X, Y"
}

func shouldPassDerivedFromEnforcedWithLiteralIgnore() {
	// Literal-level ignore overrides default checking
	//exhaustruct:ignore
	_ = DerivedFromEnforced{}
}
