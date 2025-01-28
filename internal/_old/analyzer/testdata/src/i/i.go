//nolint:all
package i

import (
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

func shouldPassPointer() {
	_ = &Test{
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
	_ = Test{ // want "i.Test is missing field D"
		A: "",
		B: 0,
		C: 0.0,
	}
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
	_ = Test2{ // want "i.Test2 is missing field Embedded"
		External: e.External{ // want "e.External is missing field B"
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
	_ = testGenericStruct[int]{} // want "i.testGenericStruct is missing fields A, B"
	_ = testGenericStruct[int]{  // want "i.testGenericStruct is missing field B"
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
		{},            // want "i.Test3 is missing field A"
		Test3{B: 123}, // want "i.Test3 is missing field A"
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
		"a": {},            // want "i.Test3 is missing field A"
		"b": Test3{B: 123}, // want "i.Test3 is missing field A"
	}
}

func shouldPassSlice() {
	_ = []string{"a", "b"}
}

func shouldPassAnonymousStruct() {
	_ = struct {
		A string
		B int
	}{
		A: "a",
		B: 1,
	}
}

func shouldFailAnonymousStructUnfilled() {
	_ = struct { // want "i.<anonymous> is missing field A"
		A string
		B int
	}{
		B: 1,
	}
}

type TestAlias Test
type TestAliasAlias TestAlias
type TestExcludedAlias TestExcluded

func shouldFailTypeAliases() {
	_ = TestAlias{}         // want "i.TestAlias is missing fields A, B, C, D"
	_ = TestAliasAlias{}    // want "i.TestAliasAlias is missing fields A, B, C, D"
	_ = TestExcludedAlias{} // want "i.TestExcludedAlias is missing fields A, B"
}

type TestAliasExcluded TestAlias

func shouldSucceedExcludedAliases() {
	_ = TestAliasExcluded{}
}

type TestExternalAlias e.External
type TestExternalAliasAlias TestExternalAlias
type TestExternalExcludedAlias e.ExternalExcluded

func shouldFailExternalTypeAliases() {
	_ = TestExternalAlias{}         // want "i.TestExternalAlias is missing fields A, B"
	_ = TestExternalAliasAlias{}    // want "i.TestExternalAliasAlias is missing fields A, B"
	_ = TestExternalExcludedAlias{} // want "i.TestExternalExcludedAlias is missing fields A, B"
}
