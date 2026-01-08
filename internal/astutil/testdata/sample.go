// Package sample is a test file for parser testing.
package sample

type SampleStruct struct {
	Required string
	//exhaustruct:optional
	OptionalByDirective string
	OptionalByTag       string `exhaustruct:"optional"`
}

type AnotherStruct struct {
	A int
	B string //exhaustruct:optional
}
