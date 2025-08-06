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
			IncludeRx:    []string{".*Test.*", ".*Mock.*"},
			ExcludeRx:    []string{".*Excluded.*"},
			AllowEmptyRx: []string{".*Empty.*"},
		}

		err := config.Prepare()
		require.NoError(t, err)

		assert.Len(t, config.includePatterns, 2)
		assert.Len(t, config.excludePatterns, 1)
		assert.Len(t, config.allowEmptyPatterns, 1)

		// Test pattern matching
		assert.True(t, config.includePatterns.MatchFullString("pkg.TestStruct"))
		assert.True(t, config.includePatterns.MatchFullString("pkg.MockStruct"))
		assert.False(t, config.includePatterns.MatchFullString("pkg.RegularStruct"))

		assert.True(t, config.excludePatterns.MatchFullString("pkg.ExcludedStruct"))
		assert.False(t, config.excludePatterns.MatchFullString("pkg.RegularStruct"))

		assert.True(t, config.allowEmptyPatterns.MatchFullString("pkg.EmptyStruct"))
		assert.False(t, config.allowEmptyPatterns.MatchFullString("pkg.RegularStruct"))
	})

	t.Run("invalid include pattern", func(t *testing.T) {
		t.Parallel()

		config := Config{
			IncludeRx: []string{"[invalid"},
		}

		err := config.Prepare()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "compile include patterns")
	})

	t.Run("invalid exclude pattern", func(t *testing.T) {
		t.Parallel()

		config := Config{
			ExcludeRx: []string{"[invalid"},
		}

		err := config.Prepare()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "compile exclude patterns")
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

		assert.Empty(t, config.includePatterns)
		assert.Empty(t, config.excludePatterns)
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
			"include", "i", "exclude", "e",
			"allow-empty", "allow-empty-include",
			"allow-empty-returns", "allow-empty-declarations",
		}

		for _, flagName := range expectedFlags {
			f := fs.Lookup(flagName)
			assert.NotNil(t, f, "flag %s should be registered", flagName)
		}
	})

	t.Run("flag parsing include patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-include", ".*Test.*", "-i", ".*Mock.*"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.Equal(t, []string{".*Test.*", ".*Mock.*"}, config.IncludeRx)
	})

	t.Run("flag parsing exclude patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-exclude", ".*Exclude.*", "-e", ".*Skip.*"}
		err := fs.Parse(args)
		require.NoError(t, err)

		assert.Equal(t, []string{".*Exclude.*", ".*Skip.*"}, config.ExcludeRx)
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

	t.Run("flag parsing allow-empty-include patterns", func(t *testing.T) {
		t.Parallel()

		config := Config{}
		fs := config.BindToFlagSet(flag.NewFlagSet("test", flag.ContinueOnError))

		args := []string{"-allow-empty-include", ".*Empty.*"}
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
			"-include", ".*Test.*",
			"-exclude", ".*Skip.*",
			"-allow-empty",
			"-allow-empty-include", ".*Empty.*",
			"-allow-empty-returns",
		}
		err := fs.Parse(args)
		require.NoError(t, err)

		// Prepare patterns
		err = config.Prepare()
		require.NoError(t, err)

		// Verify configuration state
		assert.Equal(t, []string{".*Test.*"}, config.IncludeRx)
		assert.Equal(t, []string{".*Skip.*"}, config.ExcludeRx)
		assert.Equal(t, []string{".*Empty.*"}, config.AllowEmptyRx)
		assert.True(t, config.AllowEmpty)
		assert.True(t, config.AllowEmptyReturns)
		assert.False(t, config.AllowEmptyDeclarations)

		// Verify patterns work
		assert.True(t, config.includePatterns.MatchFullString("pkg.TestStruct"))
		assert.False(t, config.includePatterns.MatchFullString("pkg.RegularStruct"))
		assert.True(t, config.excludePatterns.MatchFullString("pkg.SkipStruct"))
		assert.True(t, config.allowEmptyPatterns.MatchFullString("pkg.EmptyStruct"))
	})
}
