// Package promoted tests composite literals that name promoted fields, which
// Go 1.27 permits (golang/go#77245). Naming a promoted field and its enclosing
// embedded field in one literal does not compile, so every case here picks one.
package promoted

import "go127/dep"

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

// EnforcedInside carries a field enforced in its own right, which outranks a
// type that marks every field optional.
type EnforcedInside struct {
	Loose string
	//exhaustruct:enforce
	Strict string
}

//exhaustruct:optional
type OptionalHolder struct {
	EnforcedInside

	Own int
}

func shouldFailEnforcedThroughOptionalEmbedded() {
	_ = OptionalHolder{Own: 1} // want "promoted.OptionalHolder is missing field Strict"
	_ = OptionalHolder{}       // want "promoted.OptionalHolder is missing field Strict"
}

// Derived takes its fields from another package, so the unexported hidden is
// not writable here. Its exported X is, by promotion.
type Derived dep.Base

func shouldFailPromotedThroughInaccessibleEmbedded() {
	_ = Derived{Own: 1}       // want "promoted.Derived is missing field X"
	_ = dep.Base{Own: 1}      // want "dep.Base is missing field X"
	_ = Derived{Own: 1, X: 2}
}

// Shadowing carries a field enforced in its own right out of reach: `x` in a
// literal of Shadowing writes the direct field, and nothing writes Hidden.x.
type Hidden struct {
	//exhaustruct:enforce
	x int
}

//exhaustruct:optional
type Shadowing struct {
	Hidden

	x int
}

func shouldPassEnforcedFieldShadowed() {
	_ = Shadowing{x: 1}
}

// D is reached twice at one depth from Ambiguous, and at two depths from
// Layered and Twice. Go resolves a promoted name to its shallowest occurrence,
// and to none at all where two occurrences share a depth.
type D struct {
	//exhaustruct:enforce
	X int
}

type wrapA struct{ D }

type wrapB struct{ D }

//exhaustruct:optional
type Ambiguous struct {
	wrapA
	wrapB
}

type mid struct{ D }

//exhaustruct:optional
type Layered struct {
	D
	mid
}

type Twice struct {
	D
	mid
}

func shouldPassAmbiguousPromotedName() {
	_ = Ambiguous{}
}

func shouldFailShallowestPromotedName() {
	_ = Layered{} // want "promoted.Layered is missing field X"
}

func shouldFailKeyFillsTheShallowestPath() {
	_ = Twice{X: 1} // want "promoted.Twice is missing field mid"
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

// hidden repeats the name of the unexported struct dep.Base embeds. The two are
// declared in different packages, so they are different identifiers: a literal
// written here reaches this one, and the one under Base shadows nothing.
type hidden struct{ Local int }

type withLocal struct{ hidden }

type Collide struct {
	withLocal

	dep.Base
}

func shouldPassLocalNameOfAForeignUnexportedField() {
	_ = Collide{hidden: hidden{Local: 1}, Base: dep.Base{X: 3, Own: 2}}
}

func shouldFailLocalNameOfAForeignUnexportedField() {
	_ = Collide{Base: dep.Base{X: 3, Own: 2}} // want "promoted.Collide is missing field withLocal"
}
