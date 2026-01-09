package analyzer

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"runtime"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"dev.gaijin.team/go/exhaustruct/v4/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

type analyzer struct {
	config     Config
	directives *directive.Scanner
	processor  *structure.Processor `exhaustruct:"optional"`
}

func NewAnalyzer(config Config) (*analysis.Analyzer, error) {
	fp := astutil.NewFileParser()
	dirScanner := directive.NewScanner(fp)

	a := analyzer{
		config:     config,
		directives: dirScanner,
		processor: structure.NewProcessor(
			dirScanner,
			structure.NewOriginScanner(fp),
			structure.WithEnforce(config.EnforcePatterns),
			structure.WithIgnore(config.IgnorePatterns),
			structure.WithOptional(config.OptionalPatterns),
			structure.WithAllowEmpty(config.AllowEmptyPatterns),
		),
	}

	return &analysis.Analyzer{ //nolint:exhaustruct
		Name:     "exhaustruct",
		Doc:      "Checks if all structure fields are initialized",
		Run:      a.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Flags:    *a.config.BindToFlagSet(flag.NewFlagSet("", flag.PanicOnError)),
	}, nil
}

func (a *analyzer) run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector) //nolint:forcetypeassert

	// Pre-populate directive cache with files from this pass
	for _, diag := range a.directives.Add(pass.Fset, pass.Files...) {
		pass.Report(diag)
	}

	insp.WithStack([]ast.Node{(*ast.CompositeLit)(nil)}, a.newVisitorFunc(pass))

	newTagMigrationVisitor(pass, insp).run()

	if a.config.DebugCacheMetrics {
		a.printCacheStats(pass.Pkg.Path())
	}

	return nil, nil //nolint:nilnil
}

// literal holds resolved info for a struct literal being checked.
// Short-lived: created per composite literal, discarded after check.
type literal struct {
	strct    *structure.Struct
	typeName string // actual type name (may differ from strct.name for derived types)
	ignored  bool
	enforced bool
}

// shouldCheck implements checking decision priority.
// Priority: literal:ignore > literal:enforce > struct:ignore > struct:enforce > mode default.
// The only effect of explicit mode is the default: explicit defaults to skip, implicit to check.
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

// visitor carries context for processing a single composite literal.
// Small enough to pass by value (~48 bytes: 4 pointers + slice header).
type visitor struct {
	a     *analyzer
	pass  *analysis.Pass
	lit   *ast.CompositeLit
	stack []ast.Node
}

// resolveLiteralType resolves the composite literal's type and definition position.
// Returns (typeName, struct, pos) where:
//   - For named types: (typeName, struct, typeName.Pos())
//   - For type aliases: (aliasTypeName, struct, aliasTypeName.Pos()) - alias's own TypeName
//   - For anonymous structs: (nil, struct, pos)
//   - For non-struct types: (nil, nil, NoPos)
//
// Pointers are dereferenced (e.g., &Struct{} or []*Struct{{}}).
func (v visitor) resolveLiteralType() (name *types.TypeName, strct *types.Struct, pos token.Pos) {
	typ := v.pass.TypesInfo.TypeOf(v.lit)

	// Resolve pointers (e.g., &Struct{} or []*Struct{{}})
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	// Extract TypeName BEFORE unaliasing.
	// For aliases, this gives us the alias's TypeName (name, position).
	// For named types, this gives us the type's TypeName.
	switch t := typ.(type) {
	case *types.Alias:
		name = t.Obj()
	case *types.Named:
		name = t.Obj()
	}

	// Unalias to get the underlying struct type (if it is structu ofc =)).
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
		pos = v.findAnonymousStructPos()

		return name, t, pos

	default:
		return nil, nil, token.NoPos
	}
}

// findAnonymousStructPos finds the position of the struct keyword for anonymous structs.
// For explicit anonymous structs (struct{...}{}), returns position from lit.Type.
// For inferred types (inner slice/map elements), finds direct parent's type.
func (v visitor) findAnonymousStructPos() token.Pos {
	if v.lit.Type != nil {
		// explicit type definition
		if st, ok := v.lit.Type.(*ast.StructType); ok {
			return st.Struct
		}

		return token.NoPos
	}

	// Inferred type: find direct parent CompositeLit (skip only KeyValueExpr)
	// Start from len-2 since len-1 is the current literal
	for i := len(v.stack) - 2; i >= 0; i-- { //nolint:mnd
		switch parent := v.stack[i].(type) {
		case *ast.KeyValueExpr:
			continue

		case *ast.CompositeLit:
			return structPosFromType(parent.Type)

		default:
			// some weird situation with non-literal parent, for our case, the only known way
			// to have implicit type is map/array literals as direct parent
			return token.NoPos
		}
	}

	return token.NoPos
}

// structPosFromType extracts struct keyword position from array/map type expressions.
// Handles pointer types (e.g., []*struct{...}).
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

// structPosFromExpr extracts struct keyword position, unwrapping pointers if needed.
func structPosFromExpr(expr ast.Expr) token.Pos {
	// Unwrap pointer: *struct{...} -> struct{...}
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	if st, ok := expr.(*ast.StructType); ok {
		return st.Struct
	}

	return token.NoPos
}

// newVisitorFunc returns callback for [inspector.WithStack] that processes composite literals.
func (a *analyzer) newVisitorFunc(pass *analysis.Pass) func(n ast.Node, push bool, stack []ast.Node) bool {
	return func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		v := visitor{a: a, pass: pass, lit: lit, stack: stack}
		v.process()

		return true
	}
}

