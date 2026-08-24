// Package embedded tests embedded struct field behavior.
package embedded

import (
	"testdata/external"
)

// Embedded is a local type with unexported field.
type Embedded struct {
	E string
	F string
	g string // unexported
	H string
}

// TestEmbedded has an embedded field and external field.
type TestEmbedded struct {
	Embedded
	External external.Simple
}

func shouldPassComplete() {
	_ = TestEmbedded{
		Embedded: Embedded{E: "", F: "", g: "", H: ""},
		External: external.Simple{A: "", B: 0},
	}
}

func shouldFailMissingUnexported() {
	_ = TestEmbedded{
		Embedded: Embedded{E: "", F: "", H: ""}, // want "embedded.Embedded is missing field g"
		External: external.Simple{A: "", B: 0},
	}
}

func shouldFailMissingEmbedded() {
	_ = TestEmbedded{ // want "embedded.TestEmbedded is missing field Embedded"
		External: external.Simple{A: "", B: 0},
	}
}

func shouldFailMissingExternalField() {
	_ = TestEmbedded{
		Embedded: Embedded{E: "", F: "", g: "", H: ""},
		External: external.Simple{A: ""}, // want "external.Simple is missing field B"
	}
}

// EnforcedInside carries a field enforced in its own right, which outranks a
// type that marks every field optional. Reaching it through an embedded field
// has to answer the same as naming it directly.
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

//exhaustruct:optional
type OptionalDirect struct {
	Loose string
	//exhaustruct:enforce
	Strict string
}

func shouldFailEnforcedThroughOptionalEmbedded() {
	_ = OptionalDirect{Loose: ""} // want "embedded.OptionalDirect is missing field Strict"
	_ = OptionalDirect{}          // want "embedded.OptionalDirect is missing field Strict"
}

// This module declares Go 1.24, so a literal here cannot write Strict as a key:
// EnforcedInside is the one key that reaches it, and that is what is reported.
// go127/promoted covers the same shape where Strict itself can be written.
func shouldFailEnforcedBelowGo127() {
	_ = OptionalHolder{Own: 1} // want "embedded.OptionalHolder is missing field EnforcedInside"
	_ = OptionalHolder{}       // want "embedded.OptionalHolder is missing field EnforcedInside"
	_ = OptionalHolder{EnforcedInside: EnforcedInside{Loose: "", Strict: ""}}
}
