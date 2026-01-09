package tags

// Simple deprecated tag - only exhaustruct tag
type Simple struct {
	Required string
	Optional string `exhaustruct:"optional"` // want `struct tag "exhaustruct" is not supported anymore`
}

// Tag with other keys - should preserve other tags in fix
type WithOtherTags struct {
	Field string `json:"field" exhaustruct:"optional"` // want `struct tag "exhaustruct" is not supported anymore`
}

// Tag at start of tag string
type TagAtStart struct {
	Field string `exhaustruct:"optional" json:"field"` // want `struct tag "exhaustruct" is not supported anymore`
}

// Modern format - no warning expected
type Modern struct {
	//exhaustruct:optional
	Optional string
}

// Embedded field with deprecated tag
type WithEmbedded struct {
	Simple `exhaustruct:"optional"` // want `struct tag "exhaustruct" is not supported anymore`
}

// Invalid tag values - should just be removed
type WithInvalidTag struct {
	Field1 string `exhaustruct:"enforce"` // want `struct tag "exhaustruct" is not supported anymore`
	Field2 string `exhaustruct:"foo"`     // want `struct tag "exhaustruct" is not supported anymore`
}
