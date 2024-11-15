package files_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/go-exhaustruct/v4/internal/files"
)

func TestCommentsCache(t *testing.T) {
	t.Parallel()

	filename := "./testdata/comment-source.go"

	t.Run("ParseFile", func(t *testing.T) {
		t.Parallel()

		c := files.NewCommentsCache()

		require.Error(t, c.ParseFile("./definitely/non/existent"))

		require.NoError(t, c.ParseFile(filename))
		// add second time to see what everything ok happen in case of consequent call
		require.NoError(t, c.ParseFile(filename))

		assert.Len(t, c.Comments(filename), 6)

		testFileComments(t, c, filename)
	})

	t.Run("AddFile", func(t *testing.T) {
		t.Parallel()

		c := files.NewCommentsCache()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
		require.NoError(t, err)

		c.AddFile(fset, file)
		// add second time to see what everything ok happen in case of consequent call
		c.AddFile(fset, file)

		require.NoError(t, c.ParseFile(filename))

		assert.Len(t, c.Comments(filename), 6)

		testFileComments(t, c, filename)
	})
}

func testFileComments(t *testing.T, c *files.CommentsCache, filename string) {
	t.Helper()

	assert.Equal(t, []files.CommentGroup{
		{
			Text: []string{"// Test before structure name."},
			Start: token.Position{
				Filename: filename,
				Line:     3,
				Column:   1,
				Offset:   18,
			},
			End: token.Position{
				Filename: filename,
				Line:     3,
				Column:   31,
				Offset:   48,
			},
		},
		{
			Text: []string{"// after structure name"},
			Start: token.Position{
				Filename: filename,
				Line:     4,
				Column:   20,
				Offset:   68,
			},
			End: token.Position{
				Filename: filename,
				Line:     4,
				Column:   43,
				Offset:   91,
			},
		},
	}, c.CommentsForPosition(token.Position{ //nolint:exhaustruct
		Filename: filename,
		Line:     4,
	}))

	assert.Equal(t, []files.CommentGroup{
		{
			Text: []string{"// after field declaration"},
			Start: token.Position{
				Filename: filename,
				Line:     5,
				Column:   13,
				Offset:   104,
			},
			End: token.Position{
				Filename: filename,
				Line:     5,
				Column:   39,
				Offset:   130,
			},
		},
	}, c.CommentsForPosition(token.Position{ //nolint:exhaustruct
		Filename: filename,
		Line:     5,
	}, token.Position{ //nolint:exhaustruct
		Filename: filename,
		Line:     4,
	}))

	assert.Equal(t, []files.CommentGroup{
		{
			Text: []string{
				"// before field declaration",
				"// miltiline comment",
				"//",
				"// with empty lines",
			},
			Start: token.Position{
				Filename: filename,
				Line:     6,
				Column:   2,
				Offset:   132,
			},
			End: token.Position{
				Filename: filename,
				Line:     9,
				Column:   21,
				Offset:   206,
			},
		},
		{
			Text: []string{"// after field declaration [2]"},
			Start: token.Position{
				Filename: filename,
				Line:     10,
				Column:   13,
				Offset:   219,
			},
			End: token.Position{
				Filename: filename,
				Line:     10,
				Column:   43,
				Offset:   249,
			},
		},
	}, c.CommentsForPosition(token.Position{ //nolint:exhaustruct
		Filename: filename,
		Line:     10,
	}, token.Position{ //nolint:exhaustruct
		Filename: filename,
		Line:     5,
	}))

	//nolint:exhaustruct
	assert.Nil(t, c.CommentsForPosition(token.Position{Filename: "./definitely/non/existent"}))
}
