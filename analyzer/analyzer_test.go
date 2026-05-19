package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"dev.gaijin.team/go/exhaustruct/v5/analyzer"
)

var testdataPath, _ = filepath.Abs("./testdata/") //nolint:gochecknoglobals

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzer(analyzer.Config{
		EnforcePatterns: []string{`.*\.TestExcluded`, `.*\.<anonymous>`},
		IgnorePatterns:  []string{`.*Excluded$`, `testdata/config/excluded\.<anonymous>`},
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

// TestAnalyzer_FlagsAffectAnalysis is a regression test for issue #155: flag-driven
// pattern lists must take effect, since the processor used to capture them at
// NewAnalyzer time, before flag parsing populated them.
func TestAnalyzer_FlagsAffectAnalysis(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzer(analyzer.Config{
		ExplicitMode: true,
	})
	require.NoError(t, err)

	require.NoError(t, a.Flags.Set("enforce-rx", `.*\.Test`))

	analysistest.Run(t, testdataPath, a, "testdata/types/basic")
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
				EnforcePatterns: []string{`.*\.Test`},
			},
			testPackage: "testdata/types/basic",
			testFixes:   false,
		},
		{
			name: "aliases",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.(Base|Alias|Simple).*`},
				IgnorePatterns:  []string{`.*Excluded.*`},
			},
			testPackage: "testdata/types/aliases",
			testFixes:   false,
		},
		{
			name: "derived",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.(Base|Derived|External|Simple).*`},
				IgnorePatterns:  []string{`.*Excluded.*`},
			},
			testPackage: "testdata/types/derived",
			testFixes:   false,
		},
		{
			name: "embedded",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.(Embedded|TestEmbedded|Simple).*`},
			},
			testPackage: "testdata/types/embedded",
			testFixes:   false,
		},
		{
			name: "generics",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.testGenericStruct`},
			},
			testPackage: "testdata/types/generics",
			testFixes:   false,
		},
		{
			name: "collections",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.Test`},
			},
			testPackage: "testdata/types/collections",
			testFixes:   false,
		},
		{
			name: "anonymous",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.<anonymous>`},
			},
			testPackage: "testdata/types/anonymous",
			testFixes:   false,
		},
		{
			name: "directives",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.(Test|Embedded|Simple|WithOptionalDirective).*`},
				IgnorePatterns:  []string{`.*Excluded.*`},
			},
			testPackage: "testdata/types/directives",
			testFixes:   false,
		},
		{
			name: "filtering",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.Test.*`},
				IgnorePatterns:  []string{`.*Excluded.*`},
				ExplicitMode:    true,
			},
			testPackage: "testdata/types/filtering",
			testFixes:   false,
		},
		{
			name: "optional_pattern",
			config: analyzer.Config{
				EnforcePatterns:  []string{`.*\.Test.*`},
				OptionalPatterns: []string{`.*\.TestOptionalByPattern`},
				ExplicitMode:     true,
			},
			testPackage: "testdata/types/optional_pattern",
			testFixes:   false,
		},
		{
			name: "explicit mode with directives",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*Enforced.*`},
				ExplicitMode:    true,
			},
			testPackage: "testdata/types/explicit",
			testFixes:   false,
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
