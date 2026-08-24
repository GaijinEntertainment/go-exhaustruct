package analyzer_test

import (
	"go/build"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"dev.gaijin.team/go/exhaustruct/v5/analyzer"
)

var testdataPath, _ = filepath.Abs("./testdata/") //nolint:gochecknoglobals

// go127TestdataPath holds fixtures that need a language version newer than the
// main testdata module declares. It is a module of its own so it can raise that
// version without dragging every other fixture along.
var go127TestdataPath, _ = filepath.Abs("./testdata/go127/") //nolint:gochecknoglobals

// TestAnalyzerPromotedFields covers literals naming promoted fields, allowed
// since Go 1.27 (golang/go#77245). Older toolchains cannot compile the fixture,
// so they skip it -- the resolution logic itself is covered version-independently
// in internal/structure.
func TestAnalyzerPromotedFields(t *testing.T) {
	t.Parallel()

	if !slices.Contains(build.Default.ReleaseTags, "go1.27") {
		t.Skip("promoted fields in composite literals require Go 1.27")
	}

	a, err := analyzer.NewAnalyzerWithConfig(analyzer.Config{})
	require.NoError(t, err)

	analysistest.Run(t, go127TestdataPath, a, "go127/promoted")
}

// TestAnalyzerIgnoredTags covers deprecated tags on a type excluded by pattern.
func TestAnalyzerIgnoredTags(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzerWithConfig(analyzer.Config{
		IgnorePatterns: []string{`.*\.Excluded.*`, `.*\.<anonymous>`},
	})
	require.NoError(t, err)

	analysistest.RunWithSuggestedFixes(t, testdataPath, a, "testdata/config/ignored_tags")
}

// TestAnalyzerDependencyDirectives covers directives that are malformed in a
// dependency. A diagnostic about them would land on a file the analyzed package
// does not own and usually cannot edit, so the consuming package reports only
// its own findings.
func TestAnalyzerDependencyDirectives(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzerWithConfig(analyzer.Config{})
	require.NoError(t, err)

	// The consumer runs first and alone, so its lookup is what parses the
	// dependency's file. Nothing about the malformed directive is reported
	// here, and the order is the test's rather than the scheduler's.
	analysistest.Run(t, testdataPath, a, "testdata/config/dep_directives/user")

	// The owner is analysed after that parse, against a FileSet of its own, and
	// reports its own directive at its own position.
	analysistest.Run(t, testdataPath, a, "testdata/config/dep_directives/dep")
}

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzerWithConfig(analyzer.Config{
		EnforcePatterns: []string{`.*\.TestExcluded`, `.*\.<anonymous>`},
		IgnorePatterns:  []string{`.*Excluded$`, `testdata/config/excluded\.<anonymous>`},
	})
	require.NoError(t, err)

	analysistest.Run(t, testdataPath, a, "testdata/config/excluded")
}

func TestAnalyzerReportFullTypePath(t *testing.T) {
	t.Parallel()

	a, err := analyzer.NewAnalyzerWithConfig(analyzer.Config{
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

	a := analyzer.NewAnalyzer()

	require.NoError(t, a.Flags.Set("explicit", "true"))
	require.NoError(t, a.Flags.Set("enforce-rx", `.*\.Test`))

	analysistest.Run(t, testdataPath, a, "testdata/types/basic")
}

// typeCase is one fixture package with the configuration it is written
// against. The tests run every case and the benchmark runs those marked for
// it, so the two read one description of a fixture and cannot drift apart.
type typeCase struct {
	name        string
	config      analyzer.Config
	testPackage string
	testFixes   bool
	// benchmark marks a case BenchmarkAnalyzer runs as well.
	benchmark bool //exhaustruct:optional
}

func typeCases() []typeCase {
	return []typeCase{
		{
			name: "basic",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.Test`},
			},
			testPackage: "testdata/types/basic",
			testFixes:   false,
			benchmark:   true,
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
			benchmark:   true,
		},
		{
			name: "embedded",
			config: analyzer.Config{
				EnforcePatterns: []string{`.*\.(Embedded|TestEmbedded|Simple).*`},
			},
			testPackage: "testdata/types/embedded",
			testFixes:   false,
			benchmark:   true,
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
			benchmark:   true,
		},
		{
			name:        "blank",
			config:      analyzer.Config{},
			testPackage: "testdata/types/blank",
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
			name:        "pointers",
			config:      analyzer.Config{},
			testPackage: "testdata/types/pointers",
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
		{
			name:        "deprecated tags behind a line directive",
			config:      analyzer.Config{},
			testPackage: "testdata/types/lined_tags",
			testFixes:   true,
		},
		{
			name: "deprecated tags on a parenthesised declaration",
			config: analyzer.Config{
				IgnorePatterns: []string{`.*\.ExcludedParen`},
			},
			testPackage: "testdata/types/paren_tags",
			testFixes:   false,
		},
	}
}

func TestAnalyzerTypes(t *testing.T) {
	t.Parallel()

	for _, tt := range typeCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := analyzer.NewAnalyzerWithConfig(tt.config)
			require.NoError(t, err)

			if tt.testFixes {
				analysistest.RunWithSuggestedFixes(t, testdataPath, a, tt.testPackage)
			} else {
				analysistest.Run(t, testdataPath, a, tt.testPackage)
			}
		})
	}
}
