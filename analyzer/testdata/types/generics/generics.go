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

// leftStructish and rightStructish both reach structish, so a constraint
// embedding the two reaches it twice. The second branch has to answer with what
// the first found rather than read the repeat as a cycle.
type leftStructish interface {
	structish
}

type rightStructish interface {
	structish
}

type diamondStructish interface {
	leftStructish
	rightStructish
}

func shouldFailDiamondConstraintLiteral[T diamondStructish]() T {
	return T{A: 1} // want "generics.T is missing field B"
}

// stringish names a method and no type at all.
type stringish interface {
	String() string
}

// methodStructish restricts methods as well as types. The embedded interface
// names no type, so it narrows the type set without erasing the core its
// sibling term carries.
type methodStructish interface {
	~struct {
		A int
		B string
	}
	stringish
}

func shouldFailMethodConstraintLiteral[T methodStructish]() T {
	return T{A: 1} // want "generics.T is missing field B"
}

// sameShape and alsoSameShape are two declarations of one shape, and only one
// of them makes B optional. Neither stands for the type set, so a union of both
// leaves the constraint without a core -- whichever order it lists them in.
type sameShape struct {
	A int
	//exhaustruct:optional
	B string
}

type alsoSameShape struct {
	A int
	B string
}

type twoShapes interface {
	sameShape | alsoSameShape
}

type twoShapesReversed interface {
	alsoSameShape | sameShape
}

func shouldPassTwoDeclarationsOfOneShape[T twoShapes]() T {
	return T{A: 1}
}

func shouldPassTwoDeclarationsOfOneShapeReversed[T twoShapesReversed]() T {
	return T{A: 1}
}

// twoShapesBesideStruct embeds twoShapes beside a term of the same shape. The
// type set is still the two declarations, which disagree, so the constraint has
// no core: the answer of the embedded interface stands, and the sibling term
// does not take its place.
type twoShapesBesideStruct interface {
	twoShapes
	~struct {
		A int
		B string
	}
}

func shouldPassNoCoreEmbeddedBesideStructTerm[T twoShapesBesideStruct]() T {
	return T{A: 1}
}

// equivalentOrders and alsoEquivalentOrders annotate one field in either order,
// which names the same optionality. The two are one core, and the literal is
// checked against it.
type equivalentOrders struct {
	A int
	//exhaustruct:optional,enforce
	B string
}

type alsoEquivalentOrders struct {
	A int
	//exhaustruct:enforce,optional
	B string
}

type equivalentAnnotations interface {
	equivalentOrders | alsoEquivalentOrders
}

func shouldFailEquivalentAnnotationOrders[T equivalentAnnotations]() T {
	return T{A: 1} // want "generics.T is missing field B"
}
