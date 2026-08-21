package directive_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v5/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v5/internal/directive"
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

	// A file's diagnostics belong to the file, not to the parse that found
	// them: the package owning it must receive them however many times the file
	// was already parsed, and by whom.
	assert.Equal(t, diagnostics, scanner.ProcessFiles(fset, file),
		"diagnostics are reported per file, not per parse")
}

// Test_Scanner_ProcessFiles_AfterLookupParsedIt covers a file another package
// parsed first. A consumer resolving an imported type reaches the file through
// Lookup, and the package that owns it is analysed after that. It receives the
// file's diagnostics all the same, where following the parse rather than the
// file left the owner with none and gave them to the first importer.
func Test_Scanner_ProcessFiles_AfterLookupParsedIt(t *testing.T) {
	t.Parallel()

	const filename = "testdata/directives.go"

	fset := token.NewFileSet()

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	// The consumer's lookup parses the file and keeps its diagnostics to
	// itself: they belong to the package that owns the file.
	scanner.Lookup(fset, token.Position{Filename: filename, Line: 1})

	_, _, size := scanner.Stats()
	require.Equal(t, uint64(1), size, "the lookup has to have parsed the file")

	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	require.NoError(t, err)

	assert.NotEmpty(t, scanner.ProcessFiles(fset, file),
		"the owner receives its file's diagnostics after another package parsed it")
}

// Test_Scanner_ProcessFiles_ForeignFileSet covers a file one pass parsed and
// another reports. A token.Pos means nothing outside the FileSet that produced
// it, and the two passes need not share one, so a kept finding is rebuilt
// against the FileSet the owner reports with.
func Test_Scanner_ProcessFiles_ForeignFileSet(t *testing.T) {
	t.Parallel()

	const filename = "testdata/directives.go"

	scanner := directive.NewScanner(astutil.NewFileParser())

	// A consumer resolving an imported type parses the file with its own set.
	foreign := token.NewFileSet()
	scanner.Lookup(foreign, token.Position{Filename: filename, Line: 1})

	// The owner analyses the same file with a set of its own, which holds other
	// files of the package and so numbers this one from a different base.
	own := token.NewFileSet()
	own.AddFile("other.go", -1, 4096)

	file, err := parser.ParseFile(own, filename, nil, parser.ParseComments)
	require.NoError(t, err)

	diags := scanner.ProcessFiles(own, file)
	require.NotEmpty(t, diags, "the owner has to receive its file's findings")

	for i, d := range diags {
		pos := own.Position(d.Pos)

		assert.Equal(t, filename, pos.Filename,
			"diagnostic %d has to name a position the reporting set resolves", i)
		assert.Positive(t, pos.Line, "diagnostic %d has to carry a line", i)
	}
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

	pos1 := token.Position{Filename: "file1.go", Line: 3}
	d := scanner.Lookup(fset, pos1)
	assert.Equal(t, directive.Directives{directive.Ignore}, d)

	pos2 := token.Position{Filename: "file2.go", Line: 3}

	d = scanner.Lookup(fset, pos2)
	assert.Equal(t, directive.Directives{directive.Enforce}, d)
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

	pos := token.Position{Filename: "test.go", Line: 3}

	d := scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Optional}, d)

	hits, misses, _ := scanner.Stats()
	assert.Equal(t, uint64(1), hits)
	assert.Equal(t, uint64(1), misses) // from ProcessFiles

	d = scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Optional}, d)

	hits, misses, _ = scanner.Stats()
	assert.Equal(t, uint64(2), hits)
	assert.Equal(t, uint64(1), misses)
}

func Test_Scanner_Lookup_EmptyFilename(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	pos := token.Position{}

	d := scanner.Lookup(fset, pos)
	assert.Nil(t, d)

	hits, misses, size := scanner.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, uint64(0), size)
}

// A definition file that cannot be read carries no directives anyone can act
// on, and it belongs to no package in the current run — reporting the failure
// would emit a diagnostic with no usable position. Both -trimpath, which records
// module-relative paths, and //line directives, which name templates and
// grammars, produce such filenames routinely.
func Test_Scanner_Lookup_UnreadableFile(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	for _, filename := range []string{
		"nonexistent.go",
		"example.com/module@v1.2.3/trimmed.go",
		"generated.tmpl",
	} {
		pos := token.Position{Filename: filename, Line: 1}

		assert.Nil(t, scanner.Lookup(fset, pos), "filename %q", filename)
	}
}

