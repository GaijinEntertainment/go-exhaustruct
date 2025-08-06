package empty_declarations

type TestStruct struct {
	A string
	B int
}

type NestedStruct struct {
	Inner TestStruct
	Value string
}

func shouldPassEmptyStructInVarDeclaration() {
	// Should pass because AllowEmptyDeclarations is true
	var test = TestStruct{}
	_ = test
}

func shouldPassEmptyStructInShortDeclaration() {
	// Should pass because AllowEmptyDeclarations is true
	test := TestStruct{}
	_ = test
}

func shouldPassPointerToEmptyStructInDeclaration() {
	// Should pass because AllowEmptyDeclarations is true
	test := &TestStruct{}
	_ = test
}

func shouldFailEmptyStructInReturn() TestStruct {
	// Should fail because this is a return statement, not a declaration
	return TestStruct{} // want "empty_declarations.TestStruct is missing fields A, B"
}

func shouldFailEmptyStructInSliceNotDeclaration() {
	// Should fail because this is not a variable declaration
	_ = []TestStruct{{}} // want "empty_declarations.TestStruct is missing fields A, B"
}

func shouldPassNestedEmptyInDeclaration() {
	// Should pass because AllowEmptyDeclarations is true
	nested := NestedStruct{}
	_ = nested
}

func shouldPassMultivalueDeclaration() {
	// Should pass because AllowEmptyDeclarations is true
	a, b := TestStruct{}, NestedStruct{}
	_ = a
	_ = b
}

func shouldFailMultivalueInReturn() (TestStruct, NestedStruct) {
	// Should fail because this is a return statement, not a declaration
	return TestStruct{}, NestedStruct{} // want "empty_declarations.TestStruct is missing fields A, B" "empty_declarations.NestedStruct is missing fields Inner, Value"
}

func shouldFailDirectAssignment() {
	var test TestStruct
	// Should fail because this is an assignment, not a declaration
	test = TestStruct{} // want "empty_declarations.TestStruct is missing fields A, B"
	_ = test
}
