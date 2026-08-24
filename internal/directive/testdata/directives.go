package testdata

// Test file for comment directive parsing.
// Each section demonstrates a specific behavior.

// === DOC COMMENTS (alone on line, apply to next line) ===

//exhaustruct:optional
var docCommentApplies int // line 9: directive on line 8 applies here

//exhaustruct:enforce
var docCommentEnforce int // line 12: directive on line 11 applies here

// === INLINE COMMENTS (after code, apply to same line) ===

var inlineCommentApplies int //exhaustruct:optional // line 16: directive applies to this line

var inlineCommentEnforce int //exhaustruct:enforce // line 18: directive applies to this line

// === INLINE COMMENT DOES NOT AFFECT NEXT LINE ===

var lineWithInline int //exhaustruct:optional // line 22: directive applies here only
var lineAfterInline int                       // line 23: NO directive here (inline above doesn't carry over)

// === DOC COMMENT WITH GAP DOES NOT APPLY ===

//exhaustruct:optional

var lineAfterGap int // line 29: NO directive (blank line 28 breaks the association)

// === MULTIPLE DIRECTIVES IN SAME COMMENT GROUP (first wins) ===

//exhaustruct:optional
//exhaustruct:enforce
var multipleDirectives int // line 35: "optional" wins (first directive)

// === BLOCK COMMENTS (as doc) ===

/*exhaustruct:optional*/
var blockDocComment int // line 40: optional

/* exhaustruct:enforce */
var blockDocWithSpaces int // line 43: none, a space after the delimiter makes the comment prose

// === BLOCK COMMENTS (inline) ===

var blockInline int /*exhaustruct:optional*/ // line 47: optional

// === MIXED: DOC AND INLINE ON SAME TARGET (doc wins, conflict reported) ===

//exhaustruct:optional
var mixedDocAndInline int //exhaustruct:enforce // line 52: "optional" wins (doc directive is first; inline emits conflict diagnostic)

// === REGULAR COMMENTS (no directive) ===

// This is just a regular comment
var regularComment int // line 57: NO directive

var regularInline int // just a comment // line 59: NO directive

// === DIRECTIVE IN MIDDLE OF COMMENT ===

// some text exhaustruct:optional more text
var directiveInMiddle int // line 64: NO directive (must start with //exhaustruct:)

// === PARTIAL DIRECTIVE NAMES ===

//exhaustruct:opt
var partialDirective int // line 69: NO directive (invalid directive name)

//exhaustruct:
var emptyDirective int // line 72: NO directive (empty directive value)

// === WHITESPACE VARIATIONS ===

// exhaustruct:optional
var directiveWithSpace int // line 76: NO directive (space after //)

//  exhaustruct:optional
var directiveWithTwoSpaces int // line 79: NO directive (spaces after //)

//exhaustruct:ignore
var ignoreDirective int // line 82: "ignore" directive applies here

// === DEEPLY NESTED CODE ===

func nestedCode() {
	//exhaustruct:optional
	var nestedDoc int // line 88: directive applies here
	_ = nestedDoc

	var nestedInline int //exhaustruct:enforce // line 91: directive applies here
	_ = nestedInline
}

// === STRUCT FIELD COMMENTS ===

type StructWithFieldComments struct {
	//exhaustruct:optional
	DocField int // line 99: directive applies here

	InlineField int //exhaustruct:optional // line 101: directive applies here

	InlineFieldA int //exhaustruct:optional // line 103: directive applies here
	NextField    int                        // line 104: NO directive

	//exhaustruct:optional

	FieldAfterGap int // line 108: NO directive (gap breaks association)
}

// === BLOCK COMMENTS SPANNING LINES ===

/*exhaustruct:optional
*/ var blockClosingBesideCode int // line 115: optional (block closes beside this code)

var blockOpeningBesideCode int /*exhaustruct:enforce
*/ // line 117: enforce (block opens beside this code)

/*exhaustruct:optional
*/
var blockClosingAlone int // line 122: optional (block touches no code)

// === DIRECTIVE BESIDE A CLOSING TOKEN ===

var closingBeside = []int{
	1,
} //exhaustruct:optional // line 128: targets line 126, where the value opens

var afterClosing int // line 130: NO directive (the one above belongs to the value)

// === NESTED CONSTRUCTS CLOSING ON ONE LINE ===

var nestedClosing = [][]int{
	{
		1,
	}} //exhaustruct:enforce // line 137: targets line 135, where the inner value opens

// === A LINE THAT CLOSES ONE CONSTRUCT AND OPENS ANOTHER ===

var closesAndOpens = map[string][]int{
	"a": {
		1,
	}, "b": { //exhaustruct:optional // line 144: targets itself, where the second value opens
		2,
	},
}
