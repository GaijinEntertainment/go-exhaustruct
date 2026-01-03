package directive

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"dev.gaijin.team/go/golib/e"
	"golang.org/x/tools/go/analysis"
)

// Parse errors.
var (
	ErrEmptyDirective      = e.New("empty directive")
	ErrUnknownDirective    = e.New("unknown directive")
	ErrDuplicateDirectives = e.New("duplicate directives")
)

// Directive represents an exhaustruct directive type.
type Directive string

const (
	// Ignore skips checking for a specific struct literal.
	Ignore Directive = "ignore"
	// Enforce forces check even if type is excluded.
	Enforce Directive = "enforce"
	// Optional marks a field as optional.
	Optional Directive = "optional"
)

// Validate returns true if the directive is a known valid directive.
func (d Directive) Validate() bool {
	switch d {
	case Ignore, Enforce, Optional:
		return true

	default:
		return false
	}
}

// Directives represents a collection of directives for a single line.
// Multiple directives can be specified comma-separated: //exhaustruct:enforce,optional.
type Directives []Directive

// Contains reports whether the directive d is present in the collection.
func (ds Directives) Contains(d Directive) bool {
	return slices.Contains(ds, d)
}

// directivePrefix is the exact prefix for exhaustruct directives.
// Format: //exhaustruct:<directives> [optional comment].
const directivePrefix = "//exhaustruct:"

func Parse(text string) (found bool, result Directives, errs []error) {
	text, found = strings.CutPrefix(text, directivePrefix)
	if !found {
		return false, nil, nil
	}

	if idx := strings.IndexAny(text, " \t\n"); idx > 0 {
		text = text[:idx]
	}

	if text == "" {
		return true, nil, []error{ErrEmptyDirective}
	}

	parts := strings.Split(text, ",")

	result = make(Directives, 0, len(parts))

	var hasDups bool

	for _, part := range parts {
		d := Directive(part)

		if !d.Validate() {
			errs = append(errs, ErrUnknownDirective.WithField("directive", d))
			continue
		}

		// giving the resulting size, linear search would be most efficient
		if slices.Contains(result, d) {
			hasDups = true
			continue
		}

		result = append(result, d)
	}

	if hasDups {
		errs = append(errs, ErrDuplicateDirectives)
	}

	if len(result) == 0 {
		result = nil
	}

	return true, result, errs
}

// File holds directives found in a single file, indexed by line number.
type File map[int]Directives

// Lookup returns the directives associated with the given line.
// Directives are pre-indexed by the line they apply to during parsing.
func (f File) Lookup(line int) Directives {
	return f[line]
}

// NewFile parses an AST file and extracts all exhaustruct directives.
// Returns diagnostics for comment groups containing multiple conflicting directives.
func NewFile(fset *token.FileSet, file *ast.File) (File, []analysis.Diagnostic) {
	directives, diagnostics := parseDirectives(fset, file.Comments)

	if len(directives) == 0 {
		return nil, diagnostics
	}

	// Adjust targets for inline comments
	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case nil, *ast.Comment, *ast.CommentGroup:
			return false
		}

		line := fset.Position(n.Pos()).Line
		if d, ok := directives[line]; ok {
			d.forLine = line
			directives[line] = d
		}

		return true
	})

	result := make(File, len(directives))

	for line, d := range directives {
		_, exists := result[d.forLine]
		if !exists {
			result[d.forLine] = d.directives
			continue
		}

		pos := d.pos

		// directives from block comments win over inline comments
		if line != d.forLine {
			result[d.forLine] = d.directives
			pos = directives[d.forLine].pos
		}

		diagnostics = append(diagnostics, analysis.Diagnostic{
			Pos:     pos,
			Message: "directive ignored, conflicting directive already exists for the same target line",
		})
	}

	return result, diagnostics
}

type parsedDirective struct {
	pos        token.Pos
	forLine    int
	directives Directives
}

func parseDirectives(
	fset *token.FileSet,
	comments []*ast.CommentGroup,
) (map[int]parsedDirective, []analysis.Diagnostic) {
	var (
		directives  = make(map[int]parsedDirective)
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
			directives[fset.Position(pos).Line] = parsedDirective{
				pos:        pos,
				forLine:    fset.Position(cg.End()).Line + 1,
				directives: parsed,
			}
		}
	}

	return directives, diagnostics
}
