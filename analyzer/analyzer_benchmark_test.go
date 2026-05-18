package analyzer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"dev.gaijin.team/go/exhaustruct/v5/analyzer"
)

func BenchmarkAnalyzer(b *testing.B) {
	a, err := analyzer.NewAnalyzer(analyzer.Config{
		EnforcePatterns: newList(b, `.*[Tt]est.*`, `.*External`, `.*Embedded`, `.*\.<anonymous>`),
		IgnorePatterns:  newList(b, `.*Excluded$`, `e\.<anonymous>`),
	})
	require.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = analysistest.Run(b, testdataPath, a, "i")
	}
}
