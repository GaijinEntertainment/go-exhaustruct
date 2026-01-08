// Package directives tests directive priority chain.
//
// Priority (highest to lowest):
//  1. literal:ignore  - always skips checking
//  2. literal:enforce - forces checking (if not ignored)
//  3. struct:ignore   - skips checking for all literals of this type
//  4. struct:enforce  - forces checking for all literals of this type
//  5. mode default    - implicit mode checks, explicit mode skips
package directives

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

// === Priority tests ===

func shouldPassLiteralIgnoreBeatsStructEnforce() {
	// literal:ignore > struct:enforce
	//exhaustruct:ignore
	_ = PriorityEnforced{}
}

func shouldFailLiteralEnforceBeatsStructIgnore() {
	// literal:enforce > struct:ignore
	//exhaustruct:enforce
	_ = PriorityIgnored{} // want "directives.PriorityIgnored is missing fields A, B"
}

func shouldPassStructIgnore() {
	_ = PriorityIgnored{}
}

func shouldFailStructEnforce() {
	_ = PriorityEnforced{} // want "directives.PriorityEnforced is missing fields A, B"
}

func shouldPassCombinedIgnoreEnforce() {
	// When both ignore and enforce are on same literal, ignore wins
	//exhaustruct:ignore,enforce
	_ = PriorityEnforced{}
}

func shouldHandleInlineDirective() {
	_ = PriorityIgnored{} //exhaustruct:enforce // want "directives.PriorityIgnored is missing fields A, B"
}
