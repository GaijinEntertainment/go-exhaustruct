package analyzer

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"runtime"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"dev.gaijin.team/go/exhaustruct/v4/internal/cache"
	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

type analyzer struct {
	config Config

	structCache        *structure.Cache           `exhaustruct:"optional"`
	fileDirectives     *directive.FileCache       `exhaustruct:"optional"`
	typeProcessingNeed *cache.Cache[string, bool] `exhaustruct:"optional"`
}

const typeProcessingCacheSize = 64

func NewAnalyzer(config Config) (*analysis.Analyzer, error) {
	err := config.Prepare()
	if err != nil {
		return nil, err
	}

	a := analyzer{
		config:             config,
		structCache:        structure.NewCache(),
		fileDirectives:     directive.NewFileCache(&defaultParser{}),
		typeProcessingNeed: cache.New[string, bool](typeProcessingCacheSize),
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
	for _, diag := range a.fileDirectives.Add(pass.Fset, pass.Files...) {
		pass.Report(diag)
	}

	insp.WithStack([]ast.Node{(*ast.CompositeLit)(nil)}, a.newVisitor(pass))

	if a.config.DebugCacheMetrics {
		a.printCacheStats(pass.Pkg.Path())
	}

	return nil, nil //nolint:nilnil
}

// newVisitor returns visitor that only expects [ast.CompositeLit] nodes.
func (a *analyzer) newVisitor(pass *analysis.Pass) func(n ast.Node, push bool, stack []ast.Node) bool {
	return func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			// this should never happen, but better be prepared
			return true
		}

		strct, pkg, typePos, ti, ok := getStructType(pass, lit)
		if !ok {
			return true
		}

		if len(lit.Elts) == 0 && a.checkEmptyStructAllowed(pass, stack, ti) {
			return true
		}

		litPos := pass.Fset.Position(lit.Pos())
		litDirs, _ := a.fileDirectives.Lookup(pass.Fset, litPos)

		pos, msg := a.processStruct(pass, lit, strct, pkg, typePos, ti, litDirs)

		if pos != nil {
			pass.Reportf(*pos, "%s", msg)
		}

		return true
	}
}

func (a *analyzer) checkEmptyStructAllowed(pass *analysis.Pass, stack []ast.Node, typeInfo *TypeInfo) bool {
	// empty structs are globally allowed
	if a.config.AllowEmpty {
		return true
	}

	// some structs are allowed to be empty, basing on pattern
	if a.config.allowEmptyPatterns.MatchFullString(typeInfo.String()) {
		return true
	}

	if ret, ok := getParentReturnStmt(stack); ok {
		// empty structures are allowed in all return statements
		if a.config.AllowEmptyReturns {
			return true
		}

		// empty structures are allowed in error returns
		if isErrorReturnStatement(pass, ret, stack[len(stack)-1]) {
			return true
		}
	}

	// empty structures are allowed in variable declarations
	if isChildOfVariableDeclaration(stack) && a.config.AllowEmptyDeclarations {
		return true
	}

	return false
}

