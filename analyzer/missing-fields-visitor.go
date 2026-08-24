package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"dev.gaijin.team/go/exhaustruct/v5/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v5/internal/structure"
)

// missingFieldsVisitor checks struct literals for missing field initializations.
type missingFieldsVisitor struct {
	pass      *analysis.Pass
	config    *Config
	processor *structure.Processor
}

func newMissingFieldsVisitor(
	pass *analysis.Pass,
	config *Config,
	processor *structure.Processor,
) *missingFieldsVisitor {
	return &missingFieldsVisitor{
		pass:      pass,
		config:    config,
		processor: processor,
	}
}

func (v *missingFieldsVisitor) run() {
	insp := v.pass.ResultOf[inspect.Analyzer].(*inspector.Inspector) //nolint:forcetypeassert

	insp.WithStack([]ast.Node{(*ast.CompositeLit)(nil)}, v.visit)
}

func (v *missingFieldsVisitor) visit(n ast.Node, push bool, stack []ast.Node) bool {
	if !push {
		return true
	}

	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		return true
	}

	lv := literalVisitor{missingFieldsVisitor: v, lit: lit, stack: stack}
	lv.process()

	return true
}

// literalVisitor carries context for processing a single composite literal.
type literalVisitor struct {
	*missingFieldsVisitor

	lit   *ast.CompositeLit
	stack []ast.Node
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
		lv.pass.Reportf(*pos, "%s", msg)
	}
}

// resolveLiteral extracts struct type information from the composite literal,
// retrieves cached metadata, and looks up directives.
//
// The named result carries the zero value on the paths that report ok=false,
// where the caller discards it and there is no struct to describe.
func (lv literalVisitor) resolveLiteral() (lit literal, ok bool) {
	typeName, strct, pos := lv.resolveLiteralType()
	if strct == nil {
		return lit, false
	}

	s := lv.processor.ResolveStruct(lv.pass.Fset, typeName, strct, pos, lv.pass.Pkg)
	if s == nil {
		return lit, false
	}

	dirs := lv.useSiteDirectives()

	return literal{
		strct:    s,
		ignored:  dirs.Contains(directive.Ignore),
		enforced: dirs.Contains(directive.Enforce),
	}, true
}

// useSiteDirectives collects the directives that apply to the literal.
//
// A directive is written above the statement holding a literal, not above every
// literal nested inside it, so the search climbs out of the expression the
// literal sits in and reads the statement holding it. Enumerating the node
// kinds a literal may be written within cannot be complete -- parentheses,
// selectors, operators, calls, sends and index expressions each nest literals,
// and the list grew by one every time a shape was missed -- so the walk asks
// what a node is instead: it climbs through expressions, and through the specs
// and declarations a var block is built from, until it reaches a statement.
//
// It stops there. A statement is where a directive is written, and going above
// one would let a directive on an `if` or a `for` reach every literal in the
// block below it.
func (lv literalVisitor) useSiteDirectives() directive.Directives {
	var dirs directive.Directives

	for _, node := range slices.Backward(lv.stack) {
		if !carriesUseSiteDirective(node) {
			break
		}

		// Unadjusted: the directive was written in the physical file, whatever
		// a //line directive renames it to.
		found := lv.processor.Directives().Lookup(
			lv.pass.Fset, lv.pass.Fset.PositionFor(node.Pos(), false),
		)

		for _, d := range found {
			if !dirs.Contains(d) {
				dirs = append(dirs, d)
			}
		}

		if _, isStatement := node.(ast.Stmt); isStatement {
			break
		}
	}

	return dirs
}

// carriesUseSiteDirective reports whether a directive written above node can
// apply to a composite literal nested inside it.
func carriesUseSiteDirective(node ast.Node) bool {
	switch node.(type) {
	case ast.Expr, ast.Spec, ast.Decl, ast.Stmt:
		return true

	default:
		return false
	}
}

