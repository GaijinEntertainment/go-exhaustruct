// Package report_full_path tests ReportFullTypePath=true behavior.
// Error messages include the full package path instead of just the package name.
package report_full_path

// Test is a simple struct for testing.
type Test struct {
	A string
	B int
}

func shouldFailEmpty() {
	_ = Test{} // want "testdata/config/report_full_path.Test is missing fields A, B"
}

func shouldFailPartial() {
	_ = Test{A: ""} // want "testdata/config/report_full_path.Test is missing field B"
}
