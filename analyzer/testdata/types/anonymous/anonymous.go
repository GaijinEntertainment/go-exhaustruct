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

// === Directive tests for anonymous structs ===

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

func shouldFailAnonymousInSliceWithoutIgnore() {
	_ = []struct {
		A string
		B int
	}{
		{A: "a", B: 1},
		{}, // want "anonymous.<anonymous> is missing fields A, B"
		{A: "b", B: 2},
	}
}

func shouldPassAnonymousInMapWithIgnore() {
	_ = map[string]struct {
		A string
		B int
	}{
		"a": {A: "a", B: 1},
		"b": {}, //exhaustruct:ignore
	}
}

func shouldFailAnonymousInMapWithoutIgnore() {
	_ = map[string]struct {
		A string
		B int
	}{
		"a": {A: "a", B: 1},
		"b": {}, // want "anonymous.<anonymous> is missing fields A, B"
	}
}

// Pointer slices with anonymous structs
func shouldFailPointerSliceAnonymous() {
	_ = []*struct {
		A string
		B int
	}{
		{A: "a", B: 1},
		{}, // want "anonymous.<anonymous> is missing fields A, B"
	}
}

func shouldPassPointerSliceAnonymousWithIgnore() {
	_ = []*struct {
		A string
		B int
	}{
		{A: "a", B: 1},
		{}, //exhaustruct:ignore
	}
}

// Nested anonymous structs with directives
func shouldHandleNestedAnonymousDirectives() {
	type Outer struct {
		Inner struct {
			X int
			Y int
		}
	}

	//exhaustruct:ignore
	_ = Outer{}

	_ = Outer{ // want "anonymous.Outer is missing field Inner"
		// Inner not initialized
	}
}
