package comment

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Directive represents an exhaustruct directive type.
type Directive string

const (
	// DirectiveIgnore skips checking for a specific struct literal.
	DirectiveIgnore Directive = "ignore"
	// DirectiveEnforce forces check even if type is excluded.
	DirectiveEnforce Directive = "enforce"
	// DirectiveOptional marks a field as optional.
	DirectiveOptional Directive = "optional"
)

// directivePrefix is the exact prefix for exhaustruct directives.
// Format: //exhaustruct:<directive> [optional comment].
const directivePrefix = "//exhaustruct:"

// ExtractDirective extracts the directive from comment text if present.
// Returns (directive, true) if valid directive found.
// Returns ("", true) if prefix found but directive is invalid.
// Returns ("", false) if no directive prefix found.
func ExtractDirective(text string) (Directive, bool) {
	directive, found := strings.CutPrefix(text, directivePrefix)
	if !found {
		return "", false
	}

	// Extract just the directive word (stop at space or end)
	if idx := strings.IndexAny(directive, " \t\n"); idx > 0 {
		directive = directive[:idx]
	}

	switch Directive(directive) {
	case DirectiveIgnore, DirectiveEnforce, DirectiveOptional:
		return Directive(directive), true

	default:
		return "", true
	}
}

// FileDirectives holds directives found in a single file.
type FileDirectives map[int]Directive

// Lookup returns the directive associated with the given line.
// Directives are pre-indexed by the line they apply to during parsing.
func (f FileDirectives) Lookup(line int) Directive {
	return f[line]
}

// NewFileDirectives parses an AST file and extracts all exhaustruct directives.
// Returns diagnostics for comment groups containing multiple conflicting directives.
func NewFileDirectives(fset *token.FileSet, file *ast.File) (FileDirectives, []analysis.Diagnostic) {
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

	result := make(FileDirectives, len(directives))

	for line, d := range directives {
		_, exists := result[d.forLine]
		if !exists {
			result[d.forLine] = d.directive
			continue
		}

		pos := d.pos

		// directives from block comments win over inline comments
		if line != d.forLine {
			result[d.forLine] = d.directive
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
	pos       token.Pos
	forLine   int
	directive Directive
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
			directive, found := ExtractDirective(comment.Text)
			if !found {
				continue
			}

			pos := comment.Pos()

			if directive == "" {
				diagnostics = append(diagnostics, analysis.Diagnostic{
					Pos:     pos,
					Message: "invalid exhaustruct directive",
				})

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
				pos:       pos,
				forLine:   fset.Position(cg.End()).Line + 1,
				directive: directive,
			}
		}
	}

	return directives, diagnostics
}
