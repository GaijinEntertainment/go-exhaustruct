//nolint:all
package i

import (
	"errors"

	"e"
)

type Embedded struct {
	E string
	F string
	g string
	H string
}

type Test struct {
	A string
	B int
	C float32
	D bool
	E string `exhaustruct:"optional"`
}

type Test2 struct {
	Embedded
	External e.External
}

func shouldPassFullyDefined() {
	_ = Test{
		A: "",
		B: 0,
		C: 0.0,
		D: false,
		E: "",
	}
}

func shouldPassOnlyOptionalOmitted() {
	_ = Test{
		A: "",
		B: 0,
		C: 0.0,
		D: false,
	}
}

func shouldFailRequiredOmitted() {
	_ = Test{ // want "Test is missing field D"
		A: "",
		B: 0,
		C: 0.0,
	}
}

func shouldPassEmptyStructWithNonNilErr() (Test, error) {
	return Test{}, errors.New("some error")
}

func shouldFailEmptyStructWithNilErr() (Test, error) {
	return Test{}, nil // want "Test is missing fields A, B, C, D"
}

func shouldFailEmptyNestedStructWithNonNilErr() ([]Test, error) {
	return []Test{{}}, nil // want "Test is missing fields A, B, C, D"
}

func shouldPassUnnamed() {
	_ = []Test{{"", 0, 0.0, false, ""}}
}

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

func shouldFailEmbeddedCompletelyMissing() {
	_ = Test2{ // want "Test2 is missing field Embedded"
		External: e.External{ // want "External is missing field B"
			A: "",
		},
	}
}

type testGenericStruct[T any] struct {
	A T
	B string
}

func shouldPassGeneric() {
	_ = testGenericStruct[int]{
		A: 42,
		B: "the answer",
	}
}

func shouldFailGeneric() {
	_ = testGenericStruct[int]{} // want "testGenericStruct is missing fields A, B"
	_ = testGenericStruct[int]{  // want "testGenericStruct is missing field B"
		A: 42,
	}
}

type TestExcluded struct {
	A string
	B int
}

func shouldPassExcluded() {
	_ = TestExcluded{}
}

type NotIncluded struct {
	A string
	B int
}

func shouldPassNotIncluded() {
	_ = NotIncluded{}
}

type Test3 struct {
	A string
	B int `exhaustruct:"optional"`
}

func shouldPassSlicesOfStructs() {
	_ = []Test3{
		{"a", 1},
		{A: "a"},
		Test3{A: "b"},
	}
}

func shouldFailSlicesOfStructs() {
	_ = []Test3{
		{},            // want "Test3 is missing field A"
		Test3{B: 123}, // want "Test3 is missing field A"
	}
}

func shouldPassMapOfStructs() {
	_ = map[string]Test3{
		"a": {"a", 1},
		"b": {A: "a"},
		"c": Test3{A: "b"},
	}
}

func shouldFailMapOfStructs() {
	_ = map[string]Test3{
		"a": {},            // want "Test3 is missing field A"
		"b": Test3{B: 123}, // want "Test3 is missing field A"
	}
}
