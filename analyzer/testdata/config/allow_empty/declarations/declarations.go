// Package declarations tests AllowEmptyDeclarations=true behavior.
// Empty struct literals in variable declarations are allowed.
package declarations

// TestStruct is a simple struct.
type TestStruct struct {
	A string
	B int
}

// NestedStruct contains another struct.
type NestedStruct struct {
	Inner TestStruct
	Value string
}

func shouldPassVarDeclaration() {
	var test = TestStruct{}
	_ = test
}

func shouldPassShortDeclaration() {
	test := TestStruct{}
	_ = test
}

func shouldPassPointerDeclaration() {
	test := &TestStruct{}
	_ = test
}

func shouldPassNestedDeclaration() {
	nested := NestedStruct{}
	_ = nested
}

func shouldPassMultiValueDeclaration() {
	a, b := TestStruct{}, NestedStruct{}
	_, _ = a, b
}

func shouldFailInReturn() TestStruct {
	return TestStruct{} // want "declarations.TestStruct is missing fields A, B"
}

func shouldFailInSlice() {
	_ = []TestStruct{{}} // want "declarations.TestStruct is missing fields A, B"
}

func shouldFailMultiValueReturn() (TestStruct, NestedStruct) {
	return TestStruct{}, NestedStruct{} // want "declarations.TestStruct is missing fields A, B" "declarations.NestedStruct is missing fields Inner, Value"
}

func shouldFailAssignment() {
	var test TestStruct
	test = TestStruct{} // want "declarations.TestStruct is missing fields A, B"
	_ = test
}
