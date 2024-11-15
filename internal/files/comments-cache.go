package files

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sync"
)

// CommentGroup is a "baked" representation of an AST's comment group that
// doesn't require further access to the [token.FileSet] it was taken from, as we
// won't have access to the original fset. Among other benefits, it allows
// reducing the number of locks within the accessed fset, as it is accessed only
// twice during conversion.
type CommentGroup struct {
	Text  []string
	Start token.Position
	End   token.Position
}

func NewCommentGroup(fset *token.FileSet, cg *ast.CommentGroup) CommentGroup {
	text := make([]string, 0, len(cg.List))
	for _, comment := range cg.List {
		text = append(text, comment.Text)
	}

	return CommentGroup{
		Text:  text,
		Start: fset.PositionFor(cg.Pos(), true),
		End:   fset.PositionFor(cg.End(), true),
	}
}

type CommentsCache struct {
	mu       sync.RWMutex
	comments map[string][]CommentGroup
}

func NewCommentsCache() *CommentsCache {
	return &CommentsCache{
		mu:       sync.RWMutex{},
		comments: make(map[string][]CommentGroup, 64), //nolint:mnd
	}
}

// ParseFile parses the provided file, including comments, and adds the parsed
// comments to the internal cache. If the file is already parsed and present
// in the cache, the function does nothing.
//
// The function returns an error if the file cannot be parsed.
func (c *CommentsCache) ParseFile(filename string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.comments[filename] != nil {
		return nil
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err //nolint:wrapcheck
	}

	comments := make([]CommentGroup, len(file.Comments))
	for i, cg := range file.Comments {
		comments[i] = NewCommentGroup(fset, cg)
	}

	c.comments[filename] = comments

	return nil
}

// AddFile adds file comments to the internal cache. If the filename already
// exists in the cache, the function does nothing.
func (c *CommentsCache) AddFile(fset *token.FileSet, file *ast.File) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// ToDo: add check if file belongs to provided fset.
	filename := fset.PositionFor(file.Pos(), true).Filename

	if c.comments[filename] != nil {
		return
	}

	comments := make([]CommentGroup, len(file.Comments))
	for i, cg := range file.Comments {
		comments[i] = NewCommentGroup(fset, cg)
	}

	c.comments[filename] = comments
}

// Comments returns a list of all comments in the file. If the file is not parsed
// yet, a nil slice is returned.
func (c *CommentsCache) Comments(filename string) []CommentGroup {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.comments[filename]
}

// CommentsForPosition retrieves comment groups related to provided position.
// Comment group considered to be related in case it located on previous or same
// row.
func (c *CommentsCache) CommentsForPosition(p token.Position, exclude ...token.Position) []CommentGroup {
	comments := c.Comments(p.Filename)
	if comments == nil {
		return nil
	}

	// we expect to get 0 to 2 comments (none, prev|same, prev & same)
	result := make([]CommentGroup, 0, 2) //nolint:mnd

	for _, cg := range comments {
		if positionEndRelatesTo(cg.End, p) && !positionEndRelatesTo(cg.End, exclude...) {
			result = append(result, cg)
		}
	}

	return result
}

func positionEndRelatesTo(p token.Position, references ...token.Position) bool {
	for _, rp := range references {
		if rp.Line == p.Line || rp.Line-1 == p.Line {
			return true
		}
	}

	return false
}
