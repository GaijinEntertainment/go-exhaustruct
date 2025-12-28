// Package returns tests AllowEmptyReturns=true behavior.
// Empty struct literals in return statements are allowed.
package returns

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

func shouldPassEmptyInReturn() TestStruct {
	return TestStruct{}
}

func shouldPassEmptyInMultiReturn() (TestStruct, error) {
	return TestStruct{}, nil
}

func shouldPassPointerInReturn() *TestStruct {
	return &TestStruct{}
}

func shouldPassNestedInReturn() (NestedStruct, error) {
	return NestedStruct{}, nil
}

func shouldFailEmptyInDeclaration() {
	_ = TestStruct{} // want "returns.TestStruct is missing fields A, B"
}

func shouldFailEmptyInSlice() {
	_ = []TestStruct{{}} // want "returns.TestStruct is missing fields A, B"
}
