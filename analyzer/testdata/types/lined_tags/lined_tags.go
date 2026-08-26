//line generated.tmpl:1

package lined_tags

// A generator that emits its //line ahead of the package clause remaps every
// position in this file. The bytes a suggested fix is written against are found
// by reading the physical file, not the name the directive gives it.
type LinedTag struct {
	Required string
	// want +1 `struct tag "exhaustruct" is not supported anymore`
	Optional string `exhaustruct:"optional"`
}

// A generator is free to repeat one virtual line number, so two names on
// separate physical lines can read as one. What the fix writes is decided by
// the physical lines.
//
// A line directive names the line below it, so the migrated directive goes in
// front of the line directive rather than under it: written under it, it takes
// the pinned line for itself and moves the name to the next one.
type RepeatedLine struct {
	Required string
//line generated.tmpl:100
	First,
//line generated.tmpl:100
	Second string `exhaustruct:"optional"` // want `struct tag "exhaustruct" is not supported anymore`
}

// A line directive has a block form too. It names the position after the
// comment rather than the line below it, and a directive written under it
// moves the name past that position all the same.
type BlockLine struct {
	Required string
/*line generated.tmpl:200*/
	Tagged string `exhaustruct:"optional"` // want `struct tag "exhaustruct" is not supported anymore`
}

// The block form can also stand on the name's own line, in front of it. The
// name reads the line the directive names, and no comment can be written above
// it: gofmt puts the two on lines of their own, which moves the name off that
// line. Replacing the tag asks for no such comment, so a tag ending its line
// migrates in place.
type SameLineAppendable struct {
	Required string
/*line generated.tmpl:300*/ Tagged string /* want `struct tag "exhaustruct" is not supported anymore` */ `exhaustruct:"optional"`
}

// With the tag no longer ending the line, neither placement is open to it. The
// tag is reported and left alone rather than migrated at the cost of the line
// the generator pinned.
type SameLineBlocked struct {
	Required string
/*line generated.tmpl:400*/ Tagged string `exhaustruct:"optional"` /* want `struct tag "exhaustruct" is not supported anymore` */ // note
}
