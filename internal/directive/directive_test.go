package directive_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

func Test_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		text       string
		directives directive.Directives
		found      bool
		wantErrs   []error
	}{
		{
			name:       "no prefix",
			text:       "// regular comment",
			directives: nil,
			found:      false,
			wantErrs:   nil,
		},
		{
			name:       "ignore directive",
			text:       "//exhaustruct:ignore",
			directives: directive.Directives{directive.Ignore},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "enforce directive",
			text:       "//exhaustruct:enforce",
			directives: directive.Directives{directive.Enforce},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "optional directive",
			text:       "//exhaustruct:optional",
			directives: directive.Directives{directive.Optional},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "directive with trailing comment",
			text:       "//exhaustruct:ignore some reason",
			directives: directive.Directives{directive.Ignore},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "invalid directive name",
			text:       "//exhaustruct:invalid",
			directives: nil,
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective},
		},
		{
			name:       "partial directive name",
			text:       "//exhaustruct:opt",
			directives: nil,
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective},
		},
		{
			name:       "empty directive",
			text:       "//exhaustruct:",
			directives: nil,
			found:      true,
			wantErrs:   []error{directive.ErrEmptyDirective},
		},
		{
			name:       "space after slashes",
			text:       "// exhaustruct:ignore",
			directives: nil,
			found:      false,
			wantErrs:   nil,
		},
		{
			name:       "block comment",
			text:       "/*exhaustruct:ignore*/",
			directives: nil,
			found:      false,
			wantErrs:   nil,
		},
		{
			name:       "multiple directives comma-separated",
			text:       "//exhaustruct:enforce,optional",
			directives: directive.Directives{directive.Enforce, directive.Optional},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "multiple directives with invalid",
			text:       "//exhaustruct:enforce,invalid,optional",
			directives: directive.Directives{directive.Enforce, directive.Optional},
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective},
		},
		{
			name:       "all directives",
			text:       "//exhaustruct:ignore,enforce,optional",
			directives: directive.Directives{directive.Ignore, directive.Enforce, directive.Optional},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "multiple directives with trailing comment",
			text:       "//exhaustruct:enforce,optional some reason",
			directives: directive.Directives{directive.Enforce, directive.Optional},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "duplicate directives",
			text:       "//exhaustruct:enforce,enforce",
			directives: directive.Directives{directive.Enforce},
			found:      true,
			wantErrs:   []error{directive.ErrDuplicateDirectives},
		},
		{
			name:       "duplicate and invalid",
			text:       "//exhaustruct:enforce,invalid,enforce",
			directives: directive.Directives{directive.Enforce},
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective, directive.ErrDuplicateDirectives},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			found, d, errs := directive.Parse(tt.text)
			assert.Equal(t, tt.found, found, "found mismatch")
			assert.Equal(t, tt.directives, d, "directives mismatch")
			require.Len(t, errs, len(tt.wantErrs), "error count mismatch")

			for i, wantErr := range tt.wantErrs {
				assert.ErrorIs(t, errs[i], wantErr)
			}
		})
	}
}

func Test_File_Lookup(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	tests := []struct {
		name string
		src  string
		line int
		want directive.Directives
	}{
		{
			name: "no directives returns nil",
			src:  "package foo\n// regular comment\nvar x int\n",
			line: 10,
			want: nil,
		},
		{
			name: "doc comment on previous line",
			src:  "package foo\n//exhaustruct:optional\nvar x int\n",
			line: 3,
			want: directive.Directives{directive.Optional},
		},
		{
			name: "inline comment on same line",
			src:  "package foo\nvar x int //exhaustruct:ignore\n",
			line: 2,
			want: directive.Directives{directive.Ignore},
		},
		{
			name: "doc and inline on same target line - first wins",
			src: `package foo
//exhaustruct:optional
var x int //exhaustruct:enforce
`,
			line: 3,
			want: directive.Directives{directive.Optional}, // first wins
		},
		{
			name: "directive two lines above returns nil",
			src:  "package foo\n//exhaustruct:optional\n\nvar x int\n",
			line: 4,
			want: nil,
		},
		{
			name: "multi-directive on previous line",
			src:  "package foo\n//exhaustruct:enforce,optional\nvar x int\n",
			line: 3,
			want: directive.Directives{directive.Enforce, directive.Optional},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			require.NoError(t, err)

			fd, _ := directive.NewFile(fset, file)
			got := fd.Lookup(tt.line)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_NewFile_Testdata(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/directives.go", nil, parser.ParseComments)
	require.NoError(t, err)

	fd, diagnostics := directive.NewFile(fset, file)

	// Lines where we expect directives to apply
	expectDirective := map[int]directive.Directives{
		// Doc comments
		9:  {directive.Optional}, // docCommentApplies (doc on line 8)
		12: {directive.Enforce},  // docCommentEnforce (doc on line 11)

		// Inline comments
		16: {directive.Optional}, // inlineCommentApplies
		18: {directive.Enforce},  // inlineCommentEnforce

		// Inline does not affect next line
		22: {directive.Optional}, // lineWithInline

		// Multiple directives - first wins
		35: {directive.Optional}, // multipleDirectives (doc on lines 33-34)

		// Mixed doc and inline - first wins
		52: {directive.Optional}, // mixedDocAndInline

		// Ignore directive
		83: {directive.Ignore}, // ignoreDirective (doc on line 82)

		// Nested code
		89: {directive.Optional}, // nestedDoc (doc on line 88)
		92: {directive.Enforce},  // nestedInline

		// Struct fields
		100: {directive.Optional}, // DocField (doc on line 99)
		102: {directive.Optional}, // InlineField
		104: {directive.Optional}, // InlineFieldA
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
		assert.Equal(t, want, got, "line %d: expected %v, got %v", line, want, got)
	}

	// Verify no directives where expected
	for _, line := range expectNoDirective {
		got := fd.Lookup(line)
		assert.Empty(t, got, "line %d: expected no directive, got %v", line, got)
	}

	// Expect 4 diagnostics:
	// 1. Multiple directives in same comment group (lines 33-34)
	// 2. Mixed doc and inline targeting same line (line 52)
	// 3. Invalid directive on line 68 (//exhaustruct:opt)
	// 4. Invalid directive on line 71 (//exhaustruct:)
	assert.Len(t, diagnostics, 4, "expected 4 diagnostics")
}
