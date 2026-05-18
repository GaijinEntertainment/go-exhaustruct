package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"dev.gaijin.team/go/exhaustruct/v5/analyzer"
	"dev.gaijin.team/go/exhaustruct/v5/internal/pattern"
)

var testdataPath, _ = filepath.Abs("./testdata/") //nolint:gochecknoglobals

// newList is a test helper that creates a pattern.List from strings.
func newList(tb testing.TB, patterns ...string) pattern.List {
	tb.Helper()

	list, err := pattern.NewList(patterns...)
	require.NoError(tb, err)

	return list
}

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	// Empty pattern should fail during Set()
	config := analyzer.Config{}

	err := config.EnforcePatterns.Set("")
	require.Error(t, err)

	// Invalid regex should fail during Set()
	config = analyzer.Config{}

	err = config.EnforcePatterns.Set("[")
	require.Error(t, err)

	// Test ignored package behavior
	a, err := analyzer.NewAnalyzer(analyzer.Config{
		EnforcePatterns: newList(t, `.*\.TestExcluded`, `.*\.<anonymous>`),
		IgnorePatterns:  newList(t, `.*Excluded$`, `testdata/config/excluded\.<anonymous>`),
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
		testFixes   bool
	}{
		{
			name: "basic",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.Test`),
			},
			testPackage: "testdata/types/basic",
		},
		{
			name: "aliases",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.(Base|Alias|Simple).*`),
				IgnorePatterns:  newList(t, `.*Excluded.*`),
			},
			testPackage: "testdata/types/aliases",
		},
		{
			name: "derived",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.(Base|Derived|External|Simple).*`),
				IgnorePatterns:  newList(t, `.*Excluded.*`),
			},
			testPackage: "testdata/types/derived",
		},
		{
			name: "embedded",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.(Embedded|TestEmbedded|Simple).*`),
			},
			testPackage: "testdata/types/embedded",
		},
		{
			name: "generics",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.testGenericStruct`),
			},
			testPackage: "testdata/types/generics",
		},
		{
			name: "collections",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.Test`),
			},
			testPackage: "testdata/types/collections",
		},
		{
			name: "anonymous",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.<anonymous>`),
			},
			testPackage: "testdata/types/anonymous",
		},
		{
			name: "directives",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.(Test|Embedded|Simple|WithOptionalDirective).*`),
				IgnorePatterns:  newList(t, `.*Excluded.*`),
			},
			testPackage: "testdata/types/directives",
		},
		{
			name: "filtering",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*\.Test.*`),
				IgnorePatterns:  newList(t, `.*Excluded.*`),
				ExplicitMode:    true,
			},
			testPackage: "testdata/types/filtering",
		},
		{
			name: "optional_pattern",
			config: analyzer.Config{
				EnforcePatterns:  newList(t, `.*\.Test.*`),
				OptionalPatterns: newList(t, `.*\.TestOptionalByPattern`),
				ExplicitMode:     true,
			},
			testPackage: "testdata/types/optional_pattern",
		},
		{
			name: "explicit mode with directives",
			config: analyzer.Config{
				EnforcePatterns: newList(t, `.*Enforced.*`),
				ExplicitMode:    true,
			},
			testPackage: "testdata/types/explicit",
		},
		{
			name:        "deprecated tags",
			config:      analyzer.Config{},
			testPackage: "testdata/types/tags",
			testFixes:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := analyzer.NewAnalyzer(tt.config)
			require.NoError(t, err)

			if tt.testFixes {
				analysistest.RunWithSuggestedFixes(t, testdataPath, a, tt.testPackage)
			} else {
				analysistest.Run(t, testdataPath, a, tt.testPackage)
			}
		})
	}
}
