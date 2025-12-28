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
