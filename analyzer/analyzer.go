package analyzer

import (
	"flag"
	"fmt"
	"os"
	"runtime"
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

type analyzer struct {
	config Config

	// init builds directives and processor on first call. NewAnalyzer defers
	// it to the first run so that flag-driven Config mutations performed by
	// the analysis driver after construction are visible; NewAnalyzerWithConfig
	// invokes it eagerly so configuration errors surface at construction time.
	init func() error

	directives *directive.Scanner   `exhaustruct:"optional"`
	processor  *structure.Processor `exhaustruct:"optional"`
}

// NewAnalyzer returns an analyzer configured exclusively through command-line
// flags, intended for CLI drivers (singlechecker, go vet -vettool). The
// configuration is consumed on the first run, after the driver has parsed
// the flags.
func NewAnalyzer() *analysis.Analyzer {
	a := &analyzer{config: Config{}} //nolint:exhaustruct

	a.init = sync.OnceValue(a.initialize)

	aa := newBaseAnalyzer(a.run)

	aa.Flags = *a.config.bindToFlagSet(flag.NewFlagSet("", flag.PanicOnError))

	return aa
}

// NewAnalyzerWithConfig returns an analyzer configured programmatically,
// intended for library consumers such as golangci-lint. The configuration is
// copied and validated immediately; it exposes no flags, and later mutations
// of the passed Config have no effect.
func NewAnalyzerWithConfig(config Config) (*analysis.Analyzer, error) {
	a := &analyzer{config: config} //nolint:exhaustruct

	a.init = sync.OnceValue(a.initialize)

	if err := a.init(); err != nil {
		return nil, err
	}

	return newBaseAnalyzer(a.run), nil
}

func newBaseAnalyzer(run func(*analysis.Pass) (any, error)) *analysis.Analyzer {
	return &analysis.Analyzer{ //nolint:exhaustruct
		Name:     "exhaustruct",
		Doc:      "Checks if all structure fields are initialized",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func (a *analyzer) initialize() error {
	fp := astutil.NewFileParser()

	a.directives = directive.NewScanner(fp)

	enforce, err := pattern.NewList(a.config.EnforcePatterns...)
	if err != nil {
		return e.NewFrom("compile enforce patterns", err, fields.F("flag", "enforce-rx"))
	}

	ignore, err := pattern.NewList(a.config.IgnorePatterns...)
	if err != nil {
		return e.NewFrom("compile ignore patterns", err, fields.F("flag", "ignore-rx"))
	}

	optional, err := pattern.NewList(a.config.OptionalPatterns...)
	if err != nil {
		return e.NewFrom("compile optional patterns", err, fields.F("flag", "optional-rx"))
	}

	allowEmpty, err := pattern.NewList(a.config.AllowEmptyPatterns...)
	if err != nil {
		return e.NewFrom("compile allow-empty patterns", err, fields.F("flag", "allow-empty-rx"))
	}

	a.processor = structure.NewProcessor(
		a.directives,
		structure.NewOriginScanner(fp),
		structure.WithEnforce(enforce),
		structure.WithIgnore(ignore),
		structure.WithOptional(optional),
		structure.WithAllowEmpty(allowEmpty),
	)

	return nil
}

func (a *analyzer) run(pass *analysis.Pass) (any, error) {
	if err := a.init(); err != nil {
		return nil, err
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector) //nolint:forcetypeassert

	for _, diag := range a.directives.ProcessFiles(pass.Fset, pass.Files...) {
		pass.Report(diag)
	}

	newMissingFieldsVisitor(a, pass, insp).run()
	newTagMigrationVisitor(pass, insp).run()

	if a.config.DebugCacheMetrics {
		pkgPath := pass.Pkg.Path()

		siHits, siMisses, siSize := a.processor.Stats()
		printCacheLine(pkgPath, "struct-infos", siHits, siMisses, siSize)

		fdHits, fdMisses, fdSize := a.directives.Stats()
		printCacheLine(pkgPath, "file-directives", fdHits, fdMisses, fdSize)

		printMemStats(pkgPath)
	}

	return nil, nil //nolint:nilnil
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
