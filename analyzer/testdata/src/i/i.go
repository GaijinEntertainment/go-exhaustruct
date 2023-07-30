//nolint:all
package i

import (
	"errors"

	"e"
)

// Embedded is a test struct that is subject to enforcement by inclusion flags
type Embedded struct {
	E string
	F string
	g string
	H string
}

// Test is a test struct that is subject to enforcement by inclusion flags but
// contains an optional field
type Test struct {
	A string
	B int
	C float32
	D bool
	E string `exhaustruct:"optional"`
}

// Test2 is a test struct that is subject to enforcement by inclusion flags
type Test2 struct {
	Embedded
	External e.External
}

// The struct literal is fully filled out and should pass
func shouldPassFullyDefined() {
	_ = Test{
		A: "",
		B: 0,
		C: 0.0,
		D: false,
		E: "",
	}
}

// The struct pointer literal is fully filled out and should pass
func shouldPassPointer() {
	_ = &Test{
		A: "",
		B: 0,
		C: 0.0,
		D: false,
		E: "",
	}
}

// The struct pointer literal is fully filled out aside from optional fields and should pass
func shouldPassOnlyOptionalOmitted() {
	_ = Test{
		A: "",
		B: 0,
		C: 0.0,
		D: false,
	}
}

// The struct pointer literal is missing non-optional fields and should fail
func shouldFailRequiredOmitted() {
	_ = Test{ // want "i.Test is missing field D"
		A: "",
		B: 0,
		C: 0.0,
	}
}

// Returning an empty struct literal with a non-nil error should pass
func shouldPassEmptyStructWithNonNilErr() (Test, error) {
	return Test{}, errors.New("some error")
}

// Returning an empty struct literal with a nil error should fail
func shouldFailEmptyStructWithNilErr() (Test, error) {
	return Test{}, nil // want "i.Test is missing fields A, B, C, D"
}

// Returning an slice of empty struct literals with a nil error should fail
func shouldFailEmptyNestedStructWithNonNilErr() ([]Test, error) {
	return []Test{{}}, nil // want "i.Test is missing fields A, B, C, D"
}

// The struct is fully filled out using a list assignment and should pass
func shouldPassUnnamed() {
	_ = []Test{{"", 0, 0.0, false, ""}}
}

// The struct and its inner structs are fully filled out and should pass
func shouldPassEmbedded() {
	_ = Test2{
		External: e.External{
			A: "",
			B: "",
		},
		Embedded: Embedded{
			E: "",
			F: "",
			H: "",
			g: "",
		},
	}
}

// The embedded inner struct is missing a field and should fail
func shouldFailEmbedded() {
	_ = Test2{
		External: e.External{
			A: "",
			B: "",
		},
		Embedded: Embedded{ // want "Embedded is missing field g"
			E: "",
			F: "",
			H: "",
		},
	}
}

// The embedded inner struct is not specified and should fail
func shouldFailEmbeddedCompletelyMissing() {
	_ = Test2{ // want "i.Test2 is missing field Embedded"
		External: e.External{ // want "e.External is missing field B"
			A: "",
		},
	}
}

// Struct with type parameters
type testGenericStruct[T any] struct {
	A T
	B string
}

// The type-parameterized struct is fully filled out and should pass
func shouldPassGeneric() {
	_ = testGenericStruct[int]{
		A: 42,
		B: "the answer",
	}
}

// The type-parameterized struct is missing a field and should fail
func shouldFailGeneric() {
	_ = testGenericStruct[int]{} // want "i.testGenericStruct is missing fields A, B"
	_ = testGenericStruct[int]{  // want "i.testGenericStruct is missing field B"
		A: 42,
	}
}

// TestExcluded is a test struct that is subject to exclusion by exclusion flags
type TestExcluded struct {
	A string
	B int
}

// The struct is excluded and should pass
func shouldPassExcluded() {
	_ = TestExcluded{}
}

// NotIncluded is a test struct that is not included by inclusion flags
type NotIncluded struct {
	A string
	B int
}

// The struct is not excluded and should pass
func shouldPassNotIncluded() {
	_ = NotIncluded{}
}

// Test3 is a test struct that is subject to enforcement by inclusion flags and has an optional field
type Test3 struct {
	A string
	B int `exhaustruct:"optional"`
}

// All structs in the slice are fully filled out and should pass
func shouldPassSlicesOfStructs() {
	_ = []Test3{
		{"a", 1},
		{A: "a"},
		Test3{A: "b"},
	}
}

// All structs in the slice are missing some fields and should fail
func shouldFailSlicesOfStructs() {
	_ = []Test3{
		{},            // want "i.Test3 is missing field A"
		Test3{B: 123}, // want "i.Test3 is missing field A"
	}
}

// All structs in the map are fully filled out and should pass
func shouldPassMapOfStructs() {
	_ = map[string]Test3{
		"a": {"a", 1},
		"b": {A: "a"},
		"c": Test3{A: "b"},
	}
}

// All structs in the map are missing some fields and should fail
func shouldFailMapOfStructs() {
	_ = map[string]Test3{
		"a": {},            // want "i.Test3 is missing field A"
		"b": Test3{B: 123}, // want "i.Test3 is missing field A"
	}
}

// Slices of strings are not subject to enforcement and should pass
func shouldPassSlice() {
	_ = []string{"a", "b"}
}

// All anonymous structs are fully filled out and should pass
func shouldPassAnonymousStruct() {
	_ = struct {
		A string
		B int
	}{
		A: "a",
		B: 1,
	}
}

// All anonymous structs are subject to enforcement and missing some fields and should fail
func shouldFailAnonymousStructUnfilled() {
	_ = struct { // want "i.<anonymous> is missing field A"
		A string
		B int
	}{
		B: 1,
	}
}
