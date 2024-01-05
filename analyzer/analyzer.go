package analyzer

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/comment"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/file"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/pattern"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/structure"
)

type analyzer struct {
	include pattern.List `exhaustruct:"optional"`
	exclude pattern.List `exhaustruct:"optional"`

	structFields *structure.FieldsCache `exhaustruct:"optional"`
	comments     *comment.Cache         `exhaustruct:"optional"`

	astCache *file.ASTCache `exhaustruct:"optional"`

	typeProcessingNeed   map[string]bool
	typeProcessingNeedMu sync.RWMutex `exhaustruct:"optional"`
}

func NewAnalyzer(include, exclude []string) (*analysis.Analyzer, error) {
	a := analyzer{
		typeProcessingNeed: make(map[string]bool),
		astCache:           &file.ASTCache{Mode: parser.ParseComments},
		structFields:       &structure.FieldsCache{},
		comments:           &comment.Cache{},
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

	a.astCache.AddFiles(pass.Fset, pass.Files...)

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
			// should never happen, but better be prepared
			return true
		}

		structInfo, err := structure.GetInfo(pass.TypesInfo.TypeOf(lit), pass.Fset, a.astCache)
		if err != nil {
			panic(err)
		}

		if !structInfo.IsValid() {
			// not a structure - skipping
			return true
		}

		if len(lit.Elts) == 0 {
			if ret, ok := stackParentIsReturn(stack); ok {
				if returnContainsNonNilError(pass, ret) {
					// it is okay to return uninitialized structure in case struct's direct parent is
					// a return statement containing non-nil error
					//
					// we're unable to check if returned error is custom, but at least we're able to
					// cover std error type.
					return true
				}
			}
		}

		commentsMap := a.comments.Get(pass.Fset, stack[0].(*ast.File)) //nolint:forcetypeassert
		rc := getCompositeLitRelatedComments(stack, commentsMap)
		pos, msg := a.processStruct(pass, lit, structInfo, rc)

		if pos != nil {
			pass.Reportf(*pos, msg)
		}

		return true
	}
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

func stackParentIsReturn(stack []ast.Node) (*ast.ReturnStmt, bool) {
	// it is safe to skip boundary check, since stack always has at least one element
	// - whole file.
	ret, ok := stack[len(stack)-2].(*ast.ReturnStmt)

	return ret, ok
}

func returnContainsNonNilError(pass *analysis.Pass, ret *ast.ReturnStmt) bool {
	// errors are mostly located at the end of return statement, so we're starting
	// from the end.
	for i := len(ret.Results) - 1; i >= 0; i-- {
		if pass.TypesInfo.TypeOf(ret.Results[i]).String() == "error" {
			return true
		}
	}

	return false
}

func (a *analyzer) processStruct(
	pass *analysis.Pass,
	lit *ast.CompositeLit,
	info structure.Info,
	comments []*ast.CommentGroup,
) (*token.Pos, string) {
	shouldProcess := a.shouldProcessType(info.String())

	if shouldProcess && comment.HasDirective(comments, comment.DirectiveIgnore) {
		return nil, ""
	}

	if !shouldProcess && !comment.HasDirective(comments, comment.DirectiveEnforce) {
		return nil, ""
	}

	// unnamed structures are only defined in same package, along with types that has
	// prefix identical to current package name.
	isSamePackage := info.PackagePath == pass.Pkg.Path()

	if f := a.litSkippedFields(lit, info.Type, !isSamePackage); len(f) > 0 {
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
func (a *analyzer) shouldProcessType(name string) bool {
	if len(a.include) == 0 && len(a.exclude) == 0 {
		return true
	}

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
