package analyzer

import (
	"flag"
	"go/ast"
	"go/types"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/fields"
	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/pattern"
)

type analyzer struct {
	include pattern.List `exhaustruct:"optional"`
	exclude pattern.List `exhaustruct:"optional"`

	fieldsCache   map[types.Type]fields.StructFields
	fieldsCacheMu sync.RWMutex `exhaustruct:"optional"`

	typeProcessingNeed   map[string]bool
	typeProcessingNeedMu sync.RWMutex `exhaustruct:"optional"`
}

func NewAnalyzer(include, exclude []string) (*analysis.Analyzer, error) {
	a := analyzer{
		fieldsCache:        make(map[types.Type]fields.StructFields),
		typeProcessingNeed: make(map[string]bool),
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

	visitor := &Visitor{
		analyzer: a,
		pass:     pass,
	}

	insp.WithStack(
		[]ast.Node{
			(*ast.CompositeLit)(nil),
		},
		visitor.Visit,
	)

	return nil, nil //nolint:nilnil
}
