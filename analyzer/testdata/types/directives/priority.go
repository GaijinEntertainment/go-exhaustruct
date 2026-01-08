// Package directives tests directive priority chain.
//
// Priority (highest to lowest):
//  1. literal:ignore  - always skips checking
//  2. literal:enforce - forces checking (if not ignored)
//  3. struct:ignore   - skips checking for all literals of this type
//  4. struct:enforce  - forces checking for all literals of this type
//  5. mode default    - implicit mode checks, explicit mode skips
//
// This file tests combinations to verify priority order.
package directives

// === Type-level directives for priority tests ===

// PriorityIgnored has type-level ignore.
//
//exhaustruct:ignore
type PriorityIgnored struct {
	A string
	B int
}

// PriorityEnforced has type-level enforce.
//
//exhaustruct:enforce
type PriorityEnforced struct {
	A string
	B int
}

// PriorityPlain has no type-level directive.
type PriorityPlain struct {
	A string
	B int
}

// === Priority 1: literal:ignore beats everything ===

func shouldPassLiteralIgnoreBeatsStructEnforce() {
	// literal:ignore > struct:enforce
	//exhaustruct:ignore
	_ = PriorityEnforced{}
}

func shouldPassLiteralIgnoreOnPlain() {
	// literal:ignore > implicit mode default
	//exhaustruct:ignore
	_ = PriorityPlain{}
}

// === Priority 2: literal:enforce beats struct-level ===

func shouldFailLiteralEnforceBeatsStructIgnore() {
	// literal:enforce > struct:ignore
	//exhaustruct:enforce
	_ = PriorityIgnored{} // want "directives.PriorityIgnored is missing fields A, B"
}

func shouldFailLiteralEnforceOnPlain() {
	// literal:enforce > implicit mode default (redundant but valid)
	//exhaustruct:enforce
	_ = PriorityPlain{} // want "directives.PriorityPlain is missing fields A, B"
}

// === Priority 3: struct:ignore beats struct:enforce ===
// Note: Can't have both on same type, so this tests struct:ignore works

func shouldPassStructIgnore() {
	_ = PriorityIgnored{}
}

// === Priority 4: struct:enforce beats mode default ===

func shouldFailStructEnforce() {
	_ = PriorityEnforced{} // want "directives.PriorityEnforced is missing fields A, B"
}

// === Priority 5: mode default (implicit mode = check) ===

func shouldFailModeDefault() {
	_ = PriorityPlain{} // want "directives.PriorityPlain is missing fields A, B"
}

// === Combined literal directives ===
// When both ignore and enforce are on same literal, ignore wins

func shouldPassCombinedIgnoreEnforce() {
	//exhaustruct:ignore,enforce
	_ = PriorityPlain{}
}

func shouldPassCombinedEnforceIgnore() {
	//exhaustruct:enforce,ignore
	_ = PriorityPlain{}
}

// === Multiple literals with different directives ===

func shouldHandleMixedLiteralDirectives() {
	// First literal: enforced
	//exhaustruct:enforce
	_ = PriorityIgnored{} // want "directives.PriorityIgnored is missing fields A, B"

	// Second literal: uses type-level ignore
	_ = PriorityIgnored{}

	// Third literal: explicit enforce again
	//exhaustruct:enforce
	_ = PriorityIgnored{} // want "directives.PriorityIgnored is missing fields A, B"
}

// === Inline vs block directive position ===

func shouldHandleInlineDirective() {
	_ = PriorityIgnored{} //exhaustruct:enforce // want "directives.PriorityIgnored is missing fields A, B"
}

func shouldHandleBlockDirective() {
	//exhaustruct:enforce
	_ = PriorityIgnored{} // want "directives.PriorityIgnored is missing fields A, B"
}
