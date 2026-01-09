package directive_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

func Test_Scanner_ProcessFiles(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/directives.go", nil, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	hits, misses, size := scanner.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, uint64(0), size)

	diagnostics := scanner.ProcessFiles(fset, file)
	assert.NotEmpty(t, diagnostics, "first ProcessFiles should return diagnostics")

	_, _, size = scanner.Stats()
	assert.Equal(t, uint64(1), size)

	diagnostics = scanner.ProcessFiles(fset, file)
	assert.Nil(t, diagnostics, "second ProcessFiles should return nil")
}

func Test_Scanner_ProcessFiles_MultipleFiles(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	src1 := "package foo\n//exhaustruct:ignore\nvar x int\n"
	file1, err := parser.ParseFile(fset, "file1.go", src1, parser.ParseComments)
	require.NoError(t, err)

	src2 := "package foo\n//exhaustruct:enforce\nvar y int\n"
	file2, err := parser.ParseFile(fset, "file2.go", src2, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	scanner.ProcessFiles(fset, file1)
	scanner.ProcessFiles(fset, file2)

	_, _, size := scanner.Stats()
	assert.Equal(t, uint64(2), size)

	pos1 := token.Position{Filename: "file1.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed
	d, diags := scanner.Lookup(fset, pos1)
	assert.Equal(t, directive.Directives{directive.Ignore}, d)
	assert.Nil(t, diags) // cache hit, no diagnostics

	pos2 := token.Position{Filename: "file2.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed

	d, diags = scanner.Lookup(fset, pos2)
	assert.Equal(t, directive.Directives{directive.Enforce}, d)
	assert.Nil(t, diags)
}

func Test_Scanner_Lookup(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	src := "package foo\n//exhaustruct:optional\nvar x int\n"
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	scanner.ProcessFiles(fset, file)

	pos := token.Position{Filename: "test.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed

	d, diags := scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Optional}, d)
	assert.Nil(t, diags)

	hits, misses, _ := scanner.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(1), misses) // from ProcessFiles

	d, diags = scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Optional}, d)
	assert.Nil(t, diags)

	hits, misses, _ = scanner.Stats()
	assert.Equal(t, uint64(2), hits)
	assert.Equal(t, uint64(1), misses)
}

func Test_Scanner_Lookup_EmptyFilename(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	pos := token.Position{} //nolint:exhaustruct // testing empty filename

	d, diags := scanner.Lookup(fset, pos)
	assert.Nil(t, d)
	assert.Nil(t, diags)

	hits, misses, size := scanner.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, uint64(0), size)
}

func Test_Scanner_Lookup_ParseError(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	pos := token.Position{Filename: "nonexistent.go", Line: 1} //nolint:exhaustruct // only Filename and Line needed

	d, diags := scanner.Lookup(fset, pos)
	assert.Nil(t, d)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "read file")
}

func Test_Scanner_Lookup_NoDirectiveAtLine(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	src := "package foo\n//exhaustruct:optional\nvar x int\n"
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)
	scanner.ProcessFiles(fset, file)

	pos := token.Position{Filename: "test.go", Line: 1} //nolint:exhaustruct // only Filename and Line needed
	d, _ := scanner.Lookup(fset, pos)
	assert.Nil(t, d)
}

func Test_Scanner_Lookup_AfterAdd(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	src := "package foo\n//exhaustruct:enforce\nvar x int\n"
	file, err := parser.ParseFile(fset, "shared.go", src, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	scanner.ProcessFiles(fset, file)

	pos := token.Position{Filename: "shared.go", Line: 3} //nolint:exhaustruct // only Filename and Line needed
	d, diags := scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Enforce}, d)
	assert.Nil(t, diags) // cache hit, no diagnostics

	hits, misses, _ := scanner.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(1), misses)
}

func Test_Scanner_Lookup_ProcessFilename(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	pos := token.Position{Filename: filename, Line: 4} //nolint:exhaustruct // only Filename and Line needed

	d, diags := scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Optional}, d)
	assert.Empty(t, diags)

	hits, misses, size := scanner.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(1), misses)
	assert.Equal(t, uint64(1), size)

	d, diags = scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Optional}, d)
	assert.Nil(t, diags)

	hits, _, _ = scanner.Stats()
	assert.Equal(t, uint64(2), hits)
}

func Test_Scanner_DirectiveLookup(t *testing.T) {
	t.Parallel()

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
			want: directive.Directives{directive.Optional},
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

			fset := token.NewFileSet()

			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			require.NoError(t, err)

			fp := astutil.NewFileParser()
			scanner := directive.NewScanner(fp)
			scanner.ProcessFiles(fset, file)

			pos := token.Position{Filename: "test.go", Line: tt.line} //nolint:exhaustruct
			got, _ := scanner.Lookup(fset, pos)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Scanner_Testdata(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/directives.go", nil, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)
	diagnostics := scanner.ProcessFiles(fset, file)

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
		pos := token.Position{Filename: "testdata/directives.go", Line: line} //nolint:exhaustruct
		got, _ := scanner.Lookup(fset, pos)
		assert.Equal(t, want, got, "line %d: expected %v, got %v", line, want, got)
	}

	// Verify no directives where expected
	for _, line := range expectNoDirective {
		pos := token.Position{Filename: "testdata/directives.go", Line: line} //nolint:exhaustruct
		got, _ := scanner.Lookup(fset, pos)
		assert.Empty(t, got, "line %d: expected no directive, got %v", line, got)
	}

	// Expect 4 diagnostics:
	// 1. Multiple directives in same comment group (lines 33-34)
	// 2. Mixed doc and inline targeting same line (line 52)
	// 3. Invalid directive on line 68 (//exhaustruct:opt)
	// 4. Invalid directive on line 71 (//exhaustruct:)
	assert.Len(t, diagnostics, 4, "expected 4 diagnostics")
}

func Test_Scanner_Lookup_Concurrent(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	filename := filepath.Join("testdata", "sample.go")

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	// Lookup same position concurrently WITHOUT pre-populating.
	// Verifies thread-safety of on-demand parsing.
	pos := token.Position{Filename: filename, Line: 4} //nolint:exhaustruct

	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			d, _ := scanner.Lookup(fset, pos)
			assert.Equal(t, directive.Directives{directive.Optional}, d)
		}()
	}

	wg.Wait()

	_, misses, _ := scanner.Stats()
	assert.Equal(t, uint64(1), misses, "file should be parsed once")
}
