package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

// missingFieldsVisitor checks struct literals for missing field initializations.
type missingFieldsVisitor struct {
	analyzer *analyzer
	pass     *analysis.Pass
	insp     *inspector.Inspector
}

func newMissingFieldsVisitor(
	a *analyzer,
	pass *analysis.Pass,
	insp *inspector.Inspector,
) *missingFieldsVisitor {
	return &missingFieldsVisitor{analyzer: a, pass: pass, insp: insp}
}

func (v *missingFieldsVisitor) run() {
	v.insp.WithStack([]ast.Node{(*ast.CompositeLit)(nil)}, v.visit)
}

func (v *missingFieldsVisitor) visit(n ast.Node, push bool, stack []ast.Node) bool {
	if !push {
		return true
	}

	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		return true
	}

	lv := literalVisitor{analysis: v, lit: lit, stack: stack}
	lv.process()

	return true
}

// literalVisitor carries context for processing a single composite literal.
type literalVisitor struct {
	analysis *missingFieldsVisitor
	lit      *ast.CompositeLit
	stack    []ast.Node
}

// literal holds resolved info for a struct literal being checked.
type literal struct {
	strct    *structure.Struct
	ignored  bool
	enforced bool
}

// shouldCheck implements checking decision priority.
func (l literal) shouldCheck(explicitMode bool) bool {
	if l.ignored {
		return false
	}

	if l.enforced {
		return true
	}

	if l.strct.IsIgnored() {
		return false
	}

	if l.strct.IsEnforced() {
		return true
	}

	return !explicitMode
}

func (lv literalVisitor) process() {
	lit, ok := lv.resolveLiteral()
	if !ok {
		return
	}

	if len(lv.lit.Elts) == 0 && lv.checkEmptyAllowed(lit.strct) {
		return
	}

	if pos, msg := lv.checkLiteral(lit); pos != nil {
		lv.analysis.pass.Reportf(*pos, "%s", msg)
	}
}

// resolveLiteral extracts struct type information from the composite literal,
// retrieves cached metadata, and looks up directives.
func (lv literalVisitor) resolveLiteral() (lit literal, ok bool) {
	typeName, strct, pos := lv.resolveLiteralType()
	if strct == nil {
		return literal{}, false //nolint:exhaustruct
	}

	s, diags := lv.analysis.analyzer.processor.ResolveStruct(
		lv.analysis.pass.Fset, typeName, strct, pos, lv.analysis.pass.Pkg,
	)

	for _, diag := range diags {
		lv.analysis.pass.Report(diag)
	}

	if s == nil {
		return literal{}, false //nolint:exhaustruct
	}

	litPos := lv.analysis.pass.Fset.Position(lv.lit.Pos())
	dirs, dirDiags := lv.analysis.analyzer.directives.Lookup(lv.analysis.pass.Fset, litPos)

	for _, d := range dirDiags {
		lv.analysis.pass.Report(d)
	}

	return literal{
		strct:    s,
		ignored:  dirs.Contains(directive.Ignore),
		enforced: dirs.Contains(directive.Enforce),
	}, true
}

// resolveLiteralType resolves the composite literal's type and definition position.
func (lv literalVisitor) resolveLiteralType() (name *types.TypeName, strct *types.Struct, pos token.Pos) {
	typ := lv.analysis.pass.TypesInfo.TypeOf(lv.lit)

	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	switch t := typ.(type) {
	case *types.Alias:
		name = t.Obj()
	case *types.Named:
		name = t.Obj()
	}

	typ = types.Unalias(typ)

	switch t := typ.(type) {
	case *types.Named:
		var ok bool
		if strct, ok = t.Underlying().(*types.Struct); !ok {
			return nil, nil, token.NoPos
		}

		pos = name.Pos()

		return name, strct, pos

	case *types.Struct:
		pos = lv.findAnonymousStructPos()

		return name, t, pos

	default:
		return nil, nil, token.NoPos
	}
}

