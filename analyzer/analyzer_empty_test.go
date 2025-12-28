package analyzer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"dev.gaijin.team/go/exhaustruct/v4/analyzer"
)

func TestAnalyzerEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      analyzer.Config
		testPackage string
	}{
		{
			name: "allow empty globally",
			config: analyzer.Config{
				AllowEmpty: true,
			},
			testPackage: "testdata/config/allow_empty/global",
		},
		{
			name: "allow empty returns",
			config: analyzer.Config{
				AllowEmptyReturns: true,
			},
			testPackage: "testdata/config/allow_empty/returns",
		},
		{
			name: "allow empty declarations",
			config: analyzer.Config{
				AllowEmptyDeclarations: true,
			},
			testPackage: "testdata/config/allow_empty/declarations",
		},
		{
			name: "allow empty by pattern",
			config: analyzer.Config{
				AllowEmptyRx: []string{".*Allowed.*", ".*Nested.*"},
			},
			testPackage: "testdata/config/allow_empty/patterns",
		},
		{
			name:   "error returns behavior",
			config: analyzer.Config{
				// Test error returns without any special allowances -
				// structures should be allowed in error returns by default
			},
			testPackage: "testdata/config/allow_empty/error_returns",
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
