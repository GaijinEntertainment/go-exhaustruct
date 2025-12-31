// Package sample is a test file for parser testing.
package sample

// SampleStruct has fields with comments.
type SampleStruct struct {
	// Required field.
	Required string
	//exhaustruct:optional
	OptionalByDirective string
	OptionalByTag       string `exhaustruct:"optional"`
}

// AnotherStruct for additional testing.
type AnotherStruct struct {
	A int
	B string //exhaustruct:optional
}
