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
