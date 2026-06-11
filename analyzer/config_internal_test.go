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
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

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
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-enforce-rx", ".*Test.*", "-enforce-rx", ".*Mock.*"}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, Patterns{".*Test.*", ".*Mock.*"}, config.EnforcePatterns)
	})

	t.Run("flag parsing ignore patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-ignore-rx", ".*Ignore.*", "-ignore-rx", ".*Skip.*"}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, Patterns{".*Ignore.*", ".*Skip.*"}, config.IgnorePatterns)
	})

	t.Run("flag parsing optional patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-optional-rx", ".*Optional.*"}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, Patterns{".*Optional.*"}, config.OptionalPatterns)
	})

	t.Run("flag parsing boolean flags", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-allow-empty", "-allow-empty-returns", "-allow-empty-declarations"}
		require.NoError(t, fs.Parse(args))

		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.True(t, config.AllowEmptyDeclarations)
	})

	t.Run("flag parsing allow-empty-rx patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-allow-empty-rx", ".*Empty.*"}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, Patterns{".*Empty.*"}, config.AllowEmptyPatterns)
	})

	t.Run("invalid pattern fails at parse time", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		assert.Error(t, fs.Parse([]string{"-enforce-rx", "[invalid"}))
	})

	t.Run("empty pattern fails at parse time", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		assert.Error(t, fs.Parse([]string{"-enforce-rx", ""}))
	})
}

func TestConfig_Integration(t *testing.T) {
	t.Parallel()

	t.Run("full workflow", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{
			"-enforce-rx", ".*Test.*",
			"-ignore-rx", ".*Skip.*",
			"-optional-rx", ".*Optional.*",
			"-allow-empty",
			"-allow-empty-rx", ".*Empty.*",
			"-allow-empty-returns",
		}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, Patterns{".*Test.*"}, config.EnforcePatterns)
		assert.Equal(t, Patterns{".*Skip.*"}, config.IgnorePatterns)
		assert.Equal(t, Patterns{".*Optional.*"}, config.OptionalPatterns)
		assert.Equal(t, Patterns{".*Empty.*"}, config.AllowEmptyPatterns)
		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.False(t, config.AllowEmptyDeclarations)
	})
}

func TestConfig_ProgrammaticDefaults(t *testing.T) {
	t.Parallel()

	t.Run("programmatically set values are preserved as flag defaults", func(t *testing.T) {
		t.Parallel()

		config := Config{
			AllowEmpty:             true,
			AllowEmptyReturns:      true,
			AllowEmptyDeclarations: true,
		}

		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))
		require.NoError(t, fs.Parse([]string{}))

		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.True(t, config.AllowEmptyDeclarations)
	})

	t.Run("flags can override programmatically set values", func(t *testing.T) {
		t.Parallel()

		config := Config{
			AllowEmpty:             true,
			AllowEmptyReturns:      true,
			AllowEmptyDeclarations: true,
		}

		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))
		require.NoError(t, fs.Parse([]string{"-allow-empty", "-allow-empty-returns"}))

		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.True(t, config.AllowEmptyDeclarations)
	})

	t.Run("mixed programmatic and flag patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{
			EnforcePatterns:        []string{".*Initial.*"},
			AllowEmpty:             true,
			AllowEmptyReturns:      false,
			AllowEmptyDeclarations: true,
		}

		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{
			"-enforce-rx", ".*Flag.*",
			"-allow-empty-returns",
			"-allow-empty-rx", ".*Pattern.*",
		}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, Patterns{".*Initial.*", ".*Flag.*"}, config.EnforcePatterns)
		assert.Equal(t, Patterns{".*Pattern.*"}, config.AllowEmptyPatterns)
		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.True(t, config.AllowEmptyDeclarations)
	})
}

func TestNewAnalyzerWithConfig_ConfigPreservation(t *testing.T) {
	t.Parallel()

	t.Run("programmatic config values preserved in analyzer", func(t *testing.T) {
		t.Parallel()

		config := Config{
			EnforcePatterns:        []string{".*Test.*"},
			AllowEmpty:             true,
			AllowEmptyReturns:      true,
			AllowEmptyDeclarations: false,
		}

		a, err := NewAnalyzerWithConfig(config)
		require.NoError(t, err)
		require.NotNil(t, a)

		assert.Equal(t, "exhaustruct", a.Name)
		assert.NotEmpty(t, a.Doc)
		assert.NotNil(t, a.Run)
	})

	t.Run("invalid pattern fails at set time", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.bindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		assert.Error(t, fs.Parse([]string{"-enforce-rx", "[invalid"}))
	})
}
