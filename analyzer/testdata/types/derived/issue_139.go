// Issue #139: External derived types incorrectly require unexported fields.
// https://github.com/GaijinEntertainment/go-exhaustruct/issues/139
//
// BUG: When deriving from an external type with unexported fields,
// the analyzer reports those fields as missing even though they're
// inaccessible from this package.
//
// Expected: Only exported fields (A, B) should be required.
// Actual: All fields including unexported (d) are required.
package derived

import (
	"testdata/external"
)

// External derived types.
type ExternalDerived external.Simple
type ExternalDerivedDerived ExternalDerived
type ExternalExcludedDerived external.Excluded

func shouldFailExternalDerived() {
	// BUG: Reports "A, B, d" but should report "A, B" only
	_ = ExternalDerived{}        // want "derived.ExternalDerived is missing fields A, B, d"
	_ = ExternalDerivedDerived{} // want "derived.ExternalDerivedDerived is missing fields A, B, d"
}

func shouldPassExternalExcludedDerived() {
	_ = ExternalExcludedDerived{}
}

// TODO: Uncomment after fixing issue #139
// func shouldPassExternalDerivedComplete() {
// 	_ = ExternalDerived{A: "", B: 0}
// 	_ = ExternalDerivedDerived{A: "", B: 0}
// }
