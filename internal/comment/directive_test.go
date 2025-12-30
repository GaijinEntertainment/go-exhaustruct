package comment_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/comment"
)

func Test_ExtractDirective(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		text      string
		directive comment.Directive
		found     bool
	}{
		{
			name:      "no prefix",
			text:      "// regular comment",
			directive: "",
			found:     false,
		},
		{
			name:      "ignore directive",
			text:      "//exhaustruct:ignore",
			directive: comment.DirectiveIgnore,
			found:     true,
		},
		{
			name:      "enforce directive",
			text:      "//exhaustruct:enforce",
			directive: comment.DirectiveEnforce,
			found:     true,
		},
		{
			name:      "optional directive",
			text:      "//exhaustruct:optional",
			directive: comment.DirectiveOptional,
			found:     true,
		},
		{
			name:      "directive with trailing comment",
			text:      "//exhaustruct:ignore some reason",
			directive: comment.DirectiveIgnore,
			found:     true,
		},
		{
			name:      "invalid directive name",
			text:      "//exhaustruct:invalid",
			directive: "",
			found:     true,
		},
		{
			name:      "partial directive name",
			text:      "//exhaustruct:opt",
			directive: "",
			found:     true,
		},
		{
			name:      "empty directive",
			text:      "//exhaustruct:",
			directive: "",
			found:     true,
		},
		{
			name:      "space after slashes",
			text:      "// exhaustruct:ignore",
			directive: "",
			found:     false,
		},
		{
			name:      "block comment",
			text:      "/*exhaustruct:ignore*/",
			directive: "",
			found:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			directive, found := comment.ExtractDirective(tt.text)
			assert.Equal(t, tt.found, found, "found mismatch")
			assert.Equal(t, tt.directive, directive, "directive mismatch")
		})
	}
}

func Test_FileDirectives_Lookup(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	tests := []struct {
		name string
		src  string
		line int
		want comment.Directive
	}{
		{
			name: "no directives returns empty",
			src:  "package foo\n// regular comment\nvar x int\n",
			line: 10,
			want: "",
		},
		{
			name: "doc comment on previous line",
			src:  "package foo\n//exhaustruct:optional\nvar x int\n",
			line: 3,
			want: comment.DirectiveOptional,
		},
		{
			name: "inline comment on same line",
			src:  "package foo\nvar x int //exhaustruct:ignore\n",
			line: 2,
			want: comment.DirectiveIgnore,
		},
		{
			name: "doc and inline on same target line - first wins",
			src: `package foo
//exhaustruct:optional
var x int //exhaustruct:enforce
`,
			line: 3,
			want: comment.DirectiveOptional, // first wins
		},
		{
			name: "directive two lines above returns empty",
			src:  "package foo\n//exhaustruct:optional\n\nvar x int\n",
			line: 4,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			require.NoError(t, err)

			fd, _ := comment.NewFileDirectives(fset, file)
			got := fd.Lookup(tt.line)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_NewFileDirectives_Testdata(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/directives.go", nil, parser.ParseComments)
	require.NoError(t, err)

	fd, diagnostics := comment.NewFileDirectives(fset, file)

	// Lines where we expect directives to apply
	expectDirective := map[int]comment.Directive{
		// Doc comments
		9:  comment.DirectiveOptional, // docCommentApplies (doc on line 8)
		12: comment.DirectiveEnforce,  // docCommentEnforce (doc on line 11)

		// Inline comments
		16: comment.DirectiveOptional, // inlineCommentApplies
		18: comment.DirectiveEnforce,  // inlineCommentEnforce

		// Inline does not affect next line
		22: comment.DirectiveOptional, // lineWithInline

		// Multiple directives - first wins
		35: comment.DirectiveOptional, // multipleDirectives (doc on lines 33-34)

		// Mixed doc and inline - first wins
		52: comment.DirectiveOptional, // mixedDocAndInline

		// Ignore directive
		83: comment.DirectiveIgnore, // ignoreDirective (doc on line 82)

		// Nested code
		89: comment.DirectiveOptional, // nestedDoc (doc on line 88)
		92: comment.DirectiveEnforce,  // nestedInline

		// Struct fields
		100: comment.DirectiveOptional, // DocField (doc on line 99)
		102: comment.DirectiveOptional, // InlineField
		104: comment.DirectiveOptional, // InlineFieldA
	}

	// Lines where we expect NO directive
	expectNoDirective := []int{
		23,  // lineAfterInline (inline above doesn't carry over)
		29,  // lineAfterGap (blank line breaks association)
		40,  // blockDocComment (block comments not supported)
		43,  // blockDocWithSpaces (block comments not supported)
		47,  // blockInline (block comments not supported)
		57,  // regularComment
		59,  // regularInline
		64,  // directiveInMiddle (must start with //exhaustruct:)
		69,  // partialDirective (invalid directive name)
		72,  // emptyDirective
		77,  // directiveWithSpace (space after //)
		80,  // directiveWithTwoSpaces
		105, // NextField (inline above doesn't carry over)
		109, // FieldAfterGap (gap breaks association)
	}

	// Verify expected directives
	for line, want := range expectDirective {
		got := fd.Lookup(line)
		assert.Equal(t, want, got, "line %d: expected %q, got %q", line, want, got)
	}

	// Verify no directives where expected
	for _, line := range expectNoDirective {
		got := fd.Lookup(line)
		assert.Empty(t, got, "line %d: expected no directive, got %q", line, got)
	}

	// Expect 4 diagnostics:
	// 1. Multiple directives in same comment group (lines 33-34)
	// 2. Mixed doc and inline targeting same line (line 52)
	// 3. Invalid directive on line 68 (//exhaustruct:opt)
	// 4. Invalid directive on line 71 (//exhaustruct:)
	assert.Len(t, diagnostics, 4, "expected 4 diagnostics")
}
