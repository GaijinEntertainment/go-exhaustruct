package empty_global

type TestStruct struct {
	A string
	B int
}

type NestedStruct struct {
	Inner TestStruct
	Value string
}

func shouldPassEmptyStruct() {
	// Should pass because AllowEmpty is true
	_ = TestStruct{}
}

func shouldPassNestedEmptyStruct() {
	// Should pass because AllowEmpty is true
	_ = NestedStruct{}
}

func shouldPassEmptyStructInSlice() {
	// Should pass because AllowEmpty is true
	_ = []TestStruct{{}}
}

func shouldPassEmptyStructInReturn() TestStruct {
	// Should pass because AllowEmpty is true
	return TestStruct{}
}

func shouldPassEmptyStructInDeclaration() {
	// Should pass because AllowEmpty is true
	var test TestStruct
	test = TestStruct{}
	_ = test
}

func shouldPassPointerToEmptyStruct() {
	// Should pass because AllowEmpty is true
	_ = &TestStruct{}
}