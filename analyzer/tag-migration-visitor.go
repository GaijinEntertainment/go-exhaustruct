package analyzer

import (
	"go/ast"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
)

// tagMigrationVisitor scans struct definitions for deprecated exhaustruct tags
// and emits migration diagnostics with suggested fixes.
type tagMigrationVisitor struct {
	pass *analysis.Pass
	insp *inspector.Inspector
}

func newTagMigrationVisitor(
	pass *analysis.Pass,
	insp *inspector.Inspector,
) *tagMigrationVisitor {
	return &tagMigrationVisitor{pass: pass, insp: insp}
}

// run uses inspector to traverse StructType nodes efficiently.
func (v *tagMigrationVisitor) run() {
	v.insp.Preorder([]ast.Node{new(ast.StructType)}, v.visitStructType)
}

func (v *tagMigrationVisitor) visitStructType(n ast.Node) {
	st, ok := n.(*ast.StructType)
	if !ok {
		return
	}

	if st.Fields == nil {
		return
	}

	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}

		value, ok := parseExhastructTag(field.Tag.Value)
		if !ok {
			continue
		}

		v.pass.Report(v.buildDiagnostic(field, value))
	}
}

const exhaustructTagKey = "exhaustruct"

// parseExhastructTag extracts value from `exhaustruct:"value"` tag.
// Returns ("", false) if tag not present.
func parseExhastructTag(tagLiteral string) (string, bool) {
	if len(tagLiteral) < 2 { //nolint:mnd
		return "", false
	}
	// Strip backticks
	inner := tagLiteral[1 : len(tagLiteral)-1]

	return reflect.StructTag(inner).Lookup(exhaustructTagKey)
}

func (v *tagMigrationVisitor) buildDiagnostic(
	field *ast.Field,
	tagValue string,
) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos:            field.Tag.Pos(),
		Message:        `struct tag "exhaustruct" is not supported anymore, use comment directives`,
		SuggestedFixes: []analysis.SuggestedFix{v.buildFix(field, tagValue)},
	}
}

func (*tagMigrationVisitor) buildFix(field *ast.Field, tagValue string) analysis.SuggestedFix {
	tag := field.Tag
	newTag := removeExhastructFromTag(tag.Value)

	if tagValue == "optional" {
		if newTag != "" {
			newTag += " "
		}

		newTag += "//exhaustruct:optional"
	}

	// Calculate start position (include leading space if removing entirely)
	startPos := tag.Pos()
	if newTag == "" {
		startPos = field.Type.End()
	}

	return analysis.SuggestedFix{
		Message: "fix",
		TextEdits: []analysis.TextEdit{{
			Pos:     startPos,
			End:     tag.End(),
			NewText: []byte(newTag),
		}},
	}
}

var exhaustructTagPattern = regexp.MustCompile(`\s*exhaustruct:"[^"]*"`)

func removeExhastructFromTag(tagLiteral string) string {
	tagLiteral = tagLiteral[1 : len(tagLiteral)-1]
	tagLiteral = strings.TrimSpace(exhaustructTagPattern.ReplaceAllString(tagLiteral, ""))

	if tagLiteral == "" {
		return ""
	}

	return "`" + tagLiteral + "`"
}
