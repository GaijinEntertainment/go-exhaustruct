//line generated.tmpl:1

package testdata

// A generator that emits its //line ahead of the package clause remaps the
// file's own position as well, so the name this file is keyed under differs
// from the name it has to be read from. The directive above LinedOptional and
// the declaration of LinedDerived are only found by reading the physical file.

//exhaustruct:optional
type LinedOptional struct {
	A int
	B int
}

type LinedBase struct {
	A int
}

type LinedDerived LinedBase
