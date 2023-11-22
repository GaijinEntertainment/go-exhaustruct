package analyzer

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/fields"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/pattern"
)

type analyzer struct {
	include       pattern.List `exhaustruct:"optional"`
	exclude       pattern.List `exhaustruct:"optional"`
	useDirectives bool

	fieldsCache   map[types.Type]fields.StructFields
	fieldsCacheMu sync.RWMutex `exhaustruct:"optional"`

	typeProcessingNeed   map[string]bool
	typeProcessingNeedMu sync.RWMutex `exhaustruct:"optional"`

	commentMapCache   map[*ast.File]ast.CommentMap `exhaustruct:"optional"`
	commentMapCacheMu sync.RWMutex                 `exhaustruct:"optional"`
}

func NewAnalyzer(include, exclude []string, useDirectives bool) (*analysis.Analyzer, error) {
	a := analyzer{
		fieldsCache:        make(map[types.Type]fields.StructFields),
		typeProcessingNeed: make(map[string]bool),
		commentMapCache:    make(map[*ast.File]ast.CommentMap),
		useDirectives:      useDirectives,
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
	fs.BoolVar(&a.useDirectives, "use-directives", a.useDirectives,
		`Use directives to enforce or ignore analysis on a per struct literal basis, overriding
any include/exclude patterns. Default: false.`)

	return *fs
}

func (a *analyzer) run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector) //nolint:forcetypeassert

	insp.WithStack(
		[]ast.Node{
			(*ast.CompositeLit)(nil),
		},
		a.newVisitor(pass),
	)

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
			if ret, ok := stackParentIsReturn(stack); ok {
				if returnContainsNonNilError(pass, ret) {
					// it is okay to return uninitialized structure in case struct's direct parent is
					// a return statement containing non-nil error
					//
					// we're unable to check if returned error is custom, but at least we're able to
					// cover str [error] type.
					return true
				}
			}
		}

		var enforcement EnforcementDirective
		if a.useDirectives {
			enforcement = a.decideEnforcementDirective(pass, lit, stack)
		}

		pos, msg := a.processStruct(pass, lit, structTyp, typeInfo, enforcement)
		if pos != nil {
			pass.Reportf(*pos, msg)
		}

		return true
	}
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
	structTyp *types.Struct,
	info *TypeInfo,
	enforcement EnforcementDirective,
) (*token.Pos, string) {
	if !a.shouldProcessLit(info, enforcement) {
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

// shouldProcessLit returns true if type should be processed basing off include
// and exclude patterns, defined though constructor and\or flags, as well as off
// comment directives.
func (a *analyzer) shouldProcessLit(
	info *TypeInfo, enforcement EnforcementDirective,
) bool {
	// enforcement directives always have highest precedence if present
	switch enforcement {
	case Enforce:
		return true

	case Ignore:
		return false

	case EnforcementUnspecified:
	}

	if len(a.include) == 0 && len(a.exclude) == 0 {
		return true
	}

	return a.isTypeProcessingNeeded(info)
}

//revive:disable-next-line:unused-receiver
func (a *analyzer) litSkippedFields(
	lit *ast.CompositeLit,
	typ *types.Struct,
	onlyExported bool,
) fields.StructFields {
	a.fieldsCacheMu.RLock()
	f, ok := a.fieldsCache[typ]
	a.fieldsCacheMu.RUnlock()

	if !ok {
		a.fieldsCacheMu.Lock()
		f = fields.NewStructFields(typ)
		a.fieldsCache[typ] = f
		a.fieldsCacheMu.Unlock()
	}

	return f.SkippedFields(lit, onlyExported)
}

func (a *analyzer) decideEnforcementDirective(
	pass *analysis.Pass, lit *ast.CompositeLit, stack []ast.Node,
) EnforcementDirective {
	if !a.useDirectives {
		return EnforcementUnspecified
	}

	//revive:disable-next-line:unchecked-type-assertion
	file, _ := stack[0].(*ast.File)
	commentMap := a.getFileCommentMap(pass.Fset, file)

	if enforcement := parseEnforcement(commentMap[lit]); enforcement != EnforcementUnspecified {
		return enforcement
	}

	parent := stack[len(stack)-2]
	// allow directives to appear in parent nodes except other composite literals
	if _, parentIsCompLit := parent.(*ast.CompositeLit); parentIsCompLit {
		return EnforcementUnspecified
	}

	return parseEnforcement(commentMap[parent])
}

func (a *analyzer) getFileCommentMap(fileSet *token.FileSet, file *ast.File) ast.CommentMap {
	a.commentMapCacheMu.RLock()
	commentMap, exists := a.commentMapCache[file]
	a.commentMapCacheMu.RUnlock()

	if !exists {
		// TODO: consider avoiding risk of double-computation by using per-file mutex
		commentMap = ast.NewCommentMap(fileSet, file, file.Comments)

		a.commentMapCacheMu.Lock()
		a.commentMapCache[file] = commentMap
		a.commentMapCacheMu.Unlock()
	}

	return commentMap
}

func (a *analyzer) isTypeProcessingNeeded(info *TypeInfo) bool {
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

func parseEnforcement(commentGroups []*ast.CommentGroup) EnforcementDirective {
	// go from the end to the beginning
	for i := len(commentGroups) - 1; i >= 0; i-- {
		for j := len(commentGroups[i].List) - 1; j >= 0; j-- {
			c := commentGroups[i].List[j]

			normalized := strings.TrimSpace(c.Text)
			switch normalized {
			case "//exhaustruct:enforce":
				return Enforce

			case "//exhaustruct:ignore":
				return Ignore
			}
		}
	}

	return EnforcementUnspecified
}

type EnforcementDirective int

const (
	EnforcementUnspecified EnforcementDirective = iota
	Enforce
	Ignore
)

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
