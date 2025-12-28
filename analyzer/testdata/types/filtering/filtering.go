// Package filtering tests include/exclude pattern behavior.
package filtering

// TestExcluded matches exclusion pattern (.*Excluded.*).
type TestExcluded struct {
	A string
	B int
}

// NotIncluded does not match include pattern (.*\.Test.*).
type NotIncluded struct {
	A string
	B int
}

func shouldPassExcluded() {
	_ = TestExcluded{}
}

func shouldPassNotIncluded() {
	_ = NotIncluded{}
}