// Regression test for issue #166: standard library definitions are reported at
// "$GOROOT"-prefixed paths that cannot be opened. They carry no directives, so
// the lookup must report none instead of a read failure.
func Test_Scanner_Lookup_GoRoot(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	fp := astutil.NewFileParser()
	scanner := directive.NewScanner(fp)

	pos := token.Position{Filename: "$GOROOT/src/strings/builder.go", Line: 30}

	d := scanner.Lookup(fset, pos)
	assert.Nil(t, d)
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

	pos := token.Position{Filename: "test.go", Line: 1}
	d := scanner.Lookup(fset, pos)
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

	pos := token.Position{Filename: "shared.go", Line: 3}
	d := scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Enforce}, d)

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

	pos := token.Position{Filename: filename, Line: 4}

	d := scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Optional}, d)

	// Cold Lookup records exactly one miss and no hits — the post-parse
	// read for the just-written entry must not be counted.
	hits, misses, size := scanner.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(1), misses)
	assert.Equal(t, uint64(1), size)

	d = scanner.Lookup(fset, pos)
	assert.Equal(t, directive.Directives{directive.Optional}, d)

	hits, _, _ = scanner.Stats()
	assert.Equal(t, uint64(1), hits)
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

			pos := token.Position{Filename: "test.go", Line: tt.line}
			got := scanner.Lookup(fset, pos)

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

		// Block comments
		40: {directive.Optional}, // blockDocComment
		47: {directive.Optional}, // blockInline
	}

	// Lines where we expect NO directive
	expectNoDirective := []int{
		23,  // lineAfterInline (inline above doesn't carry over)
		29,  // lineAfterGap (blank line breaks association)
		43,  // blockDocWithSpaces (space after /* makes the comment prose)
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
		pos := token.Position{Filename: "testdata/directives.go", Line: line}
		got := scanner.Lookup(fset, pos)
		assert.Equal(t, want, got, "line %d: expected %v, got %v", line, want, got)
	}

	// Verify no directives where expected
	for _, line := range expectNoDirective {
		pos := token.Position{Filename: "testdata/directives.go", Line: line}
		got := scanner.Lookup(fset, pos)
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
	pos := token.Position{Filename: filename, Line: 4}

	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() {
			d := scanner.Lookup(fset, pos)
			assert.Equal(t, directive.Directives{directive.Optional}, d)
		})
	}

	wg.Wait()

	_, misses, _ := scanner.Stats()
	assert.Equal(t, uint64(1), misses, "file should be parsed once")
}

// Test_Scanner_LineDirective covers files carrying a //line directive. The
// virtual filename it names does not exist on disk, so the scanner has to key
// and read files by their physical position.
func Test_Scanner_LineDirective(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/lined.go", nil, parser.ParseComments)
	require.NoError(t, err)

	scanner := directive.NewScanner(astutil.NewFileParser())

	assert.Empty(t, scanner.ProcessFiles(fset, file),
		"a virtual filename must not produce a read error")

	var target token.Pos

	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
			target = gd.Pos()
		}
	}

	require.NotEqual(t, token.NoPos, target)

	// The adjusted position names virtual.y; the physical one names the file
	// the directive was actually written in.
	assert.Equal(t, filepath.Join("testdata", "virtual.y"), fset.Position(target).Filename)
	assert.Equal(t, directive.Directives{directive.Optional},
		scanner.Lookup(fset, fset.PositionFor(target, false)))
}

// Test_Scanner_LineDirectiveBeforePackage covers a //line ahead of the package
// clause, which remaps the file's own position. Keying the cache by the
// adjusted name would file the directives under a name no lookup resolves to.
func Test_Scanner_LineDirectiveBeforePackage(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	physical := filepath.Join("testdata", "line-before-package.go")

	file, err := parser.ParseFile(fset, physical, nil, parser.ParseComments)
	require.NoError(t, err)

	// The file's own position is remapped, which is what makes the cache key
	// differ from the physical name.
	require.Equal(t, filepath.Join("testdata", "generated.y"), fset.Position(file.Pos()).Filename)

	parser := astutil.NewFileParser()
	scanner := directive.NewScanner(parser)

	assert.Empty(t, scanner.ProcessFiles(fset, file))

	var target token.Pos

	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
			target = gd.Pos()
		}
	}

	require.NotEqual(t, token.NoPos, target)

	assert.Equal(t, directive.Directives{directive.Optional},
		scanner.Lookup(fset, fset.PositionFor(target, false)))

	// The parser records the file under the same physical name, so resolving a
	// definition in it later finds it already parsed rather than reading it a
	// second time and reporting its directives twice.
	assert.Empty(t, parser.ProcessFilename(fset, physical))

	hits, _, _ := parser.Stats()
	assert.Equal(t, uint64(1), hits)
}
