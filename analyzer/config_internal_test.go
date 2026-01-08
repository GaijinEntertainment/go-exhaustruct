package analyzer

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_BindToFlagSet(t *testing.T) {
	t.Parallel()

	t.Run("bind all flags", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		// Check that flags are registered
		expectedFlags := []string{
			"enforce-rx", "ignore-rx", "optional-rx",
			"allow-empty", "allow-empty-rx",
			"allow-empty-returns", "allow-empty-declarations",
			"report-full-type-path", "debug-cache-metrics",
		}

		for _, flagName := range expectedFlags {
			f := fs.Lookup(flagName)
			assert.NotNil(t, f, "flag %s should be registered", flagName)
		}
	})

	t.Run("flag parsing enforce patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-enforce-rx", ".*Test.*", "-enforce-rx", ".*Mock.*"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.Len(t, config.EnforcePatterns, 2)
		assert.True(t, config.EnforcePatterns.MatchFullString("pkg.TestStruct"))
		assert.True(t, config.EnforcePatterns.MatchFullString("pkg.MockStruct"))
		assert.False(t, config.EnforcePatterns.MatchFullString("pkg.RegularStruct"))
	})

	t.Run("flag parsing ignore patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-ignore-rx", ".*Ignore.*", "-ignore-rx", ".*Skip.*"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.Len(t, config.IgnorePatterns, 2)
		assert.True(t, config.IgnorePatterns.MatchFullString("pkg.IgnoreStruct"))
		assert.True(t, config.IgnorePatterns.MatchFullString("pkg.SkipStruct"))
		assert.False(t, config.IgnorePatterns.MatchFullString("pkg.RegularStruct"))
	})

	t.Run("flag parsing optional patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-optional-rx", ".*Optional.*"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.Len(t, config.OptionalPatterns, 1)
		assert.True(t, config.OptionalPatterns.MatchFullString("pkg.OptionalStruct"))
		assert.False(t, config.OptionalPatterns.MatchFullString("pkg.RegularStruct"))
	})

	t.Run("flag parsing boolean flags", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-allow-empty", "-allow-empty-returns", "-allow-empty-declarations"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.True(t, config.AllowEmptyDeclarations)
	})

	t.Run("flag parsing allow-empty-rx patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-allow-empty-rx", ".*Empty.*"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.Len(t, config.AllowEmptyPatterns, 1)
		assert.True(t, config.AllowEmptyPatterns.MatchFullString("pkg.EmptyStruct"))
		assert.False(t, config.AllowEmptyPatterns.MatchFullString("pkg.RegularStruct"))
	})

	t.Run("invalid pattern fails at parse time", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-enforce-rx", "[invalid"}
		err := fs.Parse(args)
		assert.Error(t, err)
	})
}

func TestConfig_Integration(t *testing.T) {
	t.Parallel()

	t.Run("full workflow", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		// Simulate command line arguments
		args := []string{
			"-enforce-rx", ".*Test.*",
			"-ignore-rx", ".*Skip.*",
			"-optional-rx", ".*Optional.*",
			"-allow-empty",
			"-allow-empty-rx", ".*Empty.*",
			"-allow-empty-returns",
		}
		err := fs.Parse(args)
		require.NoError(t, err)

		// Verify configuration state
		assert.Len(t, config.EnforcePatterns, 1)
		assert.Len(t, config.IgnorePatterns, 1)
		assert.Len(t, config.OptionalPatterns, 1)
		assert.Len(t, config.AllowEmptyPatterns, 1)
		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.False(t, config.AllowEmptyDeclarations)

		// Verify patterns work
		assert.True(t, config.EnforcePatterns.MatchFullString("pkg.TestStruct"))
		assert.False(t, config.EnforcePatterns.MatchFullString("pkg.RegularStruct"))
		assert.True(t, config.IgnorePatterns.MatchFullString("pkg.SkipStruct"))
		assert.True(t, config.OptionalPatterns.MatchFullString("pkg.OptionalStruct"))
		assert.True(t, config.AllowEmptyPatterns.MatchFullString("pkg.EmptyStruct"))
	})
}

