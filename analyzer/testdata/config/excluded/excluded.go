// Package excluded tests ExcludeRx pattern behavior.
// This package is excluded by pattern `testdata/config/excluded\.<anonymous>`.
package excluded

// TestExcluded matches exclusion pattern (.*Excluded$).
type TestExcluded struct {
	A string
	B string
}

func shouldPassExcludedType() {
	_ = TestExcluded{}
}

func shouldPassExcludedAnonymous() {
	_ = struct {
		A string
		B int
	}{}
}