func (v visitor) process() {
	lit, ok := v.resolveLiteral()
	if !ok {
		return
	}

	if len(v.lit.Elts) == 0 && v.checkEmptyAllowed(lit.strct) {
		return
	}

	if pos, msg := v.checkLiteral(lit); pos != nil {
		v.pass.Reportf(*pos, "%s", msg)
	}
}

// resolveLiteral extracts struct type information from the composite literal,
// retrieves cached metadata, and looks up directives. Returns ok=false if the
// literal is not a struct type.
func (v visitor) resolveLiteral() (lit literal, ok bool) {
	typeName, strct, pos := v.resolveLiteralType()
	if strct == nil {
		return literal{}, false //nolint:exhaustruct // ok=false signals not found
	}

	s, diags := v.a.processor.ResolveStruct(v.pass.Fset, typeName, strct, pos, v.pass.Pkg)

	for _, diag := range diags {
		v.pass.Report(diag)
	}

	if s == nil {
		return literal{}, false //nolint:exhaustruct // ok=false signals not found
	}

	// Look up directives at literal position
	litPos := v.pass.Fset.Position(v.lit.Pos())
	dirs, dirDiags := v.a.directives.Lookup(v.pass.Fset, litPos)

	for _, d := range dirDiags {
		v.pass.Report(d)
	}

	return literal{
		strct:    s,
		typeName: s.Name,
		ignored:  dirs.Contains(directive.Ignore),
		enforced: dirs.Contains(directive.Enforce),
	}, true
}

func (v visitor) checkEmptyAllowed(s *structure.Struct) bool {
	if v.a.config.AllowEmpty {
		return true
	}

	if s.AllowEmptyDecl {
		return true
	}

	if ret, ok := v.getParentReturnStmt(); ok {
		if v.a.config.AllowEmptyReturns {
			return true
		}

		if v.isErrorReturnStatement(ret) {
			return true
		}
	}

	if v.isChildOfVariableDeclaration() && v.a.config.AllowEmptyDeclarations {
		return true
	}

	return false
}

func (v visitor) checkLiteral(lit literal) (*token.Pos, string) {
	if !lit.shouldCheck(v.a.config.ExplicitMode) {
		return nil, ""
	}

	s := lit.strct

	f := s.SkippedFields(v.lit, v.pass.Pkg.Path())

	if len(f) == 0 {
		return nil, ""
	}

	pos := v.lit.Pos()

	// Use typeName from type resolution, not cached Struct.name.
	// Derived types share the same underlying struct but have different names.
	displayName := s.PackageName + "." + lit.typeName
	if v.a.config.ReportFullTypePath {
		displayName = s.FullPath[:len(s.FullPath)-len(s.Name)] + lit.typeName
	}

	if len(f) == 1 {
		return &pos, fmt.Sprintf("%s is missing field %s", displayName, structure.FormatFieldNames(f))
	}

	return &pos, fmt.Sprintf("%s is missing fields %s", displayName, structure.FormatFieldNames(f))
}

// isChildOfVariableDeclaration checks if the node is direct part of variable
// declaration, meaning that it is a first-level RHS child of `:=` or `var`.
func (v visitor) isChildOfVariableDeclaration() bool {
	if len(v.stack) < 2 { //nolint:mnd
		return false
	}

	for i := len(v.stack) - 1; i > 0; i-- {
		parent := v.stack[i-1]

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

// getParentReturnStmt checks if the direct parent of the current node is a
// return statement and returns it if so.
func (v visitor) getParentReturnStmt() (*ast.ReturnStmt, bool) {
	if len(v.stack) < 2 { //nolint:mnd
		return nil, false
	}

	for i := len(v.stack) - 1; i > 0; i-- {
		parent := v.stack[i-1]

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

// errorIface is an interface type of the [error] interface.
//
//nolint:forcetypeassert,gochecknoglobals
var errorIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// isErrorReturnStatement checks if the return statement is an error return
// statement, meaning that it contains a non-nil value that implements [error].
func (v visitor) isErrorReturnStatement(n *ast.ReturnStmt) bool {
	if len(n.Results) == 0 {
		return false
	}

	for i := len(n.Results) - 1; i >= 0; i-- {
		ri := n.Results[i]

		if ri == v.lit {
			continue
		}

		switch ri := ri.(type) {
		case *ast.Ident:
			if ri.Name == "nil" {
				continue
			}

		case *ast.UnaryExpr:
			if ri.X == v.lit {
				continue
			}
		}

		resultType := v.pass.TypesInfo.TypeOf(ri)
		if resultType != nil && types.Implements(resultType, errorIface) {
			return true
		}
	}

	return false
}

func (a *analyzer) printCacheStats(pkgPath string) {
	siHits, siMisses, siSize := a.processor.Stats()
	printCacheLine(pkgPath, "struct-infos", siHits, siMisses, siSize)

	fdHits, fdMisses, fdSize := a.directives.Stats()
	printCacheLine(pkgPath, "file-directives", fdHits, fdMisses, fdSize)

	printMemStats(pkgPath)
}

func printMemStats(pkgPath string) {
	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	const mb = 1024 * 1024

	_, _ = fmt.Fprintf(os.Stderr, "[%s] memory: alloc=%dMB sys=%dMB heap=%dMB\n",
		pkgPath, m.Alloc/mb, m.Sys/mb, m.HeapAlloc/mb)
}

func printCacheLine(pkgPath, name string, hits, misses, size uint64) {
	hitRate := float64(0)
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) / float64(total) * 100 //nolint:mnd
	}

	_, _ = fmt.Fprintf(os.Stderr, "[%s] cache: %s: hits=%d misses=%d size=%d (%.2f%%)\n",
		pkgPath, name, hits, misses, size, hitRate)
}
