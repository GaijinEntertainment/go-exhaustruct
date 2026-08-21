// Package dep stands in for a third-party dependency. Its directives are
// malformed, and its source is typically read-only — the package consuming it
// can neither fix nor suppress a finding reported against this file.
package dep

type Shared struct {
	//exhaustruct:optionall // want `unknown directive`
	A int
	B int
}
