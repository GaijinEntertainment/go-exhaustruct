// Package dep declares a struct whose exported fields are reached only by
// promotion through an unexported embedded one.
package dep

type hidden struct {
	X int
	y int
}

// Base embeds an unexported struct. A literal outside this package cannot name
// hidden, and since Go 1.27 it can name X.
type Base struct {
	hidden

	Own int
}
