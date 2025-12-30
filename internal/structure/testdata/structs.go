package testdata

type emptyStruct struct{}

type commentOptionalStruct struct {
	Required string
	//exhaustruct:optional
	OptionalDoc string
	OptionalInline string //exhaustruct:optional
}

// commentEdgeCases tests comment consumption rules.
type commentEdgeCases struct {
	// Rule 4: end-of-line comment on previous line does NOT apply to next field
	FieldWithInlineComment string //exhaustruct:optional
	FieldAfterInline       string // should NOT be optional

	// Rule 3: block/doc comment on previous line IS consumed
	//exhaustruct:optional
	FieldWithDocComment string // should be optional

	// Field with no comments
	PlainField string // should NOT be optional
}

// duplicateDirectivesStruct tests that first directive wins when duplicates exist.
type duplicateDirectivesStruct struct {
	// Duplicate directives in doc comment - first (optional) wins
	//exhaustruct:optional
	//exhaustruct:enforce
	DuplicateDocField string

	NormalField string
}

type testStruct struct {
	// some random comment

	ExportedRequired   int
	unexportedRequired int

	ExportedOptional   int `exhaustruct:"optional"`
	unexportedOptional int `exhaustruct:"optional"`
}

var (
	_unnamed = testStruct{1, 2, 3, 4}
	_named   = testStruct{
		ExportedRequired:   1,
		unexportedRequired: 2,
		ExportedOptional:   3,
		unexportedOptional: 4,
	}
	_unnamedIncomplete = testStruct{1}
	_namedIncomplete1  = testStruct{
		ExportedRequired: 1,
		ExportedOptional: 3,
	}
	_namedIncomplete2 = testStruct{
		ExportedOptional:   3,
		unexportedOptional: 4,
	}
	_empty = testStruct{}
)
