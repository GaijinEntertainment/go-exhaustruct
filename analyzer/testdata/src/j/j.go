package j

type Test struct {
	A string
	B int
	C float32
	D bool
	E string `exhaustruct:"optional"`
}

type CustomEmptyError struct{}

func (CustomEmptyError) Error() string { return "custom error" }

func shouldPassEmptyStructWithCustomErr() (Test, error) {
	return Test{}, &CustomEmptyError{}
}

type CustomNonEmptyError struct{ msg string }

func (e CustomNonEmptyError) Error() string { return e.msg }

func shouldFailEmptyStructWithCustomNonEmptyErrorMissingFields() (Test, error) {
	return Test{}, &CustomNonEmptyError{} // want "j.CustomNonEmptyError is missing field msg"
}

func shouldPassEmptyStructWithFilledCustomNonEmptyError() (Test, error) {
	return Test{}, &CustomNonEmptyError{msg: "error message"}
}

func shouldPassEmptyStructWithCustomEmptyError() (Test, error) {
	return Test{}, &CustomEmptyError{}
}

func shouldFailEmptyStructWithNilError() (Test, error) {
	return Test{}, nil // want "j.Test is missing fields A, B, C, D"
}

func shouldPassFilledStructWithNilError() (Test, error) {
	return Test{"", 0, 0.0, false, ""}, nil
}
