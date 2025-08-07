package empty_error_returns

import (
	"fmt"
	"os"
)

type TestStruct struct {
	A string
}

type AError struct{}

func (AError) Error() string { return "error message" }

type BError struct{ msg string }

func (e BError) Error() string { return e.msg }

func shouldPassEmptyStructWithConcreteAError() (TestStruct, *AError) {
	// Should pass: TestStruct{} is allowed because there's a concrete error
	return TestStruct{}, &AError{}
}

func shouldFailEmptyStructWithEmptyBError() (TestStruct, error) {
	// Should pass for TestStruct{} but fail for BError{} - the error struct itself should be checked
	return TestStruct{}, &BError{} // want "empty_error_returns.BError is missing field msg"
}

func shouldFailEmptyStructWithNilConcreteError() (TestStruct, *BError) {
	// Should fail: TestStruct{} is not allowed because error is nil
	return TestStruct{}, nil // want "empty_error_returns.TestStruct is missing field A"
}

func shouldPassEmptyStructWithFmtError() (TestStruct, error) {
	// Should pass: TestStruct{} is allowed because there's a non-nil error
	return TestStruct{}, fmt.Errorf("error message")
}

func shouldPassEmptyStructWithStaticError() (TestStruct, error) {
	// Should pass: TestStruct{} is allowed because there's a non-nil error
	return TestStruct{}, os.ErrNotExist
}

func shouldPassEmptyStructWithAnonymousFunctionReturningError() (TestStruct, error) {
	// Should pass: TestStruct{} is allowed because there's a non-nil error
	return TestStruct{}, func() error { return nil }()
}

func shouldFailAnonymousFunctionReturningEmptyError() (TestStruct, error) {
	// Should fail: BError{} should be checked even inside anonymous function
	fn := func() error { return &BError{} } // want "empty_error_returns.BError is missing field msg"

	return TestStruct{}, fn()
}

func shouldFailEmptyNestedStructWithNonNilErr() ([]TestStruct, error) {
	// Should fail: TestStruct{} in slice should be checked even with non-nil error
	return []TestStruct{{}}, os.ErrNotExist // want "empty_error_returns.TestStruct is missing field A"
}