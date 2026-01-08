// Package anonymous tests anonymous struct behavior.
package anonymous

func shouldPassComplete() {
	_ = struct {
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

func shouldPassAnonymousWithIgnoreDirective() {
	//exhaustruct:ignore
	_ = struct {
		A string
		B int
	}{}

	_ = struct { //exhaustruct:ignore
		A string
		B int
	}{}
}

func shouldPassAnonymousInSliceWithIgnore() {
	_ = []struct {
		A string
		B int
	}{
		{A: "a", B: 1},
		{}, //exhaustruct:ignore
		{A: "b", B: 2},
	}
}