// Package aliases tests type alias behavior.
// Type aliases are created with `type T = U` (with `=`).
// Each alias is treated as its own type for error messages and directives.
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
type AliasAlias = Alias // alias of alias
type AliasDerived Alias // derived from alias (creates new type)

func shouldFailLocalAliases() {
	_ = Alias{}        // want "aliases.Alias is missing fields A, B, C, D"
	_ = AliasAlias{}   // want "aliases.AliasAlias is missing fields A, B, C, D"
	_ = AliasDerived{} // want "aliases.AliasDerived is missing fields A, B, C, D"
}

// External type aliases.
type ExternalAlias = external.Simple
type ExternalAliasAlias = ExternalAlias
type ExternalExcludedAlias = external.Excluded

func shouldFailExternalAliases() {
	// Unexported field c is ignored for external types
	_ = ExternalAlias{}      // want "aliases.ExternalAlias is missing fields A, B"
	_ = ExternalAliasAlias{} // want "aliases.ExternalAliasAlias is missing fields A, B"
}

func shouldPassExcludedAlias() {
	_ = ExternalExcludedAlias{}
}

// --- Directive behavior tests ---
// Directives only apply to the type where they're defined.
// No inheritance from base types to aliases or derived types.

// IgnoredBase has ignore directive on the base type.
//
//exhaustruct:ignore
type IgnoredBase struct {
	X string
	Y int
}

// IgnoredBaseAlias is an alias to IgnoredBase.
// Directive on base type does NOT propagate to alias.
type IgnoredBaseAlias = IgnoredBase

func testDirectiveOnBase() {
	// Base type is ignored - passes
	_ = IgnoredBase{}
	// Alias to ignored base - fails (directive NOT inherited)
	_ = IgnoredBaseAlias{} // want "aliases.IgnoredBaseAlias is missing fields X, Y"
}

// AliasTarget is a normal struct (not ignored).
type AliasTarget struct {
	X string
	Y int
}

// IgnoredAlias has ignore directive on the alias itself.
//
//exhaustruct:ignore
type IgnoredAlias = AliasTarget

func testDirectiveOnAlias() {
	// Base type NOT ignored - fails
	_ = AliasTarget{} // want "aliases.AliasTarget is missing fields X, Y"
	// Alias has ignore directive - passes
	_ = IgnoredAlias{}
}
