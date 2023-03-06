package pattern_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/pattern"
)

func TestList_MatchFullString(t *testing.T) {
	t.Parallel()

	l, err := pattern.NewList()
	assert.NoError(t, err)
	assert.Nil(t, l)

	l, err = pattern.NewList("a", "b", "c")
	require.NoError(t, err)
	assert.Len(t, l, 3)

	assert.True(t, l.MatchFullString("a"))
	assert.True(t, l.MatchFullString("b"))
	assert.True(t, l.MatchFullString("c"))
	assert.False(t, l.MatchFullString("d"))

	l, err = pattern.NewList("")
	assert.Nil(t, l)
	assert.ErrorIs(t, err, pattern.ErrEmptyPattern)

	l, err = pattern.NewList("a", "b", "c[")
	assert.Nil(t, l)
	assert.ErrorIs(t, err, pattern.ErrCompilationFailed)

	l, err = pattern.NewList("abc")
	require.NoError(t, err)
	assert.Len(t, l, 1)

	assert.False(t, l.MatchFullString("a"))
	assert.False(t, l.MatchFullString("abcdef"))
	assert.True(t, l.MatchFullString("abc"))
}

func TestList_Set(t *testing.T) {
	t.Parallel()

	l, err := pattern.NewList("a", "b", "c")
	require.NoError(t, err)

	assert.NoError(t, l.Set("d"))
	assert.Len(t, l, 4)

	assert.ErrorIs(t, l.Set("e["), pattern.ErrCompilationFailed)
	assert.Len(t, l, 4)
}

func TestList_String(t *testing.T) {
	t.Parallel()

	l, err := pattern.NewList("a", "b", "c")
	require.NoError(t, err)
	assert.Equal(t, `"a", "b", "c"`, l.String())

	l, err = pattern.NewList()
	require.NoError(t, err)
	assert.Equal(t, "", l.String())
}
