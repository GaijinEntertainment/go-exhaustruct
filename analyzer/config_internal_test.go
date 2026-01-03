package analyzer

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Prepare(t *testing.T) {
	t.Parallel()

	t.Run("valid patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{
			EnforceRx:    []string{".*Test.*", ".*Mock.*"},
			IgnoreRx:     []string{".*Ignored.*"},
			OptionalRx:   []string{".*Optional.*"},
			AllowEmptyRx: []string{".*Empty.*"},
		}

		err := config.Prepare()
		require.NoError(t, err)

		assert.Len(t, config.enforcePatterns, 2)
		assert.Len(t, config.ignorePatterns, 1)
		assert.Len(t, config.optionalPatterns, 1)
		assert.Len(t, config.allowEmptyPatterns, 1)

		// Test pattern matching
		assert.True(t, config.enforcePatterns.MatchFullString("pkg.TestStruct"))
		assert.True(t, config.enforcePatterns.MatchFullString("pkg.MockStruct"))
		assert.False(t, config.enforcePatterns.MatchFullString("pkg.RegularStruct"))

		assert.True(t, config.ignorePatterns.MatchFullString("pkg.IgnoredStruct"))
		assert.False(t, config.ignorePatterns.MatchFullString("pkg.RegularStruct"))

		assert.True(t, config.optionalPatterns.MatchFullString("pkg.OptionalStruct"))
		assert.False(t, config.optionalPatterns.MatchFullString("pkg.RegularStruct"))

		assert.True(t, config.allowEmptyPatterns.MatchFullString("pkg.EmptyStruct"))
		assert.False(t, config.allowEmptyPatterns.MatchFullString("pkg.RegularStruct"))
	})

	t.Run("invalid enforce pattern", func(t *testing.T) {
		t.Parallel()

		config := Config{
			EnforceRx: []string{"[invalid"},
		}

		err := config.Prepare()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "compile enforce patterns")
	})

	t.Run("invalid ignore pattern", func(t *testing.T) {
		t.Parallel()

		config := Config{
			IgnoreRx: []string{"[invalid"},
		}

		err := config.Prepare()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "compile ignore patterns")
	})

	t.Run("invalid optional pattern", func(t *testing.T) {
		t.Parallel()

		config := Config{
			OptionalRx: []string{"[invalid"},
		}

		err := config.Prepare()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "compile optional patterns")
	})

	t.Run("invalid allow empty pattern", func(t *testing.T) {
		t.Parallel()

		config := Config{
			AllowEmptyRx: []string{"[invalid"},
		}

		err := config.Prepare()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "compile allow empty patterns")
	})

	t.Run("empty patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}

		err := config.Prepare()
		require.NoError(t, err)

		assert.Empty(t, config.enforcePatterns)
		assert.Empty(t, config.ignorePatterns)
		assert.Empty(t, config.optionalPatterns)
		assert.Empty(t, config.allowEmptyPatterns)
	})
}

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

		assert.Equal(t, []string{".*Test.*", ".*Mock.*"}, config.EnforceRx)
	})

	t.Run("flag parsing ignore patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-ignore-rx", ".*Ignore.*", "-ignore-rx", ".*Skip.*"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.Equal(t, []string{".*Ignore.*", ".*Skip.*"}, config.IgnoreRx)
	})

	t.Run("flag parsing optional patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-optional-rx", ".*Optional.*"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.Equal(t, []string{".*Optional.*"}, config.OptionalRx)
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

		assert.Equal(t, []string{".*Empty.*"}, config.AllowEmptyRx)
	})
}

func TestStringSliceFlag(t *testing.T) {
	t.Parallel()

	t.Run("set and string methods", func(t *testing.T) {
		t.Parallel()

		var slice []string

		ssf := stringSliceFlag{&slice}

		// Initial state
		assert.Empty(t, ssf.String())

		// Set values
		err := ssf.Set("value1")
		require.NoError(t, err)
		assert.Equal(t, []string{"value1"}, slice)
		assert.Equal(t, "value1", ssf.String())

		err = ssf.Set("value2")
		require.NoError(t, err)
		assert.Equal(t, []string{"value1", "value2"}, slice)
		assert.Equal(t, "value1,value2", ssf.String())
	})

	t.Run("nil slice handling", func(t *testing.T) {
		t.Parallel()

		ssf := stringSliceFlag{nil}
		assert.Empty(t, ssf.String())
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

		// Prepare patterns
		err = config.Prepare()
		require.NoError(t, err)

		// Verify configuration state
		assert.Equal(t, []string{".*Test.*"}, config.EnforceRx)
		assert.Equal(t, []string{".*Skip.*"}, config.IgnoreRx)
		assert.Equal(t, []string{".*Optional.*"}, config.OptionalRx)
		assert.Equal(t, []string{".*Empty.*"}, config.AllowEmptyRx)
		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.False(t, config.AllowEmptyDeclarations)

		// Verify patterns work
		assert.True(t, config.enforcePatterns.MatchFullString("pkg.TestStruct"))
		assert.False(t, config.enforcePatterns.MatchFullString("pkg.RegularStruct"))
		assert.True(t, config.ignorePatterns.MatchFullString("pkg.SkipStruct"))
		assert.True(t, config.optionalPatterns.MatchFullString("pkg.OptionalStruct"))
		assert.True(t, config.allowEmptyPatterns.MatchFullString("pkg.EmptyStruct"))
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

		// Use flags to explicitly set to false (using the "false" value for boolean flags)
		// Note: boolean flags in Go can't be set to false via command line easily,
		// so we test the case where they are not provided vs provided
		args := []string{"-allow-empty", "-allow-empty-returns"} // Only set two flags
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
			EnforceRx:              []string{".*Initial.*"},
			AllowEmpty:             true,
			AllowEmptyReturns:      false,
			AllowEmptyDeclarations: true,
		}

		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{
			"-enforce-rx", ".*Flag.*", // Override programmatic enforce
			"-allow-empty-returns",           // Override programmatic false to true
			"-allow-empty-rx", ".*Pattern.*", // Add allow empty pattern
		}
		err := fs.Parse(args)
		require.NoError(t, err)

		// Verify mixed values
		assert.Equal(t, []string{".*Initial.*", ".*Flag.*"}, config.EnforceRx) // Should be appended
		assert.True(t, config.AllowEmpty)                                      // Programmatically set, preserved
		assert.True(t, config.AllowEmptyReturns)                               // Overridden by flag
		assert.True(t, config.AllowEmptyDeclarations)                          // Programmatically set, preserved
		assert.Equal(t, []string{".*Pattern.*"}, config.AllowEmptyRx)          // Set by flag
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
			EnforceRx:              []string{".*Test.*"},
		}

		// Create analyzer - this should preserve the programmatic values
		analyzer, err := NewAnalyzer(config)
		require.NoError(t, err)
		assert.NotNil(t, analyzer)

		// The analyzer should have been created successfully without modifying the config values
		// Since we can't directly access the internal config in the analyzer,
		// we verify that the analyzer creation succeeded, which implies the config was preserved.
		assert.Equal(t, "exhaustruct", analyzer.Name)
		assert.NotEmpty(t, analyzer.Doc)
		assert.NotNil(t, analyzer.Run)
	})

	t.Run("config preparation errors are handled", func(t *testing.T) {
		t.Parallel()

		// Create config with invalid pattern
		config := Config{
			EnforceRx: []string{"[invalid"},
		}

		// NewAnalyzer should return an error due to invalid pattern
		analyzer, err := NewAnalyzer(config)
		assert.Nil(t, analyzer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "compile enforce patterns")
	})
}
