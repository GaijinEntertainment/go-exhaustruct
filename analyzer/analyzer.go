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

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/fields"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/pattern"
)

type analyzer struct {
	include pattern.List
	exclude pattern.List

	fieldsCache   map[types.Type]fields.StructFields
	fieldsCacheMu sync.RWMutex

	typeProcessingNeed   map[types.Type]bool
	typeProcessingNeedMu sync.RWMutex
}

func NewAnalyzer(include, exclude []string) (an *analysis.Analyzer, err error) {
	a := analyzer{ //nolint:exhaustruct
		fieldsCache:        make(map[types.Type]fields.StructFields),
		typeProcessingNeed: make(map[types.Type]bool),
	}

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

	fs.Var(&a.include, "i", "Regular expression to match structures, can receive multiple flags")
	fs.Var(&a.exclude, "e", "Regular expression to exclude structures, can receive multiple flags")

	return *fs
}

func (a *analyzer) run(pass *analysis.Pass) (interface{}, error) {
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

		structTyp, namedTyp, ok := getStructType(pass, lit)
		if !ok {
			return true
		}

		if len(lit.Elts) == 0 {
			if ret, ok := stackParentIsReturn(stack); ok {
				if returnContainsNonNilError(pass, ret) {
					// it is okay to return uninitialized structure in case struct's direct parent is
					// a return statement containing non-nil error
					//
					// we're unable to check if returned error is custom, but at leas we're able to
					// cover str [error] type.
					return true
				}
			}
		}

		pos, msg := a.processStruct(pass, lit, structTyp, namedTyp)
		if pos != nil {
			pass.Reportf(*pos, msg)
		}

		return true
	}
}

func getStructType(pass *analysis.Pass, lit *ast.CompositeLit) (*types.Struct, *types.Named, bool) {
	switch typ := pass.TypesInfo.TypeOf(lit).(type) {
	case *types.Named: // named type
		if structTyp, ok := typ.Underlying().(*types.Struct); ok {
			return structTyp, typ, true
		}

		return nil, nil, false

	case *types.Struct: // anonymous struct
		return typ, nil, true

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
	namedTyp *types.Named,
) (*token.Pos, string) {
	if !a.shouldProcessType(namedTyp) {
		return nil, ""
	}

	// unnamed structures are only defined in same package, along with types that has
	// prefix identical to current package name.
	isSamePackage := namedTyp == nil || pass.Pkg.Scope().Lookup(namedTyp.Obj().Name()) != nil

	if f := a.litSkippedFields(lit, structTyp, !isSamePackage); len(f) > 0 {
		structName := "anonymous struct"
		if namedTyp != nil {
			structName = namedTyp.Obj().Name()
		}

		pos := lit.Pos()

		if len(f) == 1 {
			return &pos, fmt.Sprintf("%s is missing field %s", structName, f.String())
		}

		return &pos, fmt.Sprintf("%s is missing fields %s", structName, f.String())
	}

	return nil, ""
}

// shouldProcessType returns true if type should be processed basing off include
// and exclude patterns, defined though constructor and\or flags.
func (a *analyzer) shouldProcessType(typ *types.Named) bool {
	if typ == nil || (len(a.include) == 0 && len(a.exclude) == 0) {
		// anonymous structs or in case no filtering configured
		return true
	}

	a.typeProcessingNeedMu.RLock()
	res, ok := a.typeProcessingNeed[typ]
	a.typeProcessingNeedMu.RUnlock()

	if !ok {
		a.typeProcessingNeedMu.Lock()
		typStr := typ.String()
		res = true

		if a.include != nil && !a.include.MatchFullString(typStr) {
			res = false
		}

		if res && a.exclude != nil && a.exclude.MatchFullString(typStr) {
			res = false
		}

		a.typeProcessingNeed[typ] = res
		a.typeProcessingNeedMu.Unlock()
	}

	return res
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
