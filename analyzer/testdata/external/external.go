// Package external provides common types for cross-package testing.
// These types are imported by other test packages to verify behavior
// with external types, especially regarding unexported fields.
package external

// Simple is a struct with exported, unexported, and optional fields.
// When used from another package, only A and B should be required.
type Simple struct {
	A string
	B int
	C string `exhaustruct:"optional"`
	d string // unexported, inaccessible from other packages
}

// Excluded matches exclusion patterns (.*Excluded.*).
type Excluded struct {
	A string
	B int
	c string
}

// Empty is an empty struct for edge case testing.
type Empty struct{}

// OnlyUnexported has only unexported fields.
// When used from another package, no fields should be required.
type OnlyUnexported struct {
	a string
	b int
}
