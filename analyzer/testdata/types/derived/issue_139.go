package derived

import (
	"testdata/external"
)

// Issue #139: External derived types should not require unexported fields.
// https://github.com/GaijinEntertainment/go-exhaustruct/issues/139
//
// When deriving from an external type, unexported fields from the original
// struct are inaccessible and should not be required.

// External derived types.

type ExternalDerived external.Simple
type ExternalDerivedDerived ExternalDerived
type ExternalExcludedDerived external.Excluded
type ExternalEmptyDerived external.Empty
type ExternalOnlyUnexportedDerived external.OnlyUnexported

func shouldFailExternalDerived() {
	// Only exported fields A, B should be required (not unexported d)
	_ = ExternalDerived{}        // want "derived.ExternalDerived is missing fields A, B"
	_ = ExternalDerivedDerived{} // want "derived.ExternalDerivedDerived is missing fields A, B"
}

func shouldPassExternalExcludedDerived() {
	_ = ExternalExcludedDerived{}
}

func shouldPassExternalDerivedComplete() {
	_ = ExternalDerived{A: "", B: 0}
	_ = ExternalDerivedDerived{A: "", B: 0}
}

func shouldPassExternalEmptyDerived() {
	// Empty struct has no fields to initialize
	_ = ExternalEmptyDerived{}
}

func shouldPassExternalOnlyUnexportedDerived() {
	// All fields are unexported and inaccessible, so nothing is required
	_ = ExternalOnlyUnexportedDerived{}
}
