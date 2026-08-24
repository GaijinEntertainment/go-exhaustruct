// Package promoted tests composite literals that name promoted fields, which
// Go 1.27 permits (golang/go#77245). Naming a promoted field and its enclosing
// embedded field in one literal does not compile, so every case here picks one.
package promoted

type C struct{ c int }

type B struct {
	C

	b string
}

type A struct {
	B

	a int
}

func shouldPassAllPromoted() {
	_ = A{b: "x", a: 1, c: 2}
}

func shouldPassEmbeddedNamed() {
	_ = A{B: B{C: C{c: 1}, b: "x"}, a: 1}
}

func shouldPassPromotedEmbeddedNamed() {
	_ = A{C: C{c: 1}, b: "x", a: 1}
}

func shouldFailNothingFromSubtree() {
	_ = A{a: 1} // want "promoted.A is missing field B"
}

func shouldFailDeeperEmbeddedLeftOut() {
	_ = A{b: "x", a: 1} // want "promoted.A is missing field C"
}

func shouldFailDeepestLeafOnly() {
	_ = A{c: 1, a: 2} // want "promoted.A is missing field b"
}

func shouldFailOuterAndPromoted() {
	_ = A{c: 1} // want "promoted.A is missing fields b, a"
}

// OptedOut marks the embedded field itself optional, which carries the fields
// it promotes out of the check along with it. A type marked optional carries
// nothing of the kind, which is what OptionalHolder above shows.
type optedOutInner struct {
	A int
	B int
}

type OptedOut struct {
	//exhaustruct:optional
	optedOutInner

	Own int
}

func shouldPassEmbeddedFieldOptedOut() {
	_ = OptedOut{Own: 1}
}
