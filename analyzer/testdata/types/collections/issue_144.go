// Issue #144: Pointer slices with omitted element type.
// https://github.com/GaijinEntertainment/go-exhaustruct/issues/144
//
// The analyzer should detect incomplete struct literals in slices
// of pointers even when the element type is omitted.
package collections

// Pointers is a named slice of pointers.
type Pointers []*Test

func shouldPassPointerSlice() {
	_ = []*Test{{A: "a"}, &Test{A: "b"}}
	_ = Pointers{{A: "a"}, &Test{A: "b"}}
}

func shouldFailPointerSlice() {
	_ = []*Test{
		{},            // want "collections.Test is missing field A"
		&Test{B: 123}, // want "collections.Test is missing field A"
	}
	_ = Pointers{
		{},            // want "collections.Test is missing field A"
		&Test{B: 123}, // want "collections.Test is missing field A"
	}
}

func shouldPassPointerMap() {
	_ = map[string]*Test{"a": {A: "a"}, "b": &Test{A: "b"}}
}

func shouldFailPointerMap() {
	_ = map[string]*Test{
		"a": {},           // want "collections.Test is missing field A"
		"b": &Test{B: 123}, // want "collections.Test is missing field A"
	}
}
