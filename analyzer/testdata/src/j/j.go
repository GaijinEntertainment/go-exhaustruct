package j

type Test struct {
	A string
	B int
	C float32
	D bool
	E string `exhaustruct:"optional"`
}

type AError struct{}

func (AError) Error() string { return "error message" }

func shouldPassEmptyStructWithAError() (Test, error) {
	return Test{}, &AError{}
}

type BError struct{ msg string }

func (e BError) Error() string { return e.msg }

func shouldFailEmptyStructWithEmptyBError() (Test, error) {
	return Test{}, &BError{} // want "j.BError is missing field msg"
}

func shouldPassEmptyStructWithFilledBError() (Test, error) {
	return Test{}, &BError{msg: "error message"}
}

func shouldPassEmptyStructWithFilledAError() (Test, error) {
	return Test{}, &AError{}
}

func shouldFailEmptyStructWithNilError() (Test, error) {
	return Test{}, nil // want "j.Test is missing fields A, B, C, D"
}

func shouldPassFilledStructWithNilError() (Test, error) {
	return Test{"", 0, 0.0, false, ""}, nil
}

func shouldPassFilledStructWithNilErrorUsingLambda() (Test, error) {
	f := func() (Test, error) {
		return Test{"", 0, 0.0, false, ""}, nil
	}
	return f()
}

func shouldFailEmptyStructWithNilErrorUsingLambda() (Test, error) {
	f := func() (Test, error) {
		return Test{}, nil // want "j.Test is missing fields A, B, C, D"
	}
	return f()
}

func shouldFailEmptyStructWithEmptyErrorUsingLambda() (Test, error) {
	f := func() (Test, error) {
		return Test{}, &BError{} // want "j.BError is missing field msg"
	}
	return f()
}

func shouldPassEmptyStructWithEmptyErrorUsingLambda() (Test, error) {
	f := func() (Test, error) {
		return Test{}, &AError{}
	}
	return f()
}
