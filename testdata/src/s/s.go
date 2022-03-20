package s

import (
	"fmt"

	"e"
)

type Embedded struct {
	E string
	F string
	g string
	H string
}

type Test struct {
	A string
	B int
	C float32
	D bool
}

type Test2 struct {
	Embedded
	External e.External
}

func shouldPass() Test {
	return Test{
		A: "a",
		B: 1,
		C: 0.0,
		D: false,
	}
}

func shouldPass2() Test2 {
	return Test2{
		External: e.External{
			A: "",
			B: "",
		},
		Embedded: Embedded{
			E: "",
			F: "",
			H: "",
			g: "",
		},
	}
}

func shouldPassWithReturn() (Test, error) {
	if true {
		// Empty structs in return statements are ignored if also returning an error
		return Test{}, fmt.Errorf("error")
	}

	_ = Test{} // want "A, B, C, D are missing in Test"

	return Test{}, fmt.Errorf("error")
}
func shouldPass3() {
	// Checking to make sure state from tracking the previous return statement doesn't affect this struct
	_ = Test{} // want "A, B, C, D are missing in Test"
}

func shouldPassWithoutNames() Test {
	return Test{"", 0, 0, false}
}

func shouldFailWithReturn() (Test, error) {
	// Empty structs in return statements are not ignored if returning nil error
	return Test{}, nil // want "A, B, C, D are missing in Test"
}

func shouldFailWithMissingFields() Test {
	return Test{ // want "C is missing in Test"
		A: "a",
		B: 1,
		D: false,
	}
}

// Unchecked is a struct not listed in StructPatternList
type Unchecked struct {
	A int
	B int
}

func unchecked() {
	// This struct is not listed in StructPatternList so the linter won't complain that it's not filled out
	_ = Unchecked{
		A: 1,
	}
}

func excluded() {
	// this struct is excluded therefore should not be linted
	_ = e.ExternalExcluded{}
}

func shouldFailOnEmbedded() Test2 {
	return Test2{
		Embedded: Embedded{ // want "E, g, H are missing in Embedded"
			F: "",
		},
		External: e.External{
			A: "",
			B: "",
		},
	}
}

func shoildFailOnExternal() Test2 {
	return Test2{
		External: e.External{ // want "A is missing in External"
			B: "",
		},
		Embedded: Embedded{
			E: "",
			F: "",
			H: "",
			g: "",
		},
	}
}
