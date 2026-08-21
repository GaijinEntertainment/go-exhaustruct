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

// structish constrains to a struct core type, so T{...} is a valid literal.
// The type parameter names the type at the use site, so it names the finding.
type structish interface {
	~struct {
		A int
		B string
	}
}

func shouldPassTypeParameterLiteral[T structish]() T {
	return T{A: 1, B: "x"}
}

func shouldFailTypeParameterLiteral[T structish]() T {
	return T{A: 1} // want "generics.T is missing field B"
}

// A directive above a generic function answers for the function, not for its
// type parameter. The parameter is declared in the signature the directive
// sits above, so reading its declaration line as the type's own would let the
// function decide what every literal of T requires.

//exhaustruct:ignore
func shouldFailIgnoreAboveGenericFunction[T structish]() T {
	return T{A: 1} // want "generics.T is missing field B"
}

//exhaustruct:optional
func shouldFailOptionalAboveGenericFunction[T structish]() T {
	return T{A: 1} // want "generics.T is missing field B"
}

// nestedStructish reaches its core type through an embedded named interface,
// which is as valid a constraint as naming the terms directly.
type nestedStructish interface {
	structish
}

func shouldFailNestedConstraintLiteral[T nestedStructish]() T {
	return T{A: 1} // want "generics.T is missing field B"
}
