// Package anonymous tests anonymous struct behavior.
package anonymous

func shouldPassComplete() {
	_ = struct {
		A string
		B int
	}{A: "a", B: 1}
}

func shouldPassPointer() {
	_ = &struct {
		A string
		B int
	}{A: "a", B: 1}
}

func shouldFailEmpty() {
	_ = struct { // want "anonymous.<anonymous> is missing fields A, B"
		A string
		B int
	}{}
}

func shouldFailPartial() {
	_ = struct { // want "anonymous.<anonymous> is missing field A"
		A string
		B int
	}{B: 1}
}
