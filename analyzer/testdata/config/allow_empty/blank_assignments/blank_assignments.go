// Package blank_assignments tests AllowEmptyBlankAssignments=true behavior.
// Empty struct literals the blank identifier receives are allowed.
package blank_assignments

// TestStruct is a simple struct.
type TestStruct struct {
	A string
	B int
}

// Other reports a different field than TestStruct, so a line holding one of
// each tells which literal was reported.
type Other struct {
	C string
}

// Doer is the interface the compile-time checks below assert.
type Doer interface {
	Do()
}

func (TestStruct) Do() {}

var _ Doer = TestStruct{}

var _ Doer = &TestStruct{}

var _ Doer = (&TestStruct{})

var _ = TestStruct{}

var (
	_ Doer = TestStruct{}
	_      = &TestStruct{}
)

// A value is matched to its name by position: the literal a name receives is
// reported beside the one the blank identifier receives, on either side.
var _, named = TestStruct{}, Other{} // want "blank_assignments.Other is missing field C"

var other, _ = TestStruct{}, Other{} // want "blank_assignments.TestStruct is missing fields A, B"

func shouldPassBlankAssignment() {
	_ = TestStruct{}
	_ = &TestStruct{}
	_ = (TestStruct{})
}

// go/types reads a parenthesized target as the blank identifier too.
func shouldPassParenthesizedBlankTarget() {
	(_) = TestStruct{}
}

// Only an empty literal is exempt; a partial one is checked as anywhere else.
func shouldFailPartialInBlankAssignment() {
	_ = TestStruct{A: ""} // want "blank_assignments.TestStruct is missing field B"
}

// A target that is not an identifier is not the blank identifier.
func shouldFailIndexedTarget() {
	var items [1]TestStruct
	items[0] = TestStruct{} // want "blank_assignments.TestStruct is missing fields A, B"
}

func shouldPassBlankDeclarationInFunction() {
	var _ Doer = TestStruct{}
}

func shouldPassBlankInMultiValueDefine() {
	_, n := TestStruct{}, 1
	_ = n
}

func shouldFailNamedInMultiValueDefine() {
	n, _ := TestStruct{}, Other{} // want "blank_assignments.TestStruct is missing fields A, B"
	_ = n
}

func shouldFailNamedInMultiValueAssignment() {
	var n Other
	_, n = TestStruct{}, Other{} // want "blank_assignments.Other is missing field C"
	_ = n
}

func shouldFailDeclaration() {
	var n = TestStruct{} // want "blank_assignments.TestStruct is missing fields A, B"
	_ = n
}

func shouldFailShortDeclaration() {
	n := TestStruct{} // want "blank_assignments.TestStruct is missing fields A, B"
	_ = n
}

func shouldFailInReturn() TestStruct {
	return TestStruct{} // want "blank_assignments.TestStruct is missing fields A, B"
}

// The blank identifier receives the slice, not the literal inside it.
func shouldFailNestedInBlankAssignment() {
	_ = []TestStruct{{}} // want "blank_assignments.TestStruct is missing fields A, B"
}

// The blank identifier receives the converted value, not the literal.
func shouldFailConvertedInBlankAssignment() {
	_ = Doer(TestStruct{}) // want "blank_assignments.TestStruct is missing fields A, B"
}
