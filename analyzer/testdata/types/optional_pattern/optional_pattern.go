// Package optionalpattern tests optional-rx pattern behavior.
package optionalpattern

// TestOptionalByPattern is matched by optional-rx pattern.
// All fields should be treated as optional without struct mutation.
type TestOptionalByPattern struct {
	A string
	B int
	C float32
}

// TestRequired is NOT matched by optional-rx pattern.
// All fields should be required.
type TestRequired struct {
	A string
	B int
	C float32
}

func shouldPassOptionalEmpty() {
	// Type matches optional-rx pattern, so empty literal is allowed
	_ = TestOptionalByPattern{}
}

func shouldFailRequiredEmpty() {
	_ = TestRequired{} // want "optionalpattern.TestRequired is missing fields A, B, C"
}
