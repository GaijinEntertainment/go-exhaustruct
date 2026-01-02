package analyzer

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

type analyzer struct {
	config Config

	structFields   structure.FieldsCache `exhaustruct:"optional"`
	fileDirectives *directive.FileCache  `exhaustruct:"optional"`

	typeProcessingNeed   map[string]bool
	typeProcessingNeedMu sync.RWMutex `exhaustruct:"optional"`
}

func NewAnalyzer(config Config) (*analysis.Analyzer, error) {
	err := config.Prepare()
	if err != nil {
		return nil, err
	}

	a := analyzer{
		config:             config,
		fileDirectives:     directive.NewFileCache(&defaultParser{}),
		typeProcessingNeed: make(map[string]bool),
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

		structTyp, typeInfo, ok := getStructType(pass, lit)
		if !ok {
			return true
		}

		if len(lit.Elts) == 0 && a.checkEmptyStructAllowed(pass, stack, typeInfo) {
			return true
		}

		litPos := pass.Fset.Position(lit.Pos())
		dir, _ := a.fileDirectives.Lookup(pass.Fset, litPos)

		pos, msg := a.processStruct(pass, lit, structTyp, typeInfo, dir)

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

func getStructType(pass *analysis.Pass, lit *ast.CompositeLit) (*types.Struct, *TypeInfo, bool) {
	typ := types.Unalias(pass.TypesInfo.TypeOf(lit))

	// Handle pointer types (e.g., implicit `{}` in `[]*Struct{...}`)
	// See: https://github.com/GaijinEntertainment/go-exhaustruct/issues/144
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(ptr.Elem())
	}

	switch typ := typ.(type) {
	case *types.Named: // named type
		if structTyp, ok := typ.Underlying().(*types.Struct); ok {
			pkg := typ.Obj().Pkg()
			ti := TypeInfo{
				Name:        typ.Obj().Name(),
				PackageName: pkg.Name(),
				PackagePath: pkg.Path(),
			}

			return structTyp, &ti, true
		}

		return nil, nil, false

	case *types.Struct: // anonymous struct
		ti := TypeInfo{
			Name:        "<anonymous>",
			PackageName: pass.Pkg.Name(),
			PackagePath: pass.Pkg.Path(),
		}

		return typ, &ti, true

	default:
		return nil, nil, false
	}
}

func (a *analyzer) processStruct(
	pass *analysis.Pass,
	lit *ast.CompositeLit,
	structTyp *types.Struct,
	info *TypeInfo,
	dirs directive.Directives,
) (*token.Pos, string) {
	shouldProcess := a.shouldProcessType(info)

	if shouldProcess && dirs.Contains(directive.Ignore) {
		return nil, ""
	}

	if !shouldProcess && !dirs.Contains(directive.Enforce) {
		return nil, ""
	}

	canAccessUnexported := structFieldsInPackage(structTyp, pass.Pkg)

	if f := a.litSkippedFields(pass, lit, structTyp, !canAccessUnexported); len(f) > 0 {
		pos := lit.Pos()

		typeName := info.ShortString()
		if a.config.ReportFullTypePath {
			typeName = info.String()
		}

		if len(f) == 1 {
			return &pos, fmt.Sprintf("%s is missing field %s", typeName, f.String())
		}

		return &pos, fmt.Sprintf("%s is missing fields %s", typeName, f.String())
	}

	return nil, ""
}

// shouldProcessType returns true if type should be processed basing off include
// and exclude patterns, defined though constructor and\or flags.
func (a *analyzer) shouldProcessType(info *TypeInfo) bool {
	if len(a.config.includePatterns) == 0 && len(a.config.excludePatterns) == 0 {
		return true
	}

	name := info.String()

	a.typeProcessingNeedMu.RLock()

	res, ok := a.typeProcessingNeed[name]

	a.typeProcessingNeedMu.RUnlock()

	if !ok {
		a.typeProcessingNeedMu.Lock()

		res = true

		if a.config.includePatterns != nil && !a.config.includePatterns.MatchFullString(name) {
			res = false
		}

		if res && a.config.excludePatterns != nil && a.config.excludePatterns.MatchFullString(name) {
			res = false
		}

		a.typeProcessingNeed[name] = res
		a.typeProcessingNeedMu.Unlock()
	}

	return res
}

func (a *analyzer) litSkippedFields(
	pass *analysis.Pass,
	lit *ast.CompositeLit,
	typ *types.Struct,
	onlyExported bool,
) structure.Fields {
	lookup := a.makeDirectiveLookup(pass, typ)

	return a.structFields.Get(typ, lookup).Skipped(lit, onlyExported)
}

// directiveLookup implements structure.DirectiveLookup for field optionality checks.
type directiveLookup struct {
	fset  *token.FileSet
	cache *directive.FileCache
}

// Lookup returns the directives at the given source position.
func (d *directiveLookup) Lookup(pos token.Pos) directive.Directives {
	resolved := d.fset.Position(pos)
	dirs, _ := d.cache.Lookup(d.fset, resolved)

	return dirs
}

// makeDirectiveLookup creates a DirectiveLookup for checking field directives.
// It works for both local types (from pass.Files) and external types
// (by parsing the source file via the cache on demand).
func (a *analyzer) makeDirectiveLookup(pass *analysis.Pass, typ *types.Struct) structure.DirectiveLookup {
	if typ.NumFields() == 0 {
		return nil
	}

	firstFieldPos := typ.Field(0).Pos()
	if !firstFieldPos.IsValid() {
		return nil
	}

	return &directiveLookup{
		fset:  pass.Fset,
		cache: a.fileDirectives,
	}
}

// defaultParser implements directive.FileParser using go/parser.
type defaultParser struct{}

// ParseFile parses a Go source file and returns its AST with comments.
func (*defaultParser) ParseFile(fset *token.FileSet, filename string) (*ast.File, error) {
	//nolint:wrapcheck // error context is added by caller
	return parser.ParseFile(fset, filename, nil, parser.ParseComments)
}

func (a *analyzer) printCacheStats(pkgPath string) {
	sfHits, sfMisses := a.structFields.Stats()
	printCacheLine(pkgPath, "struct-fields", sfHits, sfMisses, 0)

	fdHits, fdMisses, fdSize := a.fileDirectives.Stats()
	printCacheLine(pkgPath, "file-directives", fdHits, fdMisses, fdSize)
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

type TypeInfo struct {
	Name        string
	PackageName string
	PackagePath string
}

func (t TypeInfo) String() string {
	return t.PackagePath + "." + t.Name
}

func (t TypeInfo) ShortString() string {
	return t.PackageName + "." + t.Name
}
