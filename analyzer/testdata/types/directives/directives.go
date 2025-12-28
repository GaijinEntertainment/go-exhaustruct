// Package directives tests comment directive behavior.
// Directives: //exhaustruct:ignore, //exhaustruct:enforce
package directives

import (
	"testdata/external"
)

// Test is a local struct for directive tests.
type Test struct {
	A string
	B int
	C float32
	D bool
}

// TestExcluded matches exclusion patterns.
type TestExcluded struct {
	A string
	B int
}

// Embedded is a local embedded type.
type Embedded struct {
	E string
	F string
	g string
	H string
}

// TestWithEmbedded has embedded and external fields.
type TestWithEmbedded struct {
	Embedded
	External external.Simple
}

func shouldPassIgnoreDirective() {
	//exhaustruct:ignore
	_ = Test{}

	_ = Test{} //exhaustruct:ignore

	//exhaustruct:ignore
	_ = TestWithEmbedded{}
}

func shouldFailWithoutDirective() {
	_ = Test{} // want "directives.Test is missing fields A, B, C, D"
}

func shouldPassIgnoreInCollections() {
	_ = []Test{
		{},                  // want "directives.Test is missing fields A, B, C, D"
		{}, //exhaustruct:ignore
		{},                  // want "directives.Test is missing fields A, B, C, D"
	}

	_ = map[string]Test{
		"a": {},                  // want "directives.Test is missing fields A, B, C, D"
		"b": {}, //exhaustruct:ignore
		"c": {},                  // want "directives.Test is missing fields A, B, C, D"
	}
}

func shouldFailEnforceOnExcluded() {
	//exhaustruct:enforce
	_ = TestExcluded{} // want "directives.TestExcluded is missing fields A, B"

	//exhaustruct:enforce
	_ = TestExcluded{B: 0} // want "directives.TestExcluded is missing field A"
}

func shouldPassMisspelledDirective() {
	// Misspelled directive is ignored
	//exhaustive:enforce
	_ = TestExcluded{}
}

func shouldHandleDirectivesOnFields() {
	_ = TestWithEmbedded{
		//exhaustruct:ignore
		External: external.Simple{},
		//exhaustruct:enforce
		Embedded: Embedded{}, // want "directives.Embedded is missing fields E, F, g, H"
	}

	_ = TestWithEmbedded{
		//exhaustruct:ignore
		External: external.Simple{},
		//exhaustruct:ignore
		Embedded: Embedded{},
	}
}
