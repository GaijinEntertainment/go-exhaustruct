package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"dev.gaijin.team/go/exhaustruct/v4/analyzer"
)

var testdataPath, _ = filepath.Abs("./testdata/") //nolint:gochecknoglobals

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzer(analyzer.Config{IncludeRx: []string{""}})
	assert.Nil(t, a)
	assert.Error(t, err)

	a, err = analyzer.NewAnalyzer(analyzer.Config{IncludeRx: []string{"["}})
	assert.Nil(t, a)
	assert.Error(t, err)

	a, err = analyzer.NewAnalyzer(analyzer.Config{ExcludeRx: []string{""}})
	assert.Nil(t, a)
	assert.Error(t, err)

	a, err = analyzer.NewAnalyzer(analyzer.Config{ExcludeRx: []string{"["}})
	assert.Nil(t, a)
	assert.Error(t, err)

	a, err = analyzer.NewAnalyzer(analyzer.Config{
		IncludeRx: []string{`.*[Tt]est.*`, `.*External`, `.*Embedded`, `.*\.<anonymous>`},
		ExcludeRx: []string{`.*Excluded$`, `e\.<anonymous>`},
	})
	require.NoError(t, err)

	analysistest.Run(t, testdataPath, a, "i", "e")
}
