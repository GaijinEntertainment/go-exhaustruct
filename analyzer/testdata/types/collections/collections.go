// Package collections tests struct behavior in slices and maps.
package collections

// Test is a struct with optional field.
type Test struct {
	A string
	B int `exhaustruct:"optional"`
}

func shouldPassSlice() {
	_ = []Test{
		{"a", 1},       // positional
		{A: "a"},       // named, optional omitted
		Test{A: "b"},   // explicit type
	}
}

func shouldFailSlice() {
	_ = []Test{
		{},            // want "collections.Test is missing field A"
		Test{B: 123},  // want "collections.Test is missing field A"
	}
}

func shouldPassMap() {
	_ = map[string]Test{
		"a": {"a", 1},
		"b": {A: "a"},
		"c": Test{A: "b"},
	}
}

func shouldFailMap() {
	_ = map[string]Test{
		"a": {},           // want "collections.Test is missing field A"
		"b": Test{B: 123}, // want "collections.Test is missing field A"
	}
}
