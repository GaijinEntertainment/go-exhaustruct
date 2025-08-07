package empty_returns

type TestStruct struct {
	A string
	B int
}

type NestedStruct struct {
	Inner TestStruct
	Value string
}

func shouldPassEmptyStructInReturn() TestStruct {
	// Should pass because AllowEmptyReturns is true
	return TestStruct{}
}

func shouldPassEmptyStructInMultiReturn() (TestStruct, error) {
	// Should pass because AllowEmptyReturns is true
	return TestStruct{}, nil
}

func shouldPassPointerToEmptyStructInReturn() *TestStruct {
	// Should pass because AllowEmptyReturns is true
	return &TestStruct{}
}

func shouldFailEmptyStructInDeclaration() {
	// Should fail because this is not a return statement
	_ = TestStruct{} // want "empty_returns.TestStruct is missing fields A, B"
}

func shouldFailEmptyStructInSlice() {
	// Should fail because this is not a return statement
	_ = []TestStruct{{}} // want "empty_returns.TestStruct is missing fields A, B"
}

func shouldPassNestedEmptyInReturn() (NestedStruct, error) {
	// Should pass because AllowEmptyReturns is true
	return NestedStruct{}, nil
}