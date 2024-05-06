package analyzer

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/comment"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/pattern"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/structure"
)

type analyzer struct {
	include pattern.List `exhaustruct:"optional"`
	exclude pattern.List `exhaustruct:"optional"`

	structFields structure.FieldsCache `exhaustruct:"optional"`
	comments     comment.Cache         `exhaustruct:"optional"`

	typeProcessingNeed   map[string]bool
	typeProcessingNeedMu sync.RWMutex `exhaustruct:"optional"`
}

func NewAnalyzer(include, exclude []string) (*analysis.Analyzer, error) {
	a := analyzer{
		typeProcessingNeed: make(map[string]bool),
		comments:           comment.Cache{},
	}

	var err error

	a.include, err = pattern.NewList(include...)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	a.exclude, err = pattern.NewList(exclude...)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return &analysis.Analyzer{ //nolint:exhaustruct
		Name:     "exhaustruct",
		Doc:      "Checks if all structure fields are initialized",
		Run:      a.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Flags:    a.newFlagSet(),
	}, nil
}

func (a *analyzer) newFlagSet() flag.FlagSet {
	fs := flag.NewFlagSet("", flag.PanicOnError)

	fs.Var(&a.include, "i", `Regular expression to match type names, can receive multiple flags.
Anonymous structs can be matched by '<anonymous>' alias.
4ex: 
	github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer\.<anonymous>
	github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer\.TypeInfo`)
	fs.Var(&a.exclude, "e", `Regular expression to exclude type names, can receive multiple flags.
Anonymous structs can be matched by '<anonymous>' alias.
4ex: 
	github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer\.<anonymous>
	github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer\.TypeInfo`)

	return *fs
}

func (a *analyzer) run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector) //nolint:forcetypeassert

	insp.WithStack([]ast.Node{(*ast.CompositeLit)(nil)}, a.newVisitor(pass))

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

		if len(lit.Elts) == 0 {
			if litIsInUnhappyPathReturn(pass, stack, lit) {
				return true
			}
		}

		file := a.comments.Get(pass.Fset, stack[0].(*ast.File)) //nolint:forcetypeassert
		rc := getCompositeLitRelatedComments(stack, file)
		pos, msg := a.processStruct(pass, lit, structTyp, typeInfo, rc)

		if pos != nil {
			pass.Reportf(*pos, msg)
		}

		return true
	}
}

// litIsInUnhappyPathReturn reports whenever the "lit" is located in the
// return statement with etiher a non-nil value of [error] interface type or a
// struct which corresponding result type in the function declaration is the [error]
// interface type.
func litIsInUnhappyPathReturn(pass *analysis.Pass, stack []ast.Node, lit *ast.CompositeLit) bool {
	ret, ok := stackParentIsReturn(stack)
	if !ok {
		return false
	}

	if containsNonNilValOfErrType(pass, ret) {
		return true
	}

	if errLit, ok := containsNonNilValUnderErrType(pass, stack, ret); ok {
		if errLit != lit {
			// we want to process composite literals of custom error types as well.
			return true
		}
	}

	return false
}

// getCompositeLitRelatedComments returns all comments that are related to checked node. We
// have to traverse the stack manually as ast do not associate comments with
// [ast.CompositeLit].
func getCompositeLitRelatedComments(stack []ast.Node, cm ast.CommentMap) []*ast.CommentGroup {
	comments := make([]*ast.CommentGroup, 0)

	for i := len(stack) - 1; i >= 0; i-- {
		node := stack[i]

		switch node.(type) {
		case *ast.CompositeLit, // stack[len(stack)-1]
			*ast.ReturnStmt, // return ...
			*ast.IndexExpr,  // map[enum]...{...}[key]
			*ast.CallExpr,   // myfunc(map...)
			*ast.UnaryExpr,  // &map...
			*ast.AssignStmt, // variable assignment (without var keyword)
			*ast.DeclStmt,   // var declaration, parent of *ast.GenDecl
			*ast.GenDecl,    // var declaration, parent of *ast.ValueSpec
			*ast.ValueSpec:  // var declaration
			comments = append(comments, cm[node]...)

		default:
			return comments
		}
	}

	return comments
}

