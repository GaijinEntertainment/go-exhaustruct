// Package blank tests structs carrying blank fields. A blank field cannot be
// named in a composite literal, so reporting it as missing yields a finding
// nobody can act on; positional literals still have to supply a value for it.
package blank

import "structs"

// NoCompare uses the zero-length func array idiom to block comparison.
type NoCompare struct {
	_ [0]func()
	A int
	B int
}

// Padded carries explicit padding between fields.
type Padded struct {
	A byte
	_ [3]byte
	B byte
}

// Layout uses the cgo-compatible layout marker.
type Layout struct {
	_ structs.HostLayout
	A int
}

func shouldPassNamedLiterals() {
	_ = NoCompare{A: 1, B: 2}
	_ = Padded{A: 1, B: 2}
	_ = Layout{A: 1}
}

func shouldPassPositionalLiterals() {
	_ = Padded{1, [3]byte{}, 2}
	_ = Layout{structs.HostLayout{}, 1}
}

func shouldPassAnonymousNamedLiteral() {
	_ = struct {
		_ [0]func()
		A int
	}{A: 1}
}

func shouldFailNamedLiteralMissingRealField() {
	_ = NoCompare{A: 1} // want "blank.NoCompare is missing field B"
	_ = Padded{A: 1}    // want "blank.Padded is missing field B"
}
