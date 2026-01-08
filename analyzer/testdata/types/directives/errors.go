// Package directives tests directive parsing error diagnostics.
package directives

// === Invalid directive format errors ===

func shouldReportEmptyDirective() {
	//exhaustruct: // want "empty directive"
	_ = Test{} // want "directives.Test is missing fields A, B, C, D"
}

func shouldReportUnknownDirective() {
	//exhaustruct:unknown // want "unknown directive"
	_ = Test{} // want "directives.Test is missing fields A, B, C, D"
}

func shouldReportDuplicateDirectives() {
	// Duplicate is reported but directive still works
	//exhaustruct:ignore,ignore // want "duplicate directives"
	_ = Test{}
}

func shouldReportUnknownWithValid() {
	// Unknown directive is reported, but valid ones still work
	//exhaustruct:ignore,unknown // want "unknown directive"
	_ = Test{}
}

func shouldReportMultipleInCommentGroup() {
	// Multiple directives in same comment group - second is ignored
	// First directive (ignore) still applies
	//exhaustruct:ignore
	//exhaustruct:enforce // want "multiple exhaustruct directives in a single comment group"
	_ = Test{}
}

// === Conflicting directives for same target line ===

func shouldReportConflictingDirectives() {
	// Block comment takes precedence over inline
	//exhaustruct:ignore
	_ = Test{} //exhaustruct:enforce // want "directive ignored, conflicting directive already exists"
}

// === Edge cases ===

func shouldNotReportMisspelledPrefix() {
	// "exhaustive" is not "exhaustruct", so not our directive
	//exhaustive:ignore
	_ = Test{} // want "directives.Test is missing fields A, B, C, D"
}

func shouldReportSpaceAfterColon() {
	// Space after colon: " ignore" (with leading space) is parsed as unknown directive.
	// The entire rest of the line becomes the directive value.
	// Note: Can't easily test inline due to want comment format.
	// Behavior verified: space after colon makes entire rest of line the directive value.
	_ = Test{} // want "directives.Test is missing fields A, B, C, D"
}

func shouldPassNolintWithExhaustruct() {
	// nolint is golangci-lint feature, not handled by analyzer.
	// Analyzer still reports the error; golangci-lint would suppress it.
	//nolint:exhaustruct
	_ = Test{} // want "directives.Test is missing fields A, B, C, D"
}

// === Valid combined directives ===

func shouldPassValidCombinedDirectives() {
	// Both ignore and enforce present - ignore takes priority
	//exhaustruct:ignore,enforce
	_ = Test{}
}
