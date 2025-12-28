// Package basic tests core struct literal behavior.
package basic

// Test is a struct with required and optional fields.
type Test struct {
	A string
	B int
	C float32
	D bool
	E string `exhaustruct:"optional"`
}

func shouldPassFullyDefined() {
	_ = Test{A: "", B: 0, C: 0.0, D: false, E: ""}
}

func shouldPassPointer() {
	_ = &Test{A: "", B: 0, C: 0.0, D: false}
}

func shouldPassOptionalOmitted() {
	_ = Test{A: "", B: 0, C: 0.0, D: false}
}

func shouldPassPositionalInit() {
	_ = []Test{{"", 0, 0.0, false, ""}}
}

func shouldFailEmpty() {
	_ = Test{} // want "basic.Test is missing fields A, B, C, D"
}

func shouldFailPartial() {
	_ = Test{A: "", B: 0, C: 0.0} // want "basic.Test is missing field D"
}
