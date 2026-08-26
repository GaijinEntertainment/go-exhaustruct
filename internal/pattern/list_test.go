package pattern_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v5/internal/pattern"
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

func TestList_MatchFullString_NilList(t *testing.T) {
	t.Parallel()

	var list pattern.List

	assert.False(t, list.MatchFullString("anything"), "nil list should not match anything")
}

// TestList_MatchFullString_Alternation covers alternations whose shorter branch
// precedes a longer one. Go's regexp is leftmost-first, so a search commits to
// the shorter branch and the match stops short of the end — even though the
// pattern can match the whole string.
func TestList_MatchFullString_Alternation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
	}{
		{"shorter branch first", `.*\.(Config|ConfigOption)`, "example.com/pkg.ConfigOption", true},
		{"longer branch first", `.*\.(ConfigOption|Config)`, "example.com/pkg.ConfigOption", true},
		{"shorter branch still matches", `.*\.(Config|ConfigOption)`, "example.com/pkg.Config", true},
		{"prefix of a branch does not match", `.*\.(Config|ConfigOption)`, "example.com/pkg.Conf", false},
		{"branch is a strict prefix of target", `.*\.(Config|ConfigOption)`, "example.com/pkg.Configs", false},
		{"nested alternation", `.*\.(A|AB|ABC)Suffix`, "example.com/pkg.ABCSuffix", true},
		{"optional group", `.*\.Config(Option)?`, "example.com/pkg.ConfigOption", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			list, err := pattern.NewList(tt.pattern)
			require.NoError(t, err)

			assert.Equal(t, tt.want, list.MatchFullString(tt.target))
		})
	}
}

// TestList_MatchFullString_AuthorAnchors covers patterns an author anchored
// themselves. A pattern matches the whole target however it is written, so the
// anchors are redundant, and the ones written inside a group or a branch must
// keep meaning what the author meant.
func TestList_MatchFullString_AuthorAnchors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
	}{
		{"both anchors", `^example\.com/pkg\.Config$`, "example.com/pkg.Config", true},
		{"start anchor only", `^example\.com/pkg\.Config`, "example.com/pkg.Config", true},
		{"end anchor only", `example\.com/pkg\.Config$`, "example.com/pkg.Config", true},
		{"anchors around an alternation", `^.*\.(Config|ConfigOption)$`, "example.com/pkg.ConfigOption", true},
		{"anchors inside each branch", `^a$|^example\.com/pkg\.Config$`, "example.com/pkg.Config", true},
		{"end anchor inside one branch", `a$|example\.com/pkg\.Config`, "example.com/pkg.Config", true},
		{"text anchors", `\Aexample\.com/pkg\.Config\z`, "example.com/pkg.Config", true},
		{"anchored pattern still rejects a longer target", `^pkg\.Config$`, "pkg.ConfigOption", false},
		{"case-insensitive flag", `(?i)EXAMPLE\.COM/PKG\.CONFIG`, "example.com/pkg.Config", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			list, err := pattern.NewList(tt.pattern)
			require.NoError(t, err)

			assert.Equal(t, tt.want, list.MatchFullString(tt.target))
		})
	}
}

// TestList_MatchFullStringExcept covers the pattern that names one target and
// not the other. A list holding a broad pattern and a specific one answers by
// the specific one, which is what tells a rule written for a field from one
// that reaches the field through the type holding it.
func TestList_MatchFullStringExcept(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		patterns []string
		want     bool
	}{
		{"one broad pattern reaching both", []string{`.*Outer.*`}, false},
		{"one pattern naming the field", []string{`.*\.Outer#Inner$`}, true},
		{"one pattern naming the type", []string{`.*\.Outer$`}, false},
		{"a pattern for each", []string{`.*\.Outer$`, `.*\.Outer#Inner$`}, true},
		{"no pattern at all", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			list, err := pattern.NewList(tt.patterns...)
			require.NoError(t, err)

			assert.Equal(t, tt.want,
				list.MatchFullStringExcept("example.com/dep.Outer#Inner", "example.com/dep.Outer"))
		})
	}
}
