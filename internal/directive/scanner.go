package directive

import (
	"cmp"
	"go/ast"
	"go/token"
	"slices"

	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v5/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v5/internal/cache"
)

// Scanner provides thread-safe caching and lookup of file directives.
type Scanner struct {
	parser *astutil.FileParser
	cache  *cache.Cache[string, fileDirectives]
	// diags holds the parse diagnostics of each file, keyed by physical
	// filename. A file's diagnostics belong to the package that owns it, not to
	// whichever package happened to trigger the parse first.
	diags *cache.Cache[string, []fileDiagnostic]
}

// fileDiagnostic is one finding of a file, held without a token.Pos. A Pos is
// an index into the FileSet that produced it and means another file, or
// nothing, in any other. The file is parsed by whichever pass reaches it first
// and reported by the one that owns it, and those two need not share a FileSet,
// so the position is kept as an offset into the file and resolved again
// against the FileSet the owner reports with.
type fileDiagnostic struct {
	offset  int
	message string
}

const cachePreallocSize = 64

// NewScanner creates a new directive scanner that registers a callback
// with the file parser to extract directives from parsed files.
func NewScanner(parser *astutil.FileParser) *Scanner {
	s := &Scanner{
		parser: parser,
		cache:  cache.New[string, fileDirectives](cachePreallocSize),
		diags:  cache.New[string, []fileDiagnostic](cachePreallocSize),
	}

	parser.OnFileParsed(s.onFileParsed)

	return s
}

func (s *Scanner) onFileParsed(fset *token.FileSet, file *ast.File) []analysis.Diagnostic {
	filename := astutil.PhysicalFilename(fset, file.Pos())

	fd, diags := s.parseFileDirectives(fset, file)

	s.cache.Set(filename, fd)
	s.diags.Set(filename, offsetDiagnostics(fset, diags))

	return diags
}

// offsetDiagnostics keeps what a diagnostic says and where it says it, with the
// position taken as an offset into its file.
func offsetDiagnostics(fset *token.FileSet, diags []analysis.Diagnostic) []fileDiagnostic {
	held := make([]fileDiagnostic, 0, len(diags))

	for _, d := range diags {
		f := fset.File(d.Pos)
		if f == nil {
			continue
		}

		held = append(held, fileDiagnostic{offset: f.Offset(d.Pos), message: d.Message})
	}

	return held
}

// ProcessFiles pre-populates the cache by delegating to FileParser.ProcessFiles,
// and returns the directive diagnostics of exactly the given files.
//
// The diagnostics come from the per-file cache rather than from this parse, so a
// file reports its own findings even when an earlier lookup from another package
// already parsed it.
func (s *Scanner) ProcessFiles(fset *token.FileSet, files ...*ast.File) []analysis.Diagnostic {
	s.parser.ProcessFiles(fset, files...)

	var diags []analysis.Diagnostic

	for _, file := range files {
		held, ok := s.diags.Peek(astutil.PhysicalFilename(fset, file.Pos()))
		if !ok {
			continue
		}

		f := fset.File(file.Pos())
		if f == nil {
			continue
		}

		for _, d := range held {
			if d.offset < 0 || d.offset > f.Size() {
				continue
			}

			diags = append(diags, analysis.Diagnostic{
				Pos:     f.Pos(d.offset),
				Message: d.message,
			})
		}
	}

	return diags
}

// Lookup returns the directives at the given source position, which must be
// unadjusted — see astutil.PhysicalFilename. If the file is not in cache,
// triggers FileParser.ProcessFilename to parse it.
//
// Diagnostics found by such an on-demand parse are not returned: the file
// belongs to another package, which reports them through ProcessFiles.
func (s *Scanner) Lookup(fset *token.FileSet, pos token.Position) Directives {
	if pos.Filename == "" {
		return nil
	}

	if fd, ok := s.cache.Get(pos.Filename); ok {
		return fd[pos.Line]
	}

	// Cache miss - parse file (triggers onFileParsed callback, which stores
	// the result via cache.Set and increments the miss counter).
	s.parser.ProcessFilename(fset, pos.Filename)

	// Peek avoids counting this self-induced read as a hit — the miss was
	// already recorded by Set above.
	if fd, ok := s.cache.Peek(pos.Filename); ok {
		return fd[pos.Line]
	}

	// Still not in cache means parsing failed.
	return nil
}

