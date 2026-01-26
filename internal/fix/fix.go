// Package fix provides functionality to generate suggested fixes for missing struct fields.
package fix

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// MissingField represents a field that is missing from a struct literal.
type MissingField struct {
	Name string
	Type types.Type
}

// Generate creates a SuggestedFix that adds missing fields to a composite literal.
// It returns nil if the fix cannot be generated.
func Generate(fset *token.FileSet, lit *ast.CompositeLit, missing []MissingField) *analysis.SuggestedFix {
	if len(missing) == 0 || fset == nil || lit == nil {
		return nil
	}

	// Determine if the literal is multiline or single-line
	multiline := isMultilineLiteral(fset, lit)

	var textEdits []analysis.TextEdit

	if multiline {
		textEdits = generateMultilineEdits(fset, lit, missing)
	} else {
		textEdits = generateSingleLineEdits(fset, lit, missing)
	}

	if len(textEdits) == 0 {
		return nil
	}

	return &analysis.SuggestedFix{
		Message:   "Add missing struct fields",
		TextEdits: textEdits,
	}
}

// generateSingleLineEdits generates text edits for single-line composite literals.
func generateSingleLineEdits(fset *token.FileSet, lit *ast.CompositeLit, missing []MissingField) []analysis.TextEdit {
	var buf bytes.Buffer

	for i, field := range missing {
		if i > 0 || len(lit.Elts) > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(field.Name)
		buf.WriteString(": ")
		buf.WriteString(ZeroValueString(field.Type))
	}

	// Calculate the position for insertion
	var insertPos token.Pos
	if len(lit.Elts) > 0 {
		// Insert after the last element
		lastElt := lit.Elts[len(lit.Elts)-1]
		insertPos = lastElt.End()
	} else {
		// Empty literal: insert after the opening brace
		insertPos = lit.Lbrace + 1
	}

	return []analysis.TextEdit{
		{
			Pos:     insertPos,
			End:     insertPos,
			NewText: []byte(buf.String()),
		},
	}
}

// generateMultilineEdits generates text edits for multiline composite literals.
func generateMultilineEdits(fset *token.FileSet, lit *ast.CompositeLit, missing []MissingField) []analysis.TextEdit {
	// Detect the indentation of existing fields or calculate based on brace position
	indent := detectIndent(fset, lit)

	var buf bytes.Buffer

	hasExistingElts := len(lit.Elts) > 0

	// For empty multiline literals, we need to add a leading newline since we're
	// replacing everything between the braces (including the newline after {)
	if !hasExistingElts {
		buf.WriteString("\n")
	}

	// Add each field on its own line
	for _, field := range missing {
		buf.WriteString(indent)
		buf.WriteString(field.Name)
		buf.WriteString(": ")
		buf.WriteString(ZeroValueString(field.Type))
		buf.WriteString(",\n")
	}

	// Calculate the position for insertion
	var insertPos, endPos token.Pos
	if hasExistingElts {
		// Insert right before the closing brace
		// (the existing elements already have trailing newlines)
		insertPos = lit.Rbrace
		endPos = lit.Rbrace
	} else {
		// Empty multiline literal: replace the content between braces
		insertPos = lit.Lbrace + 1
		endPos = lit.Rbrace
	}

	return []analysis.TextEdit{
		{
			Pos:     insertPos,
			End:     endPos,
			NewText: []byte(buf.String()),
		},
	}
}

// isMultilineLiteral checks if the composite literal spans multiple lines.
func isMultilineLiteral(fset *token.FileSet, lit *ast.CompositeLit) bool {
	startPos := fset.Position(lit.Lbrace)
	endPos := fset.Position(lit.Rbrace)
	return startPos.Line != endPos.Line
}

// detectIndent returns the indentation string used for fields in the literal.
func detectIndent(fset *token.FileSet, lit *ast.CompositeLit) string {
	if len(lit.Elts) > 0 {
		// Use the indentation of the first element
		firstEltPos := fset.Position(lit.Elts[0].Pos())
		// Column is 1-indexed, so column 1 means no indentation
		if firstEltPos.Column > 1 {
			return makeIndent(firstEltPos.Column - 1)
		}
		return "\t"
	}

	// For empty multiline literals, add one level of indentation from the opening brace
	bracePos := fset.Position(lit.Lbrace)
	// Add one tab to the current indentation level
	return makeIndent(bracePos.Column-1) + "\t"
}

// makeIndent creates an indentation string with the specified number of spaces/columns.
// We try to detect if tabs are used based on the column being a multiple of common tab widths.
func makeIndent(columns int) string {
	if columns <= 0 {
		return ""
	}

	// Check if it looks like tab-based indentation (multiple of 4 or 8)
	// Most Go code uses tabs, which are typically rendered as 4 or 8 spaces
	if columns%4 == 0 {
		return repeatString("\t", columns/4)
	}
	if columns%8 == 0 {
		return repeatString("\t", columns/8)
	}

	// Fall back to a single tab for simplicity
	// This is a reasonable default for Go code
	return "\t"
}

// repeatString repeats a string n times.
func repeatString(s string, n int) string {
	if n <= 0 {
		return ""
	}
	var buf bytes.Buffer
	for range n {
		buf.WriteString(s)
	}
	return buf.String()
}

// ZeroValueString returns the string representation of the zero value for a type.
func ZeroValueString(t types.Type) string {
	// Handle aliases by getting the underlying type
	t = types.Unalias(t)

	switch typ := t.Underlying().(type) {
	case *types.Basic:
		return basicZeroValue(typ)
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return "nil"
	case *types.Array:
		return formatType(t) + "{}"
	case *types.Struct:
		return formatType(t) + "{}"
	case *types.Named:
		// This shouldn't happen after Underlying(), but handle it just in case
		return ZeroValueString(typ.Underlying())
	default:
		// Fallback for unknown types
		return formatType(t) + "{}"
	}
}

// basicZeroValue returns the zero value for basic types.
func basicZeroValue(t *types.Basic) string {
	switch t.Kind() {
	case types.Bool, types.UntypedBool:
		return "false"
	case types.String, types.UntypedString:
		return `""`
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Uintptr, types.UntypedInt:
		return "0"
	case types.Float32, types.Float64, types.UntypedFloat:
		return "0"
	case types.Complex64, types.Complex128, types.UntypedComplex:
		return "0"
	case types.UnsafePointer:
		return "nil"
	default:
		return "nil"
	}
}

// formatType returns the string representation of a type suitable for source code.
func formatType(t types.Type) string {
	// Use types.TypeString for a clean representation
	// This handles named types, generics, etc.
	return types.TypeString(t, func(pkg *types.Package) string {
		// Use package name (not path) for qualifier
		if pkg != nil {
			return pkg.Name()
		}
		return ""
	})
}

// FormatNode formats an AST node as a string.
func FormatNode(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}