// isPartOfVariableDeclaration checks if the node is direct part of variable
// declaration, meaning that it is a first-level RHS child of `:=` or `var`
// declaration.
func isChildOfVariableDeclaration(stack []ast.Node) bool {
	if len(stack) < 2 { //nolint:mnd // stack for sure contains at leas current node and its parent (file)
		return false
	}

	// Start from composite literal and go up the stack
	for i := len(stack) - 1; i > 0; i-- {
		parent := stack[i-1]

		switch p := parent.(type) {
		case *ast.AssignStmt:
			if p.Tok == token.DEFINE {
				return true
			}

		case *ast.ValueSpec:
			return true

		case *ast.UnaryExpr:
			// Only allow pointer taking (&)
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
func getParentReturnStmt(stack []ast.Node) (*ast.ReturnStmt, bool) {
	if len(stack) < 2 { //nolint:mnd // stack for sure contains at leas current node and its parent (file)
		return nil, false
	}

	// Start from composite literal and go up the stack
	for i := len(stack) - 1; i > 0; i-- {
		parent := stack[i-1]

		switch p := parent.(type) {
		case *ast.ReturnStmt:
			return p, true

		case *ast.UnaryExpr:
			// Only allow pointer taking (&)
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
func isErrorReturnStatement(pass *analysis.Pass, n *ast.ReturnStmt, currentNode ast.Node) bool {
	if len(n.Results) == 0 {
		return false
	}

	// iterate backwards, since idiomatic position of error is at the end
	for i := len(n.Results) - 1; i >= 0; i-- {
		ri := n.Results[i]

		// Skip the current node, since it is already being checked
		if ri == currentNode {
			continue
		}

		switch ri := ri.(type) {
		case *ast.Ident:
			// Skip nil values
			if ri.Name == "nil" {
				continue
			}

		case *ast.UnaryExpr:
			// Current node might be under the unary expression
			if ri.X == currentNode {
				continue
			}
		}

		// Check if the type implements error interface
		resultType := pass.TypesInfo.TypeOf(ri)
		if resultType != nil && types.Implements(resultType, errorIface) {
			return true
		}
	}

	return false
}

//nolint:revive // function-result-limit: 5 results needed for type resolution context
func getStructType(pass *analysis.Pass, lit *ast.CompositeLit) (
	*types.Struct, *types.Package, token.Pos, *TypeInfo, bool,
) {
	typ := types.Unalias(pass.TypesInfo.TypeOf(lit))

	// Handle pointer types (e.g., implicit `{}` in `[]*Struct{...}`)
	// See: https://github.com/GaijinEntertainment/go-exhaustruct/issues/144
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(ptr.Elem())
	}

	switch typ := typ.(type) {
	case *types.Named: // named type
		if strct, ok := typ.Underlying().(*types.Struct); ok {
			pkg := typ.Obj().Pkg()
			ti := &TypeInfo{
				Name:        typ.Obj().Name(),
				PackageName: pkg.Name(),
				PackagePath: pkg.Path(),
			}

			return strct, pkg, typ.Obj().Pos(), ti, true
		}

		return nil, nil, token.NoPos, nil, false

	case *types.Struct: // anonymous struct
		ti := &TypeInfo{
			Name:        structure.AnonymousName,
			PackageName: pass.Pkg.Name(),
			PackagePath: pass.Pkg.Path(),
		}

		return typ, pass.Pkg, token.NoPos, ti, true

	default:
		return nil, nil, token.NoPos, nil, false
	}
}

//nolint:revive // argument-limit: 7 args needed for struct processing context
func (a *analyzer) processStruct(
	pass *analysis.Pass,
	lit *ast.CompositeLit,
	strct *types.Struct,
	pkg *types.Package,
	typePos token.Pos,
	ti *TypeInfo,
	litDirs directive.Directives,
) (*token.Pos, string) {
	// Get struct metadata with directives from cache
	s, diags := a.structCache.Get(
		pass.Fset,
		strct,
		ti.Name,
		pkg,
		typePos,
		a.fileDirectives,
	)

	for _, diag := range diags {
		pass.Report(diag)
	}

	if !a.shouldCheck(s, ti, litDirs) {
		return nil, ""
	}

	// Apply optional-rx pattern if struct-level optional is not already set
	// Priority: struct:optional (already in s) > flag:optional-rx
	if !s.Optional && a.isTypeOptionalByPattern(ti) {
		s.Optional = true
	}

	externalPkg := !structFieldsInPackage(strct, pass.Pkg)
	f := s.SkippedFields(lit, externalPkg)

	if len(f) == 0 {
		return nil, ""
	}

	pos := lit.Pos()

	typeName := ti.ShortString()
	if a.config.ReportFullTypePath {
		typeName = ti.String()
	}

	if len(f) == 1 {
		return &pos, fmt.Sprintf("%s is missing field %s", typeName, f.String())
	}

	return &pos, fmt.Sprintf("%s is missing fields %s", typeName, f.String())
}

// shouldCheck implements v5 checking decision priority.
// Priority: literal:ignore > literal:enforce > struct:ignore > struct:enforce > flag patterns.
func (a *analyzer) shouldCheck(s *structure.Struct, ti *TypeInfo, litDirs directive.Directives) bool {
	if litDirs.Contains(directive.Ignore) {
		return false
	}

	if litDirs.Contains(directive.Enforce) {
		return true
	}

	if s.Ignored {
		return false
	}

	if s.Enforced {
		return true
	}

	return a.shouldProcessType(ti)
}

// shouldProcessType returns true if type should be processed basing off enforce
// and ignore patterns, defined though constructor and\or flags.
func (a *analyzer) shouldProcessType(info *TypeInfo) bool {
	if len(a.config.enforcePatterns) == 0 && len(a.config.ignorePatterns) == 0 {
		return true
	}

	name := info.String()

	return a.typeProcessingNeed.GetOrSet(name, func() bool {
		if a.config.enforcePatterns != nil && !a.config.enforcePatterns.MatchFullString(name) {
			return false
		}

		if a.config.ignorePatterns != nil && a.config.ignorePatterns.MatchFullString(name) {
			return false
		}

		return true
	})
}

// isTypeOptionalByPattern returns true if the type matches optional-rx patterns.
func (a *analyzer) isTypeOptionalByPattern(info *TypeInfo) bool {
	if len(a.config.optionalPatterns) == 0 {
		return false
	}

	return a.config.optionalPatterns.MatchFullString(info.String())
}

// defaultParser implements directive.FileParser using go/parser.
type defaultParser struct{}

// ParseFile parses a Go source file and returns its AST with comments.
func (*defaultParser) ParseFile(fset *token.FileSet, filename string) (*ast.File, error) {
	//nolint:wrapcheck // error context is added by caller
	return parser.ParseFile(fset, filename, nil, parser.ParseComments)
}

func (a *analyzer) printCacheStats(pkgPath string) {
	siHits, siMisses, siSize := a.structCache.Stats()
	printCacheLine(pkgPath, "struct-infos", siHits, siMisses, siSize)

	fdHits, fdMisses, fdSize := a.fileDirectives.Stats()
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

// structFieldsInPackage returns true if the struct's fields are defined in the
// given package. For derived types like `type Bar foo.Foo`, returns false if
// fields are from another package.
//
// We treat structs with zero fields as defined in the package, since there
// are no fields to access.
func structFieldsInPackage(structTyp *types.Struct, pkg *types.Package) bool {
	if structTyp.NumFields() == 0 {
		return true
	}

	fieldPkg := structTyp.Field(0).Pkg()

	return fieldPkg == nil || fieldPkg == pkg
}

// TypeInfo holds display information for a struct type.
type TypeInfo struct {
	Name        string
	PackageName string
	PackagePath string
}

// String returns the full type path (e.g., "net/http.Request").
func (t TypeInfo) String() string {
	return t.PackagePath + "." + t.Name
}

// ShortString returns the short type name (e.g., "http.Request").
func (t TypeInfo) ShortString() string {
	return t.PackageName + "." + t.Name
}