func getStructType(pass *analysis.Pass, lit *ast.CompositeLit) (*types.Struct, *TypeInfo, bool) {
	switch typ := pass.TypesInfo.TypeOf(lit).(type) {
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

func stackParentIsReturn(stack []ast.Node) (*ast.ReturnStmt, bool) {
	// it is safe to skip boundary check, since stack always has at least one element
	// - whole file.
	ret, ok := stack[len(stack)-2].(*ast.ReturnStmt)

	return ret, ok
}

func typeName(pass *analysis.Pass, e ast.Expr) string {
	return pass.TypesInfo.TypeOf(e).String()
}

// containsNonNilValOfErrType reports if "ret" contains value of type [error].
func containsNonNilValOfErrType(pass *analysis.Pass, ret *ast.ReturnStmt) bool {
	// errors are mostly located at the end of return statement, so we're starting
	// from the end.
	for i := len(ret.Results) - 1; i >= 0; i-- {
		expr := ret.Results[i]

		if typeName(pass, expr) == "error" {
			return true
		}
	}

	return false
}

// stackNearestFuncDecl returns nearest [ast.FuncDecl] on the stack or nil if
// there is none.
func stackNearestFuncDecl(stack []ast.Node) *ast.FuncDecl {
	for i := len(stack) - 1; i >= 0; i-- {
		n := stack[i]
		if fd, ok := n.(*ast.FuncDecl); ok {
			return fd
		}
	}

	return nil
}

// containsNonNilValUnderErrType returns expr from the "ret" which
// corresponding type in nearest function declaration is [error].
func containsNonNilValUnderErrType(pass *analysis.Pass, stack []ast.Node, ret *ast.ReturnStmt) (ast.Expr, bool) {
	// errors are mostly located at the end of return statement, so we're starting
	// from the end.
	for i := len(ret.Results) - 1; i >= 0; i-- {
		expr := ret.Results[i]
		tname := typeName(pass, expr)

		if tname == "untyped nil" {
			continue
		}

		fd := stackNearestFuncDecl(stack)
		if fd == nil {
			// Only possible in case of a bad expression, because we have a return
			// statement without corresponding function declaration.
			return nil, false
		}

		outTypes := fd.Type.Results.List
		if len(outTypes) <= i {
			// Only possible in case of a bad expression, because the number of
			// arguments in the return statement does not match the number of
			// arguments in the corresponding function declaration.
			return nil, false
		}

		if typeName(pass, outTypes[i].Type) == "error" {
			// expr is returned under the position of the [error] interface type. If
			// expr type doesn't actually implement the [error], then the Go
			// compiler will throw [InvalidIFaceAssign], so we should only care
			// about the fact that expr is intended to be returned as [error].
			//
			// See: https://pkg.go.dev/golang.org/x/tools/internal/typesinternal#InvalidIfaceAssign
			return expr, true
		}
	}

	return nil, false
}

func (a *analyzer) processStruct(
	pass *analysis.Pass,
	lit *ast.CompositeLit,
	structTyp *types.Struct,
	info *TypeInfo,
	comments []*ast.CommentGroup,
) (*token.Pos, string) {
	shouldProcess := a.shouldProcessType(info)

	if shouldProcess && comment.HasDirective(comments, comment.DirectiveIgnore) {
		return nil, ""
	}

	if !shouldProcess && !comment.HasDirective(comments, comment.DirectiveEnforce) {
		return nil, ""
	}

	// unnamed structures are only defined in same package, along with types that has
	// prefix identical to current package name.
	isSamePackage := info.PackagePath == pass.Pkg.Path()

	if f := a.litSkippedFields(lit, structTyp, !isSamePackage); len(f) > 0 {
		pos := lit.Pos()

		if len(f) == 1 {
			return &pos, fmt.Sprintf("%s is missing field %s", info.ShortString(), f.String())
		}

		return &pos, fmt.Sprintf("%s is missing fields %s", info.ShortString(), f.String())
	}

	return nil, ""
}

// shouldProcessType returns true if type should be processed basing off include
// and exclude patterns, defined though constructor and\or flags.
func (a *analyzer) shouldProcessType(info *TypeInfo) bool {
	if len(a.include) == 0 && len(a.exclude) == 0 {
		return true
	}

	name := info.String()

	a.typeProcessingNeedMu.RLock()
	res, ok := a.typeProcessingNeed[name]
	a.typeProcessingNeedMu.RUnlock()

	if !ok {
		a.typeProcessingNeedMu.Lock()
		res = true

		if a.include != nil && !a.include.MatchFullString(name) {
			res = false
		}

		if res && a.exclude != nil && a.exclude.MatchFullString(name) {
			res = false
		}

		a.typeProcessingNeed[name] = res
		a.typeProcessingNeedMu.Unlock()
	}

	return res
}

func (a *analyzer) litSkippedFields(
	lit *ast.CompositeLit,
	typ *types.Struct,
	onlyExported bool,
) structure.Fields {
	return a.structFields.Get(typ).Skipped(lit, onlyExported)
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
