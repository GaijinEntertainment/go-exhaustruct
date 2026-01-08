// Package explicit tests explicit mode with directive interactions.
// In explicit mode: only types matching EnforcePatterns are checked by default.
// Directives can override this behavior:
// - //exhaustruct:enforce forces checking even when type not in patterns
// - //exhaustruct:ignore skips checking even when type is in patterns
package explicit

// Enforced matches EnforcePatterns (.*Enforced.*).
// Will be checked by default in explicit mode.
type Enforced struct {
	A string
	B int
}

// Skipped does NOT match EnforcePatterns.
// Will be skipped by default in explicit mode.
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

func shouldPassEnforcedComplete() {
	_ = Enforced{A: "", B: 0}
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

func shouldPassEnforcedViaDirectiveComplete() {
	_ = EnforcedViaDirective{A: "", B: 0}
}

func shouldPassIgnoredViaDirective() {
	// Type has //exhaustruct:ignore, skipped despite matching patterns
	_ = IgnoredViaDirective{}
}

// === Literal-level directive overrides ===

func shouldPassEnforcedWithLiteralIgnore() {
	// Type matches patterns, but literal has ignore directive
	//exhaustruct:ignore
	_ = Enforced{}

	_ = Enforced{} //exhaustruct:ignore
}

func shouldFailSkippedWithLiteralEnforce() {
	// Type doesn't match patterns, but literal has enforce directive
	//exhaustruct:enforce
	_ = Skipped{} // want "explicit.Skipped is missing fields A, B"

	_ = Skipped{} //exhaustruct:enforce // want "explicit.Skipped is missing fields A, B"
}

// === Priority: literal directives override type directives ===

func shouldFailIgnoredTypeWithLiteralEnforce() {
	// Type has //exhaustruct:ignore, but literal has //exhaustruct:enforce
	//exhaustruct:enforce
	_ = IgnoredViaDirective{} // want "explicit.IgnoredViaDirective is missing fields A, B"
}

func shouldPassEnforcedTypeWithLiteralIgnore() {
	// Type has //exhaustruct:enforce, but literal has //exhaustruct:ignore
	//exhaustruct:ignore
	_ = EnforcedViaDirective{}
}

// === Collections with mixed directives ===

func shouldHandleCollectionDirectives() {
	_ = []Enforced{
		{},                  // want "explicit.Enforced is missing fields A, B"
		{}, //exhaustruct:ignore
		{A: "", B: 0},
	}

	_ = map[string]Skipped{
		"a": {},
		"b": {}, //exhaustruct:enforce // want "explicit.Skipped is missing fields A, B"
		"c": {},
	}
}

// === Anonymous structs in explicit mode ===

func shouldPassAnonymousInExplicitMode() {
	// Anonymous structs don't match patterns, skipped in explicit mode
	_ = struct {
		A string
		B int
	}{}
}

func shouldFailAnonymousWithEnforceDirective() {
	// Anonymous with enforce directive should be checked
	//exhaustruct:enforce
	_ = struct { // want "explicit.<anonymous> is missing fields A, B"
		A string
		B int
	}{}
}

func shouldPassAnonymousInSliceExplicitMode() {
	// Anonymous in slice, not matched by patterns
	_ = []struct {
		A string
	}{
		{},
		{A: ""},
	}
}

func shouldFailAnonymousInSliceWithEnforce() {
	_ = []struct {
		A string
	}{
		{},
		{}, //exhaustruct:enforce // want "explicit.<anonymous> is missing field A"
		{A: ""},
	}
}