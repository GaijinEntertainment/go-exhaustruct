// Package generics tests generic struct behavior.
package generics

// testGenericStruct is a generic struct.
type testGenericStruct[T any] struct {
	A T
	B string
}

func shouldPassComplete() {
	_ = testGenericStruct[int]{A: 42, B: "value"}
	_ = testGenericStruct[string]{A: "key", B: "value"}
}

func shouldFailEmpty() {
	_ = testGenericStruct[int]{} // want "generics.testGenericStruct is missing fields A, B"
}

func shouldFailPartial() {
	_ = testGenericStruct[int]{A: 42} // want "generics.testGenericStruct is missing field B"
}
