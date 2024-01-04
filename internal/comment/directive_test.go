package comment_test

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/comment"
)

func TestParseDirective(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		comments  []*ast.CommentGroup
		directive comment.Directive
		found     bool
	}{
		{
			name: "no directive",
			comments: []*ast.CommentGroup{
				{
					List: []*ast.Comment{
						{
							Text: "// some comment",
						},
					},
				},
			},
			directive: comment.DirectiveIgnore,
			found:     false,
		},
		{
			name: "directive found",
			comments: []*ast.CommentGroup{
				{
					List: []*ast.Comment{
						{
							Text: "//exhaustruct:ignore",
						},
						{
							Text: "// some comment",
						},
						{
							Text: "//exhaustruct:enforce",
						},
					},
				},
			},
			directive: comment.DirectiveIgnore,
			found:     true,
		},
		{
			name: "directive found (partial line match)",
			comments: []*ast.CommentGroup{
				{
					List: []*ast.Comment{
						{
							Text: "//exhaustruct:ignore",
						},
						{
							Text: "// some comment",
						},
						{
							Text: "//exhaustruct:enforce beacuse of some reason",
						},
					},
				},
			},
			directive: comment.DirectiveEnforce,
			found:     true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.found, comment.HasDirective(tt.comments, tt.directive))
		})
	}
}
