package analyzer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer"
)

func BenchmarkAnalyzer(b *testing.B) {
	a, err := analyzer.NewAnalyzer(
		[]string{`testdata/.*[Tt]est.*`, `testdata/.*External`, `testdata/.*Embedded`,
			`testdata/.*\.<anonymous>`},
		[]string{`testdata/.*Excluded$`, `testdata/e\.<anonymous>`},
	)
	require.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analysistest.Run(b, testdataPath, a, "./...")
	}
}
