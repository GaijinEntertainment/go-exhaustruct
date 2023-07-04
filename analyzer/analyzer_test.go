package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer"
)

var testdataPath, _ = filepath.Abs("./testdata/") //nolint:gochecknoglobals

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	t.Run("invalid patterns", func(t *testing.T) {
		a, err := analyzer.NewAnalyzer([]string{""}, nil, false)
		assert.Nil(t, a)
		assert.Error(t, err)

		a, err = analyzer.NewAnalyzer([]string{"["}, nil, false)
		assert.Nil(t, a)
		assert.Error(t, err)

		a, err = analyzer.NewAnalyzer(nil, []string{""}, false)
		assert.Nil(t, a)
		assert.Error(t, err)

		a, err = analyzer.NewAnalyzer(nil, []string{"["}, false)
		assert.Nil(t, a)
		assert.Error(t, err)
	})

	t.Run("basic test", func(t *testing.T) {
		a, err := analyzer.NewAnalyzer(
			[]string{`.*[Tt]est.*`, `.*External`, `.*Embedded`},
			[]string{`.*Excluded$`},
			false,
		)
		require.NoError(t, err)
		analysistest.Run(t, testdataPath, a, "i", "e")
	})

	t.Run("filter anon", func(t *testing.T) {
		a, err := analyzer.NewAnalyzer(
			nil,
			[]string{`ignore_anon\.<anonymous>`},
			true,
		)
		require.NoError(t, err)

		analysistest.Run(t, testdataPath, a, "ignore_anon", "match_anon")
	})
}
