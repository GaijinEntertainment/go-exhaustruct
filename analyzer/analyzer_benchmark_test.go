package analyzer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer"
)

func BenchmarkAnalyzer(b *testing.B) {
	a, err := analyzer.NewAnalyzer(
		[]string{`.*[Tt]est.*`, `.*External`, `.*Embedded`, `.*\.<anonymous>`},
		[]string{`.*Excluded$`, `e\.<anonymous>`},
	)
	require.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = analysistest.Run(b, testdataPath, a, "i")
	}
}
