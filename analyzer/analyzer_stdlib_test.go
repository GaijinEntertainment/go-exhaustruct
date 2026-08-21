package analyzer_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	"dev.gaijin.team/go/exhaustruct/v5/analyzer"
)

// TestAnalyzer_StdlibExportData is a regression test for issue #166: literals of
// standard library types used to produce positionless "read file" diagnostics.
//
// It cannot use analysistest, which type-checks dependencies from source and so
// reports standard library definitions at real, readable paths. Real drivers
// (singlechecker, go vet, golangci-lint) load dependencies from export data,
// where the compiler records the literal "$GOROOT" placeholder instead of a
// path that can be opened. This test loads the fixture the same way.
func TestAnalyzer_StdlibExportData(t *testing.T) {
	t.Parallel()

	pkg := loadFromExportData(t, "testdata/types/stdlib")

	require.True(t,
		strings.HasPrefix(stdlibTypePos(t, pkg, "strings", "Builder"), "$GOROOT"),
		"standard library positions must carry the $GOROOT placeholder, "+
			"otherwise this test no longer covers the regression",
	)

	a, err := analyzer.NewAnalyzerWithConfig(analyzer.Config{})
	require.NoError(t, err)

	diags := analyze(t, a, pkg)
	messages := make([]string, 0, len(diags))

	for _, diag := range diags {
		messages = append(messages, diag.Message)
	}

	// Standard library structs are checked like any other — only their
	// directives are known to be absent, not their fields. Resolving their
	// definitions must contribute no diagnostics of its own.
	assert.ElementsMatch(t, []string{
		"net.TCPAddr is missing fields IP, Port, Zone",
		"net.TCPAddr is missing fields Port, Zone",
	}, messages)
}

// loadFromExportData loads a single testdata package, type-checking its
// dependencies from export data rather than from source.
func loadFromExportData(t *testing.T, pattern string) *packages.Package {
	t.Helper()

	cfg := &packages.Config{
		// NeedDeps is deliberately absent: it is what makes the loader
		// type-check dependencies from source.
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Dir: testdataPath,
		Env: append(os.Environ(), "GO111MODULE=on", "GOPROXY=off", "GOWORK=off"),
	}

	pkgs, err := packages.Load(cfg, pattern)
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	require.Empty(t, pkgs[0].Errors)

	return pkgs[0]
}

// stdlibTypePos returns the file name recorded for a type imported from the
// standard library.
func stdlibTypePos(t *testing.T, pkg *packages.Package, importPath, typeName string) string {
	t.Helper()

	for _, imported := range pkg.Types.Imports() {
		if imported.Path() != importPath {
			continue
		}

		obj := imported.Scope().Lookup(typeName)
		require.NotNil(t, obj)

		return pkg.Fset.Position(obj.Pos()).Filename
	}

	t.Fatalf("package %q does not import %q", pkg.PkgPath, importPath)

	return ""
}

// analyze runs an analyzer over a loaded package and collects its diagnostics.
func analyze(t *testing.T, a *analysis.Analyzer, pkg *packages.Package) []analysis.Diagnostic {
	t.Helper()

	graph, err := checker.Analyze([]*analysis.Analyzer{a}, []*packages.Package{pkg}, nil)
	require.NoError(t, err)

	var diags []analysis.Diagnostic

	for act := range graph.All() {
		require.NoError(t, act.Err)

		diags = append(diags, act.Diagnostics...)
	}

	return diags
}
