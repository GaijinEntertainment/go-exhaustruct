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

	// Test excluded package behavior
	a, err = analyzer.NewAnalyzer(analyzer.Config{
		IncludeRx: []string{`.*\.TestExcluded`, `.*\.<anonymous>`},
		ExcludeRx: []string{`.*Excluded$`, `testdata/config/excluded\.<anonymous>`},
	})
	require.NoError(t, err)

	analysistest.Run(t, testdataPath, a, "testdata/config/excluded")
}

func TestAnalyzerReportFullTypePath(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzer(analyzer.Config{
		ReportFullTypePath: true,
	})
	require.NoError(t, err)

	analysistest.Run(t, testdataPath, a, "testdata/config/report_full_path")
}

func TestAnalyzerTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      analyzer.Config
		testPackage string
	}{
		{
			name: "basic",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.Test`},
			},
			testPackage: "testdata/types/basic",
		},
		{
			name: "aliases",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.(Base|Alias|Simple).*`},
				ExcludeRx: []string{`.*Excluded.*`},
			},
			testPackage: "testdata/types/aliases",
		},
		{
			name: "derived",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.(Base|Derived|External|Simple).*`},
				ExcludeRx: []string{`.*Excluded.*`},
			},
			testPackage: "testdata/types/derived",
		},
		{
			name: "embedded",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.(Embedded|TestEmbedded|Simple).*`},
			},
			testPackage: "testdata/types/embedded",
		},
		{
			name: "generics",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.testGenericStruct`},
			},
			testPackage: "testdata/types/generics",
		},
		{
			name: "collections",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.Test`},
			},
			testPackage: "testdata/types/collections",
		},
		{
			name: "anonymous",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.<anonymous>`},
			},
			testPackage: "testdata/types/anonymous",
		},
		{
			name: "directives",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.(Test|Embedded|Simple).*`},
				ExcludeRx: []string{`.*Excluded.*`},
			},
			testPackage: "testdata/types/directives",
		},
		{
			name: "filtering",
			config: analyzer.Config{
				IncludeRx: []string{`.*\.Test.*`},
				ExcludeRx: []string{`.*Excluded.*`},
			},
			testPackage: "testdata/types/filtering",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := analyzer.NewAnalyzer(tt.config)
			require.NoError(t, err)

			analysistest.Run(t, testdataPath, a, tt.testPackage)
		})
	}
}