// resolveLiteralType resolves the composite literal's type and definition position.
func (lv literalVisitor) resolveLiteralType() (name *types.TypeName, strct *types.Struct, pos token.Pos) {
	typ := lv.pass.TypesInfo.TypeOf(lv.lit)

	// A literal may elide `&T` when the element type is a pointer, including a
	// pointer reached through an alias or a defined pointer type. Strip it
	// before naming the type, so the name describes the struct rather than the
	// pointer to it.
	if ptr, ok := types.Unalias(typ).Underlying().(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	name = typeNameOf(typ)
	typ = types.Unalias(typ)

	switch t := typ.(type) {
	case *types.Named:
		var ok bool

		if strct, ok = t.Underlying().(*types.Struct); !ok {
			return nil, nil, token.NoPos
		}

		return name, strct, name.Pos()

	case *types.TypeParam:
		// A literal of a type parameter is written against its constraint's
		// core type, which the type checker has already validated the keys
		// against.
		if strct = coreStruct(t); strct == nil {
			return nil, nil, token.NoPos
		}

		// A type parameter is declared in a signature, and the core it resolves
		// to is a shape its constraint names rather than a type anyone declared.
		// There is no declaration to read type-level directives from, and the
		// parameter's own line would read the directives of the function.
		return name, strct, token.NoPos

	case *types.Struct:
		// An alias to an anonymous struct has a declaration of its own; only a
		// genuinely unnamed struct needs its position recovered from the AST.
		if name != nil {
			return name, t, name.Pos()
		}

		return name, t, lv.findAnonymousStructPos()

	default:
		return nil, nil, token.NoPos
	}
}

// typeNameOf returns the declaration a type is written under, or nil when the
// type is unnamed.
func typeNameOf(typ types.Type) *types.TypeName {
	switch t := typ.(type) {
	case *types.Alias:
		return t.Obj()

	case *types.Named:
		return t.Obj()

	case *types.TypeParam:
		return t.Obj()

	default:
		return nil
	}
}

// coreStruct returns the struct type every term of a type parameter's
// constraint shares, or nil when the constraint has no such core type and a
// composite literal of the parameter is therefore not possible.
func coreStruct(tp *types.TypeParam) *types.Struct {
	return coreResolver{walking: map[*types.Interface]bool{}}.constraintCore(tp.Constraint())
}

// coreResolver walks the interfaces a constraint embeds, holding the ones on
// the path it is walking. Marking an interface for the whole walk instead ends
// a cycle just the same, and reads the second branch of a diamond as one.
type coreResolver struct {
	walking map[*types.Interface]bool
}

// constraintCore resolves the struct shared by every term of a constraint,
// descending into the interfaces it embeds: a constraint may name its terms
// itself or inherit them from another interface, and both describe one type
// set.
func (r coreResolver) constraintCore(typ types.Type) *types.Struct {
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	// An interface embedding itself contributes no core of its own; whichever
	// branch opened it carries the answer.
	if r.walking[iface] {
		return nil
	}

	r.walking[iface] = true
	defer delete(r.walking, iface)

	var core *types.Struct

	for embedded := range iface.EmbeddedTypes() {
		for _, term := range unionTerms(embedded) {
			strct := r.termCore(term)
			if strct == nil {
				return nil
			}

			if core != nil && !types.Identical(core, strct) {
				return nil
			}

			core = strct
		}
	}

	return core
}

// termCore resolves one term of a constraint to the struct it stands for: the
// term's own type, or the core of another interface it names.
func (r coreResolver) termCore(term types.Type) *types.Struct {
	if _, ok := term.Underlying().(*types.Interface); ok {
		return r.constraintCore(term)
	}

	strct, _ := term.Underlying().(*types.Struct)

	return strct
}

// unionTerms splits a constraint element into its terms. A single-term element
// is not wrapped in a union, so it stands for itself.
func unionTerms(typ types.Type) []types.Type {
	union, ok := typ.(*types.Union)
	if !ok {
		return []types.Type{typ}
	}

	terms := make([]types.Type, 0, union.Len())

	for term := range union.Terms() {
		terms = append(terms, term.Type())
	}

	return terms
}

// findAnonymousStructPos finds the position of the struct keyword for anonymous structs.
func (lv literalVisitor) findAnonymousStructPos() token.Pos {
	if lv.lit.Type != nil {
		if st, ok := lv.lit.Type.(*ast.StructType); ok {
			return st.Struct
		}

		return token.NoPos
	}

	// Key and value of a map literal both elide their type, and their types come
	// from opposite sides of the map type — track which side the literal is on.
	mapKey := false

	for i := len(lv.stack) - 2; i >= 0; i-- { //nolint:mnd
		switch parent := lv.stack[i].(type) {
		case *ast.KeyValueExpr:
			mapKey = parent.Key == lv.stack[i+1]

			continue

		case *ast.CompositeLit:
			return structPosFromType(parent.Type, mapKey)

		default:
			return token.NoPos
		}
	}

	return token.NoPos
}

func structPosFromType(typ ast.Expr, mapKey bool) token.Pos {
	if typ == nil {
		return token.NoPos
	}

	switch t := typ.(type) {
	case *ast.ArrayType:
		return structPosFromExpr(t.Elt)

	case *ast.MapType:
		if mapKey {
			return structPosFromExpr(t.Key)
		}

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
	if lv.config.AllowEmpty {
		return true
	}

	if s.AllowEmptyDecl {
		return true
	}

	if ret, ok := lv.getParentReturnStmt(); ok {
		if lv.config.AllowEmptyReturns {
			return true
		}

		if lv.isErrorReturnStatement(ret) {
			return true
		}
	}

	if lv.isChildOfVariableDeclaration() && lv.config.AllowEmptyDeclarations {
		return true
	}

	return false
}

func (lv literalVisitor) checkLiteral(lit literal) (*token.Pos, string) {
	if !lit.shouldCheck(lv.config.ExplicitMode) {
		return nil, ""
	}

	strct := lit.strct

	missingFields := strct.SkippedFields(lv.lit, lv.pass.Pkg.Path())

	if len(missingFields) == 0 {
		return nil, ""
	}

	pos := lv.lit.Pos()

	displayName := strct.PackageName + "." + strct.Name
	if lv.config.ReportFullTypePath {
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

		case *ast.ParenExpr:
			continue

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

		case *ast.ParenExpr:
			continue

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

	for _, ri := range slices.Backward(n.Results) {
		ri = unparen(ri)

		if ri == lv.lit {
			continue
		}

		switch ri := ri.(type) {
		case *ast.Ident:
			if ri.Name == "nil" {
				continue
			}

		case *ast.UnaryExpr:
			if unparen(ri.X) == lv.lit {
				continue
			}

		default:
		}

		resultType := lv.pass.TypesInfo.TypeOf(ri)
		if resultType != nil && types.Implements(resultType, builtinErrorInterface) {
			return true
		}
	}

	return false
}

// unparen strips redundant parentheses, which gofmt keeps and which would
// otherwise break the walks that match a literal against its enclosing
// expression.
func unparen(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}

		expr = paren.X
	}
}
