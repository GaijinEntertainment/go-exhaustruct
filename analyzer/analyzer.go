package analyzer

import (
	"flag"
	"sync"

	"dev.gaijin.team/go/golib/e"
	"dev.gaijin.team/go/golib/fields"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"dev.gaijin.team/go/exhaustruct/v5/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v5/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v5/internal/pattern"
	"dev.gaijin.team/go/exhaustruct/v5/internal/structure"
)

// engine bundles the stateful components shared by every run of a single
// analyzer instance.
type engine struct {
	directives *directive.Scanner
	processor  *structure.Processor
}

// NewAnalyzer returns an analyzer configured exclusively through command-line
// flags, intended for CLI drivers (singlechecker, go vet -vettool). The engine
// is built lazily on the first run, after the driver has parsed the flags.
func NewAnalyzer() *analysis.Analyzer {
	config := &Config{}

	lazyEngine := sync.OnceValues(func() (engine, error) {
		return newEngine(config)
	})

	a := newBaseAnalyzer(func(pass *analysis.Pass) (any, error) {
		eng, err := lazyEngine()
		if err != nil {
			return nil, err
		}

		run(pass, config, eng)

		return nil, nil //nolint:nilnil
	})

	a.Flags.Init("", flag.PanicOnError)
	config.bindToFlagSet(&a.Flags)

	return a
}

// NewAnalyzerWithConfig returns an analyzer configured programmatically,
// intended for library consumers such as golangci-lint. The configuration is
// copied and validated immediately; it exposes no flags, and later mutations
// of the passed Config have no effect.
func NewAnalyzerWithConfig(config Config) (*analysis.Analyzer, error) {
	eng, err := newEngine(&config)
	if err != nil {
		return nil, err
	}

	return newBaseAnalyzer(func(pass *analysis.Pass) (any, error) {
		run(pass, &config, eng)

		return nil, nil //nolint:nilnil
	}), nil
}

func newBaseAnalyzer(run func(*analysis.Pass) (any, error)) *analysis.Analyzer {
	return &analysis.Analyzer{ //nolint:exhaustruct
		Name:     "exhaustruct",
		Doc:      "Checks if all structure fields are initialized",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func newEngine(config *Config) (engine, error) {
	enforce, err := pattern.NewList(config.EnforcePatterns...)
	if err != nil {
		return engine{}, e.NewFrom("compile enforce patterns", err, fields.F("flag", "enforce-rx"))
	}

	ignore, err := pattern.NewList(config.IgnorePatterns...)
	if err != nil {
		return engine{}, e.NewFrom("compile ignore patterns", err, fields.F("flag", "ignore-rx"))
	}

	optional, err := pattern.NewList(config.OptionalPatterns...)
	if err != nil {
		return engine{}, e.NewFrom("compile optional patterns", err, fields.F("flag", "optional-rx"))
	}

	allowEmpty, err := pattern.NewList(config.AllowEmptyPatterns...)
	if err != nil {
		return engine{}, e.NewFrom("compile allow-empty patterns", err, fields.F("flag", "allow-empty-rx"))
	}

	fp := astutil.NewFileParser()
	directives := directive.NewScanner(fp)

	return engine{
		directives: directives,
		processor: structure.NewProcessor(
			directives,
			structure.NewOriginScanner(fp),
			structure.WithEnforce(enforce),
			structure.WithIgnore(ignore),
			structure.WithOptional(optional),
			structure.WithAllowEmpty(allowEmpty),
		),
	}, nil
}

func run(pass *analysis.Pass, config *Config, eng engine) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector) //nolint:forcetypeassert

	for _, diag := range eng.directives.ProcessFiles(pass.Fset, pass.Files...) {
		pass.Report(diag)
	}

	newMissingFieldsVisitor(pass, insp, config, eng.directives, eng.processor).run()
	runTagMigration(pass, insp)
}
