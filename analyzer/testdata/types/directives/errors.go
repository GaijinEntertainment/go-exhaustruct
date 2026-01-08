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
