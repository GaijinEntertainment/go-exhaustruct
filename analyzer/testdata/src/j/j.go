package j

import (
	"fmt"
	"os"
)

type Test struct {
	A string
}

type AError struct{}

func (AError) Error() string { return "error message" }

type BError struct{ msg string }

func (e BError) Error() string { return e.msg }

func shouldPassEmptyStructWithConcreteAError() (Test, *AError) {
	return Test{}, &AError{}
}

func shouldFailEmptyStructWithEmptyBError() (Test, error) {
	return Test{}, &BError{} // want "j.BError is missing field msg"
}

func shouldFailEmptyStructWithNilConcreteError() (Test, *BError) {
	return Test{}, nil // want "j.Test is missing field A"
}

func shouldPassEmptyStructWithFmtError() (Test, error) {
	return Test{}, fmt.Errorf("error message")
}

func shouldPassStaticError() (Test, error) {
	return Test{}, os.ErrNotExist
}

func shouldPassAnonymousFunctionReturningError() (Test, error) {
	return Test{}, func() error { return nil }()
}

func shouldFailAnonymousFunctionReturningEmptyError() (Test, error) {
	fn := func() error { return &BError{} } // want "j.BError is missing field msg"

	return Test{}, fn()
}

func shouldFailEmptyNestedStructWithNonNilErr() ([]Test, error) {
	return []Test{{}}, os.ErrNotExist // want "j.Test is missing field A"
}
