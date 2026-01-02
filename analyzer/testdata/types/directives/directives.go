// Package directives tests comment directive behavior.
// Directives: //exhaustruct:ignore, //exhaustruct:enforce, //exhaustruct:optional
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

// TestWithOptionalFields has some fields marked as optional via comment directives.
type TestWithOptionalFields struct {
	Required string
	//exhaustruct:optional
	OptionalDoc    string
	OptionalInline int //exhaustruct:optional
	AlsoRequired   bool
}

// TestMixedOptional has both tag-based and comment-based optional fields.
type TestMixedOptional struct {
	Required string
	//exhaustruct:optional
	OptionalByDirective string
	OptionalByTag       string `exhaustruct:"optional"`
	AlsoRequired        int
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

// === Tests for field optionality via comment directives ===

func shouldPassOptionalFieldsViaDirective() {
	// Only required fields are present - optional fields are skipped
	_ = TestWithOptionalFields{
		Required:     "value",
		AlsoRequired: true,
	}

	// All fields present is also valid
	_ = TestWithOptionalFields{
		Required:       "value",
		OptionalDoc:    "doc",
		OptionalInline: 42,
		AlsoRequired:   true,
	}
}

func shouldFailMissingRequiredWithOptionalDirectives() {
	// Missing Required field
	_ = TestWithOptionalFields{ // want "directives.TestWithOptionalFields is missing fields Required, AlsoRequired"
		OptionalDoc:    "doc",
		OptionalInline: 42,
	}

	// Missing AlsoRequired field
	_ = TestWithOptionalFields{ // want "directives.TestWithOptionalFields is missing field AlsoRequired"
		Required: "value",
	}
}

func shouldPassMixedOptional() {
	// Both tag-based and directive-based optional fields work
	_ = TestMixedOptional{
		Required:     "value",
		AlsoRequired: 42,
	}

	// Can also provide optional fields
	_ = TestMixedOptional{
		Required:            "value",
		OptionalByDirective: "directive",
		OptionalByTag:       "tag",
		AlsoRequired:        42,
	}
}

func shouldFailMixedOptionalMissingRequired() {
	_ = TestMixedOptional{ // want "directives.TestMixedOptional is missing field Required"
		AlsoRequired: 42,
	}
}

// === Tests for external types with optional directive ===

func shouldPassExternalOptionalDirective() {
	// External type with optional directive should work
	_ = external.WithOptionalDirective{
		Required: "value",
	}

	// Can also provide optional fields
	_ = external.WithOptionalDirective{
		Required:            "value",
		OptionalByDirective: "directive",
		OptionalByTag:       "tag",
	}
}

func shouldFailExternalMissingRequired() {
	_ = external.WithOptionalDirective{} // want "external.WithOptionalDirective is missing field Required"
}
