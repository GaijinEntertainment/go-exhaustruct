//nolint:all
package full_type_path

type Test struct {
	A string
	B int
}

func shouldFail() {
	_ = Test{} // want "full_type_path.Test is missing fields A, B"
	_ = Test{  // want "full_type_path.Test is missing field B"
		A: "",
	}
}