// Package testdata provides test fixtures for structs package tests.
package testdata

// === BASIC STRUCTS ===

// Empty has no fields.
type Empty struct{}

// SingleField has one exported field.
type SingleField struct {
	Name string
}

// MultiField has multiple exported fields.
type MultiField struct {
	A string
	B int
	C bool
}

// MixedExported has both exported and unexported fields.
type MixedExported struct {
	Exported   string
	unexported int
	Another    bool
}

// AllUnexported has only unexported fields.
type AllUnexported struct {
	a string
	b int
}

// === DIRECTIVE-BASED OPTIONALITY ===

// WithOptionalDoc has a field marked optional via doc comment.
type WithOptionalDoc struct {
	Required string
	//exhaustruct:optional
	Optional int
}

// WithOptionalInline has a field marked optional via inline comment.
type WithOptionalInline struct {
	Required string
	Optional int //exhaustruct:optional
}

// WithEnforcedField has a field marked as enforced.
type WithEnforcedField struct {
	Normal string
	//exhaustruct:enforce
	Enforced int
}

// WithMixedDirectives has fields with different directives.
type WithMixedDirectives struct {
	Normal string
	//exhaustruct:optional
	Optional int
	//exhaustruct:enforce
	Enforced bool
}

// === STRUCT-LEVEL DIRECTIVES ===

//exhaustruct:ignore
type IgnoredStruct struct {
	A string
	B int
}

//exhaustruct:enforce
type EnforcedStruct struct {
	A string
	B int
}

//exhaustruct:optional
type OptionalStruct struct {
	A string
	B int
}

// === EMBEDDED FIELDS ===

// Embedded is a type to embed.
type Embedded struct {
	E string
}

// WithEmbedded has an embedded field.
type WithEmbedded struct {
	Embedded
	Own string
}

// unexported is an unexported embedded type.
type unexported struct {
	u string
}

// WithUnexportedEmbedded has an unexported embedded field.
type WithUnexportedEmbedded struct {
	unexported
	Own string
}