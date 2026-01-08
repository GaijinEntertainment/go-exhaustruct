// Package explicit tests explicit mode behavior.
// In explicit mode: only types matching EnforcePatterns are checked by default.
// Directives can override this behavior.
package explicit

// Enforced matches EnforcePatterns (.*Enforced.*).
type Enforced struct {
	A string
	B int
}

// Skipped does NOT match EnforcePatterns.
type Skipped struct {
	A string
	B int
}

// EnforcedViaDirective does NOT match patterns but has enforce directive.
//
//exhaustruct:enforce
type EnforcedViaDirective struct {
	A string
	B int
}

// IgnoredViaDirective matches patterns but has ignore directive.
//
//exhaustruct:ignore
type IgnoredViaDirective struct {
	A string
	B int
}

// === Default explicit mode behavior ===

func shouldFailEnforcedEmpty() {
	_ = Enforced{} // want "explicit.Enforced is missing fields A, B"
}

func shouldPassSkippedEmpty() {
	// Type doesn't match patterns, skipped in explicit mode
	_ = Skipped{}
}

// === Type-level directive overrides ===

func shouldFailEnforcedViaDirective() {
	// Type has //exhaustruct:enforce, checked despite not matching patterns
	_ = EnforcedViaDirective{} // want "explicit.EnforcedViaDirective is missing fields A, B"
}

func shouldPassIgnoredViaDirective() {
	// Type has //exhaustruct:ignore, skipped despite matching patterns
	_ = IgnoredViaDirective{}
}

// === Literal-level directive override ===

func shouldPassEnforcedWithLiteralIgnore() {
	// Type matches patterns, but literal has ignore directive
	//exhaustruct:ignore
	_ = Enforced{}
}

// === Anonymous structs in explicit mode ===

func shouldPassAnonymousInExplicitMode() {
	// Anonymous structs don't match patterns, skipped in explicit mode
	_ = struct {
		A string
		B int
	}{}
}
