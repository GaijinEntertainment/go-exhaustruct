package analyzer

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"dev.gaijin.team/go/exhaustruct/v5/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v5/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v5/internal/structure"
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
