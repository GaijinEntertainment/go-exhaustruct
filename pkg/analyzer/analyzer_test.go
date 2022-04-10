package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/GaijinEntertainment/go-exhaustruct/pkg/analyzer"
)

func TestAll(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %s", err)
	}

	testdata := filepath.Join(filepath.Dir(filepath.Dir(wd)), "testdata")

	a := analyzer.MustNewAnalyzer(
		[]string{".*\\.Test", ".*\\.Test2", ".*\\.Embedded", ".*\\.External"},
		[]string{".*Excluded$"},
	)

	analysistest.Run(t, testdata, a, "s")
}

func BenchmarkAll(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get wd: %s", err)
	}

	testdata := filepath.Join(filepath.Dir(filepath.Dir(wd)), "testdata")

	a := analyzer.MustNewAnalyzer(
		[]string{".*\\.Test", ".*\\.Test2", ".*\\.Embedded", ".*\\.External"},
		[]string{".*Excluded$"},
	)

	for i := 0; i < b.N; i++ {
		analysistest.Run(b, testdata, a, "s")
	}
}
