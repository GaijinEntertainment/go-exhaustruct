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
			testPackage: "empty_global",
		},
		{
			name: "allow empty returns",
			config: analyzer.Config{
				AllowEmptyReturns: true,
			},
			testPackage: "empty_returns",
		},
		{
			name: "allow empty declarations",
			config: analyzer.Config{
				AllowEmptyDeclarations: true,
			},
			testPackage: "empty_declarations",
		},
		{
			name: "allow empty by pattern",
			config: analyzer.Config{
				AllowEmptyRx: []string{".*Allowed.*", ".*Nested.*"},
			},
			testPackage: "empty_patterns",
		},
		{
			name:   "error returns behavior",
			config: analyzer.Config{
				// Test error returns without any special allowances -
				// structures should be allowed in error returns by default
			},
			testPackage: "empty_error_returns",
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
