// Package error_returns tests error return behavior.
// Empty struct literals are allowed when returned with a non-nil error.
package error_returns

import (
	"fmt"
	"os"
)

// TestStruct is a simple struct.
type TestStruct struct {
	A string
}

// AError is an empty error type.
type AError struct{}

func (AError) Error() string { return "error" }

// BError is an error type with a field.
type BError struct{ msg string }

func (e BError) Error() string { return e.msg }

func shouldPassWithConcreteError() (TestStruct, *AError) {
	return TestStruct{}, &AError{}
}

func shouldPassWithFmtError() (TestStruct, error) {
	return TestStruct{}, fmt.Errorf("error")
}

func shouldPassWithStaticError() (TestStruct, error) {
	return TestStruct{}, os.ErrNotExist
}

func shouldPassWithFunctionError() (TestStruct, error) {
	return TestStruct{}, func() error { return nil }()
}

func shouldFailWithNilError() (TestStruct, *BError) {
	return TestStruct{}, nil // want "error_returns.TestStruct is missing field A"
}

func shouldFailErrorStructItself() (TestStruct, error) {
	return TestStruct{}, &BError{} // want "error_returns.BError is missing field msg"
}

func shouldFailErrorInAnonymousFunc() (TestStruct, error) {
	fn := func() error { return &BError{} } // want "error_returns.BError is missing field msg"
	return TestStruct{}, fn()
}

func shouldFailNestedWithError() ([]TestStruct, error) {
	// Struct in slice is checked even with non-nil error
	return []TestStruct{{}}, os.ErrNotExist // want "error_returns.TestStruct is missing field A"
}

func shouldPassEmptyInParenthesizedErrorReturn() (TestStruct, error) {
	return (TestStruct{}), fmt.Errorf("boom")
}

// The error result is parenthesized, and is still the error accompanying the
// literal under check.
func shouldPassEmptyWithParenthesizedErrorResult() (TestStruct, error) {
	return TestStruct{}, (&BError{"boom"})
}

// The literal under check is itself an error type and parenthesized. Telling it
// apart from an error returned beside it means looking through the parentheses
// on both sides of the comparison: taken at face value the literal counts as
// its own error, and its emptiness goes unreported.
func shouldFailParenthesizedErrorLiteral() (*BError, error) {
	return (&BError{}), nil // want "error_returns.BError is missing field msg"
}

// Repeated parentheses, which unparen strips to the literal underneath. Peeling
// one layer leaves a ParenExpr that is not the literal under check, so it
// counts as its own error again.
func shouldFailRepeatedlyParenthesizedErrorLiteral() (*BError, error) {
	return ((&BError{})), nil // want "error_returns.BError is missing field msg"
}