func TestConfig_ProgrammaticDefaults(t *testing.T) {
	t.Parallel()

	t.Run("programmatically set values are preserved as flag defaults", func(t *testing.T) {
		t.Parallel()

		// Set values programmatically before binding to flags
		config := Config{
			AllowEmpty:             true,
			AllowEmptyReturns:      true,
			AllowEmptyDeclarations: true,
		}

		// Bind to flag set without parsing any arguments
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		// Parse empty arguments (no flags provided)
		err := fs.Parse([]string{})
		require.NoError(t, err)

		// Verify that programmatically set values are preserved
		assert.True(t, config.AllowEmpty, "AllowEmpty should remain true when set programmatically")
		assert.True(t, config.AllowEmptyReturns, "AllowEmptyReturns should remain true when set programmatically")
		assert.True(t, config.AllowEmptyDeclarations, "AllowEmptyDeclarations should remain true when set programmatically")
	})

	t.Run("flags can override programmatically set values", func(t *testing.T) {
		t.Parallel()

		// Set values programmatically
		config := Config{
			AllowEmpty:             true,
			AllowEmptyReturns:      true,
			AllowEmptyDeclarations: true,
		}

		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		// Use flags to explicitly set (boolean flags in Go can only be set to true via command line)
		args := []string{"-allow-empty", "-allow-empty-returns"}
		err := fs.Parse(args)
		require.NoError(t, err)

		// These should be true (set by flags)
		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		// This should remain true (programmatically set, flag not provided)
		assert.True(t, config.AllowEmptyDeclarations)
	})

	t.Run("mixed programmatic and flag values", func(t *testing.T) {
		t.Parallel()

		config := Config{
			AllowEmpty:             true,
			AllowEmptyReturns:      false,
			AllowEmptyDeclarations: true,
		}

		// Pre-add an enforce pattern programmatically
		err := config.EnforcePatterns.Set(".*Initial.*")
		require.NoError(t, err)

		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{
			"-enforce-rx", ".*Flag.*", // AddFile to existing enforce patterns
			"-allow-empty-returns",           // Override programmatic false to true
			"-allow-empty-rx", ".*Pattern.*", // AddFile allow empty pattern
		}

		err = fs.Parse(args)
		require.NoError(t, err)

		// Verify mixed values
		assert.Len(t, config.EnforcePatterns, 2) // Initial + Flag
		assert.True(t, config.EnforcePatterns.MatchFullString("pkg.InitialStruct"))
		assert.True(t, config.EnforcePatterns.MatchFullString("pkg.FlagStruct"))
		assert.True(t, config.AllowEmpty)             // Programmatically set, preserved
		assert.True(t, config.AllowEmptyReturns)      // Overridden by flag
		assert.True(t, config.AllowEmptyDeclarations) // Programmatically set, preserved
		assert.Len(t, config.AllowEmptyPatterns, 1)   // Set by flag
	})
}

func TestNewAnalyzer_ConfigPreservation(t *testing.T) {
	t.Parallel()

	t.Run("programmatic config values preserved in analyzer", func(t *testing.T) {
		t.Parallel()

		// Create config with programmatic values
		config := Config{
			AllowEmpty:             true,
			AllowEmptyReturns:      true,
			AllowEmptyDeclarations: false,
		}

		// AddFile enforce pattern programmatically
		err := config.EnforcePatterns.Set(".*Test.*")
		require.NoError(t, err)

		// Create analyzer - this should preserve the programmatic values
		analyzer, err := NewAnalyzer(config)
		require.NoError(t, err)
		assert.NotNil(t, analyzer)

		// The analyzer should have been created successfully
		assert.Equal(t, "exhaustruct", analyzer.Name)
		assert.NotEmpty(t, analyzer.Doc)
		assert.NotNil(t, analyzer.Run)
	})

	t.Run("invalid pattern fails at set time", func(t *testing.T) {
		t.Parallel()

		config := Config{}

		// Invalid pattern should fail when Set is called
		err := config.EnforcePatterns.Set("[invalid")
		assert.Error(t, err)
	})
}
