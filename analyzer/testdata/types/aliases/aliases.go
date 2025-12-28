// Package aliases tests type alias behavior.
// Type aliases are created with `type T = U` (with `=`).
// Aliases resolve to the underlying type in error messages.
package aliases

import (
	"testdata/external"
)

// Base is the local base type for aliasing.
type Base struct {
	A string
	B int
	C float32
	D bool
}

// Local type aliases.
type Alias = Base
type AliasAlias = Alias   // alias of alias
type AliasDerived Alias   // derived from alias (creates new type)

func shouldFailLocalAliases() {
	_ = Alias{}        // want "aliases.Base is missing fields A, B, C, D"
	_ = AliasAlias{}   // want "aliases.Base is missing fields A, B, C, D"
	_ = AliasDerived{} // want "aliases.AliasDerived is missing fields A, B, C, D"
}

// External type aliases.
type ExternalAlias = external.Simple
type ExternalAliasAlias = ExternalAlias
type ExternalExcludedAlias = external.Excluded

func shouldFailExternalAliases() {
	// Unexported field c is ignored for external types
	_ = ExternalAlias{}      // want "external.Simple is missing fields A, B"
	_ = ExternalAliasAlias{} // want "external.Simple is missing fields A, B"
}

func shouldPassExcludedAlias() {
	_ = ExternalExcludedAlias{}
}
