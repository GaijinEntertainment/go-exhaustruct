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

// Wrapper nests a literal inside another, so a directive written above the
// statement and one written above the nested literal reach that literal from
// two levels of the expression it sits in.
type Wrapper struct {
	Inner PriorityEnforced
	Tag   string
}

// IgnoredNested has type-level ignore, so only a directive reaching the nested
// literal can leave it checked.
//
//exhaustruct:ignore
type IgnoredNested struct {
	A string
	B int
}

type WrapperOfIgnored struct {
	Inner IgnoredNested
	Tag   string
}

func shouldPassStatementIgnoreBeatsNestedEnforce() {
	// Both reach the nested literal, and the ignore wins: the walk collects
	// every directive up to the statement rather than stopping at the first.
	//exhaustruct:ignore
	_ = Wrapper{
		//exhaustruct:enforce
		Inner: PriorityEnforced{},
		Tag:   "",
	}
}

func shouldFailStatementEnforceReachesNestedLiteral() {
	//exhaustruct:enforce
	_ = WrapperOfIgnored{
		Inner: IgnoredNested{}, // want "directives.IgnoredNested is missing fields A, B"
		Tag:   "",
	}
}

// A directive written beside the token a multiline literal closes on belongs to
// that literal, which the line it opened on identifies.
func shouldPassIgnoreBesideClosingBrace() {
	_ = PriorityEnforced{
		A: "",
	} //exhaustruct:ignore
}

// Constructs closing on one line are nested, and the directive beside the token
// belongs to the innermost of them -- not to the statement they all close with.
func shouldFailIgnoreBesideClosingBraceLeavesSibling() (PriorityEnforced, IgnoredNested) {
	return PriorityEnforced{ // want "directives.PriorityEnforced is missing field B"
		A: "",
	}, IgnoredNested{
		A: "",
	} //exhaustruct:ignore
}
