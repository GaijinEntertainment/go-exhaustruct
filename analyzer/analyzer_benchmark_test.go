package analyzer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"dev.gaijin.team/go/exhaustruct/v5/analyzer"
)

// BenchmarkAnalyzer runs the analyzer over the fixtures covering the shapes it
// has to resolve: plain structs, embedding, types derived from another package,
// and literals nested in collections. Each runs under the configuration its
// fixture is written against, taken from typeCases, so a run reports exactly
// what the fixture expects and measures analysis rather than failure handling.
//
// The analyzer is built inside the loop. One processor caches by type identity,
// and every analysistest.Run loads the fixture afresh, so a processor kept
// across iterations accumulates entries it can never reuse and the benchmark
// stops describing a single bounded run.
func BenchmarkAnalyzer(b *testing.B) {
	for _, c := range typeCases() {
		if !c.benchmark {
			continue
		}

		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				a, err := analyzer.NewAnalyzerWithConfig(c.config)
				require.NoError(b, err)

				_ = analysistest.Run(b, testdataPath, a, c.testPackage)
			}
		})
	}
}
