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

// --- TestOptionalByPattern tests ---

func shouldPassOptionalEmpty() {
	// Type matches optional-rx pattern, so empty literal is allowed
	_ = TestOptionalByPattern{}
}

func shouldPassOptionalPartial() {
	// Type matches optional-rx pattern, so partial literal is allowed
	_ = TestOptionalByPattern{A: ""}
}

func shouldPassOptionalFull() {
	_ = TestOptionalByPattern{A: "", B: 0, C: 0.0}
}

// Verify multiple uses of same type don't cause mutation issues
func shouldPassOptionalMultiple() {
	_ = TestOptionalByPattern{}
	_ = TestOptionalByPattern{A: ""}
	_ = TestOptionalByPattern{B: 1}
	_ = TestOptionalByPattern{}
}

// --- TestRequired tests ---

func shouldPassRequiredFull() {
	_ = TestRequired{A: "", B: 0, C: 0.0}
}

func shouldFailRequiredEmpty() {
	_ = TestRequired{} // want "optionalpattern.TestRequired is missing fields A, B, C"
}

func shouldFailRequiredPartial() {
	_ = TestRequired{A: ""} // want "optionalpattern.TestRequired is missing fields B, C"
}

// Verify interleaved usage doesn't cause mutation
func shouldPassInterleavedUsage() {
	_ = TestOptionalByPattern{}  // optional via pattern
	_ = TestRequired{A: "", B: 0, C: 0.0} // required
	_ = TestOptionalByPattern{A: ""}      // still optional
}