func (s *Scanner) Stats() (hits, misses, size uint64) {
	return s.cache.Stats()
}

// fileDirectives holds directives found in a single file, indexed by line number.
type fileDirectives map[int]Directives

// parseFileDirectives parses an AST file and extracts all exhaustruct directives.
// Returns diagnostics in case file parsing errors, directive parsing errors, or
// conflicting directives for the same target line.
func (*Scanner) parseFileDirectives(fset *token.FileSet, file *ast.File) (fileDirectives, []analysis.Diagnostic) {
	directives, diagnostics := parseCommentDirectives(fset, file.Comments)

	if len(directives) == 0 {
		return nil, diagnostics
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case nil, *ast.Comment, *ast.CommentGroup:
			return false
		}

		// A directive sharing its line with code targets that line; one on a
		// line of its own targets the line below, which it already carries.
		line := fset.PositionFor(n.Pos(), false).Line
		for i := range directives[line] {
			directives[line][i].targetLine = line
		}

		return true
	})

	result, conflicts := reduceByTarget(directives)

	return result, append(diagnostics, conflicts...)
}

// reduceByTarget keeps one directive set for each target line -- the one
// written on a line of its own, or else the first written -- and reports every
// other directive aimed at that line as a conflict.
func reduceByTarget(directives map[int][]parsedDirective) (fileDirectives, []analysis.Diagnostic) {
	byTarget := make(map[int][]parsedDirective, len(directives))

	for _, ds := range directives {
		for _, d := range ds {
			byTarget[d.targetLine] = append(byTarget[d.targetLine], d)
		}
	}

	var diagnostics []analysis.Diagnostic

	result := make(fileDirectives, len(byTarget))

	for target, ds := range byTarget {
		slices.SortFunc(ds, func(a, b parsedDirective) int {
			// a directive on a line of its own wins over one written inline
			if aOwn, bOwn := a.line != target, b.line != target; aOwn != bOwn {
				if aOwn {
					return -1
				}

				return 1
			}

			return cmp.Compare(a.pos, b.pos)
		})

		result[target] = ds[0].directives

		for _, d := range ds[1:] {
			diagnostics = append(diagnostics, analysis.Diagnostic{
				Pos:     d.pos,
				Message: "directive ignored, conflicting directive already exists for the same target line",
			})
		}
	}

	return result, diagnostics
}

type parsedDirective struct {
	pos token.Pos
	// line is the line the directive is written on, which decides whether it
	// targets its own line or the one below.
	line       int
	targetLine int
	directives Directives
}

func parseCommentDirectives(
	fset *token.FileSet,
	comments []*ast.CommentGroup,
) (map[int][]parsedDirective, []analysis.Diagnostic) {
	var (
		// Keyed by the line the directive is written on, and holding every
		// directive on it: two comment groups can share one line, and both
		// reach the reduction by target line.
		directives  = make(map[int][]parsedDirective)
		diagnostics []analysis.Diagnostic
	)

	for _, cg := range comments {
		hasDirective := false

		for _, comment := range cg.List {
			found, parsed, errs := Parse(comment.Text)
			if !found {
				continue
			}

			pos := comment.Pos()

			for _, err := range errs {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Pos:     pos,
					Message: err.Error(),
				})
			}

			if len(parsed) == 0 {
				continue
			}

			if hasDirective {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Pos:     pos,
					Message: "multiple exhaustruct directives in a single comment group, ignoring",
				})

				continue
			}

			hasDirective = true

			line := fset.PositionFor(pos, false).Line

			directives[line] = append(directives[line], parsedDirective{
				pos:        pos,
				line:       line,
				targetLine: fset.PositionFor(cg.End(), false).Line + 1,
				directives: parsed,
			})
		}
	}

	return directives, diagnostics
}