// findAnonymousStructPos finds the position of the struct keyword for anonymous structs.
func (lv literalVisitor) findAnonymousStructPos() token.Pos {
	if lv.lit.Type != nil {
		if st, ok := lv.lit.Type.(*ast.StructType); ok {
			return st.Struct
		}

		return token.NoPos
	}

	for i := len(lv.stack) - 2; i >= 0; i-- { //nolint:mnd
		switch parent := lv.stack[i].(type) {
		case *ast.KeyValueExpr:
			continue

		case *ast.CompositeLit:
			return structPosFromType(parent.Type)

		default:
			return token.NoPos
		}
	}

	return token.NoPos
}

func structPosFromType(typ ast.Expr) token.Pos {
	if typ == nil {
		return token.NoPos
	}

	switch t := typ.(type) {
	case *ast.ArrayType:
		return structPosFromExpr(t.Elt)

	case *ast.MapType:
		return structPosFromExpr(t.Value)
	}

	return token.NoPos
}

func structPosFromExpr(expr ast.Expr) token.Pos {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	if st, ok := expr.(*ast.StructType); ok {
		return st.Struct
	}

	return token.NoPos
}

func (lv literalVisitor) checkEmptyAllowed(s *structure.Struct) bool {
	if lv.analysis.analyzer.config.AllowEmpty {
		return true
	}

	if s.AllowEmptyDecl {
		return true
	}

	if ret, ok := lv.getParentReturnStmt(); ok {
		if lv.analysis.analyzer.config.AllowEmptyReturns {
			return true
		}

		if lv.isErrorReturnStatement(ret) {
			return true
		}
	}

	if lv.isChildOfVariableDeclaration() && lv.analysis.analyzer.config.AllowEmptyDeclarations {
		return true
	}

	return false
}

func (lv literalVisitor) checkLiteral(lit literal) (*token.Pos, string) {
	if !lit.shouldCheck(lv.analysis.analyzer.config.ExplicitMode) {
		return nil, ""
	}

	strct := lit.strct

	missingFields := strct.SkippedFields(lv.lit, lv.analysis.pass.Pkg.Path())

	if len(missingFields) == 0 {
		return nil, ""
	}

	pos := lv.lit.Pos()

	displayName := strct.PackageName + "." + strct.Name
	if lv.analysis.analyzer.config.ReportFullTypePath {
		displayName = strct.FullPath
	}

	if len(missingFields) == 1 {
		return &pos, fmt.Sprintf("%s is missing field %s", displayName, structure.FormatFieldNames(missingFields))
	}

	return &pos, fmt.Sprintf("%s is missing fields %s", displayName, structure.FormatFieldNames(missingFields))
}

func (lv literalVisitor) isChildOfVariableDeclaration() bool {
	if len(lv.stack) < 2 { //nolint:mnd
		return false
	}

	for i := len(lv.stack) - 1; i > 0; i-- {
		parent := lv.stack[i-1]

		switch p := parent.(type) {
		case *ast.AssignStmt:
			if p.Tok == token.DEFINE {
				return true
			}

		case *ast.ValueSpec:
			return true

		case *ast.UnaryExpr:
			if p.Op == token.AND {
				continue
			}

			return false

		default:
			return false
		}
	}

	return false
}

func (lv literalVisitor) getParentReturnStmt() (*ast.ReturnStmt, bool) {
	if len(lv.stack) < 2 { //nolint:mnd
		return nil, false
	}

	for i := len(lv.stack) - 1; i > 0; i-- {
		parent := lv.stack[i-1]

		switch p := parent.(type) {
		case *ast.ReturnStmt:
			return p, true

		case *ast.UnaryExpr:
			if p.Op == token.AND {
				continue
			}

			return nil, false

		default:
			return nil, false
		}
	}

	return nil, false
}

//nolint:forcetypeassert,gochecknoglobals
var builtinErrorInterface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func (lv literalVisitor) isErrorReturnStatement(n *ast.ReturnStmt) bool {
	if len(n.Results) == 0 {
		return false
	}

	for i := len(n.Results) - 1; i >= 0; i-- {
		ri := n.Results[i]

		if ri == lv.lit {
			continue
		}

		switch ri := ri.(type) {
		case *ast.Ident:
			if ri.Name == "nil" {
				continue
			}

		case *ast.UnaryExpr:
			if ri.X == lv.lit {
				continue
			}
		}

		resultType := lv.analysis.pass.TypesInfo.TypeOf(ri)
		if resultType != nil && types.Implements(resultType, builtinErrorInterface) {
			return true
		}
	}

	return false
}
