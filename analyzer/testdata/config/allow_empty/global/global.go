// Package global tests AllowEmpty=true behavior.
// All empty struct literals are allowed when this option is enabled.
package global

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

func shouldPassEmpty() {
	_ = TestStruct{}
	_ = NestedStruct{}
}

func shouldPassEmptyInSlice() {
	_ = []TestStruct{{}}
}

func shouldPassEmptyInReturn() TestStruct {
	return TestStruct{}
}

func shouldPassEmptyInDeclaration() {
	test := TestStruct{}
	_ = test
}

func shouldPassPointerToEmpty() {
	_ = &TestStruct{}
}
