package analyzer_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/GaijinEntertainment/go-exhaustruct/v2/pkg/analyzer"
)

var testdataPath, _ = filepath.Abs("../../testdata")

func TestAll(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzer(
		[]string{".*\\.Test.*", ".*\\.Test2", ".*\\.Embedded", ".*\\.External"},
		[]string{".*Excluded$"},
	)
	if err != nil {
		t.Error(err)
	}

	analysistest.Run(t, testdataPath, a, "s")
}

func BenchmarkAll(b *testing.B) {
	a, err := analyzer.NewAnalyzer(
		[]string{".*\\.Test.*", ".*\\.Test2", ".*\\.Embedded", ".*\\.External"},
		[]string{".*Excluded$"},
	)
	if err != nil {
		b.Error(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		analysistest.Run(b, testdataPath, a, "s")
	}
}
