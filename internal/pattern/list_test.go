package pattern_test

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/pattern"
)

func TestNewList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		patterns []string
		wantErr  bool
		wantLen  int
	}{
		{
			name:     "empty patterns",
			patterns: []string{},
			wantErr:  false,
			wantLen:  0,
		},
		{
			name:     "single valid pattern",
			patterns: []string{"test"},
			wantErr:  false,
			wantLen:  1,
		},
		{
			name:     "multiple valid patterns",
			patterns: []string{"test", "foo.*", "bar$"},
			wantErr:  false,
			wantLen:  3,
		},
		{
			name:     "empty string pattern causes error",
			patterns: []string{"test", "", "foo"},
			wantErr:  true,
			wantLen:  0,
		},
		{
			name:     "invalid regex pattern",
			patterns: []string{"test", "[invalid"},
			wantErr:  true,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			list, err := pattern.NewList(tt.patterns...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, list)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, tt.wantLen)
			}
		})
	}
}

func TestList_MatchFullString(t *testing.T) {
	t.Parallel()

	list, err := pattern.NewList("test", "^foo.*", ".*bar$", "^exact$")
	require.NoError(t, err)

	tests := []struct {
		name      string
		input     string
		wantMatch bool
	}{
		{
			name:      "matches first pattern exactly",
			input:     "test",
			wantMatch: true,
		},
		{
			name:      "does not match first pattern as substring",
			input:     "testing",
			wantMatch: false,
		},
		{
			name:      "matches second pattern fully",
			input:     "foobar",
			wantMatch: true,
		},
		{
			name:      "matches second pattern with prefix foo",
			input:     "foo",
			wantMatch: true,
		},
		{
			name:      "matches third pattern fully",
			input:     "foobar",
			wantMatch: true,
		},
		{
			name:      "matches third pattern with suffix bar",
			input:     "bar",
			wantMatch: true,
		},
		{
			name:      "matches fourth pattern exact",
			input:     "exact",
			wantMatch: true,
		},
		{
			name:      "does not match fourth pattern with extra chars",
			input:     "exactness",
			wantMatch: false,
		},
		{
			name:      "middle substring should not match",
			input:     "prefixfoosuffix",
			wantMatch: false,
		},
		{
			name:      "partial match at end should not match",
			input:     "testextra",
			wantMatch: false,
		},
		{
			name:      "no match",
			input:     "nomatch",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := list.MatchFullString(tt.input)
			assert.Equal(t, tt.wantMatch, got, "MatchFullString(%q) should return %v", tt.input, tt.wantMatch)
		})
	}
}

func TestList_MatchFullString_EmptyList(t *testing.T) {
	t.Parallel()

	var list pattern.List

	assert.False(t, list.MatchFullString("anything"), "empty list should not match anything")
}

func TestList_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		patterns []string
		want     string
	}{
		{
			name:     "empty list",
			patterns: []string{},
			want:     "",
		},
		{
			name:     "single pattern",
			patterns: []string{"test"},
			want:     "test",
		},
		{
			name:     "multiple patterns",
			patterns: []string{"test", "foo.*", "bar$"},
			want:     "test,foo.*,bar$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			list, err := pattern.NewList(tt.patterns...)
			require.NoError(t, err)

			got := list.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestList_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid pattern",
			value:   "test.*",
			wantErr: false,
		},
		{
			name:    "invalid pattern",
			value:   "[invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var list pattern.List

			err := list.Set(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, list, 1, "Set() should add exactly one pattern")
			}
		})
	}
}

func TestList_FlagIntegration(t *testing.T) {
	t.Parallel()

	var list pattern.List

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(&list, "pattern", "pattern to match")

	err := fs.Parse([]string{"-pattern", "test.*", "-pattern", "foo"})
	require.NoError(t, err, "flag parsing should succeed")

	assert.Len(t, list, 2, "should have parsed 2 patterns")
	assert.True(t, list.MatchFullString("testing"), "first pattern should match 'testing'")
	assert.True(t, list.MatchFullString("foo"), "second pattern should match 'foo'")
}
