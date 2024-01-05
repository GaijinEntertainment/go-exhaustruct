package file

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	ErrNotFound = errors.New("typename declaration not found")
)

type ASTCache struct {
	Mode parser.Mode
	FS   *token.FileSet `exhaustruct:"optional"`

	files map[string]*ast.File `exhaustruct:"optional"`
	mu    sync.RWMutex         `exhaustruct:"optional"`

	Hit  atomic.Int64 `exhaustruct:"optional"`
	Miss atomic.Int64 `exhaustruct:"optional"`
}

// AddFiles adds a list of AST files to the cache.
func (c *ASTCache) AddFiles(files ...*ast.File) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.files == nil {
		c.files = make(map[string]*ast.File)
	}

	for _, f := range files {
		c.files[c.FS.Position(f.Pos()).Filename] = f
	}
}

// Get returns an AST file for a given path. In case if a file is not found, it
// creates a new one by parsing it with [ASTCache.Mode] mode.
func (c *ASTCache) Get(path string) (*ast.File, error) {
	c.mu.RLock()
	f, ok := c.files[path]
	c.mu.RUnlock()

	if ok {
		c.Hit.Add(1)
		return f, nil
	}

	c.Miss.Add(1)
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.files == nil {
		c.files = make(map[string]*ast.File)
	}

	f, err := parser.ParseFile(c.FS, path, nil, c.Mode)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	c.files[path] = f

	return f, nil
}

func (c *ASTCache) RelatedComments(node ast.Node) ([]*ast.CommentGroup, error) {
	nodeStart := c.FS.Position(node.Pos())

	if strings.HasPrefix(nodeStart.Filename, "$GOROOT/") {
		// stdlib is most likely unreachable for standalone executables
		// also - there is zero chance that we will find any exhaustruct comments
		// within stdlib, therefore there is literally no reason to parse it
		return nil, ErrNotFound
	}

	nodeEnd := c.FS.Position(node.End())

	var relatedComments []*ast.CommentGroup

	f, err := c.Get(nodeStart.Filename)
	if err != nil {
		return nil, err
	}

	for _, group := range f.Comments {
		groupStart := c.FS.Position(group.Pos())
		groupEnd := c.FS.Position(group.End())

		if (groupEnd.Line == nodeStart.Line-1) || // previous line
			(groupStart.Line == nodeEnd.Line && groupStart.Column > nodeEnd.Column) { // end of same line
			relatedComments = append(relatedComments, group)
		}
	}

	return relatedComments, nil
}

// FindIdentGenDecl returns a GenDecl for a given identifier. In case if the
// declaration is not found, it returns [ErrNotFound].
func (c *ASTCache) FindIdentGenDecl(ident *ast.Ident) (*ast.GenDecl, error) {
	typPos := c.FS.Position(ident.Pos())

	if strings.HasPrefix(typPos.Filename, "$GOROOT/") {
		return nil, ErrNotFound
	}

	f, err := c.Get(typPos.Filename)
	if err != nil {
		return nil, err
	}

	if gd := findTypeGenDeclByName(f, ident.Name); gd != nil {
		return gd, nil
	}

	return nil, ErrNotFound
}

// FindTypeNameGenDecl returns a GenDecl for a given type name. In case if the
// declaration is not found, it returns [ErrNotFound].
func (c *ASTCache) FindTypeNameGenDecl(typ *types.TypeName) (*ast.GenDecl, error) {
	typPos := c.FS.Position(typ.Pos())

	if strings.HasPrefix(typPos.Filename, "$GOROOT/") {
		return nil, ErrNotFound
	}

	f, err := c.Get(typPos.Filename)
	if err != nil {
		return nil, err
	}

	if gd := findTypeGenDeclByName(f, typ.Name()); gd != nil {
		return gd, nil
	}

	return nil, ErrNotFound
}

func findTypeGenDeclByName(f *ast.File, name string) *ast.GenDecl {
	obj, ok := f.Scope.Objects[name]
	if !ok {
		return nil
	}

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if gd.Tok != token.TYPE {
			continue
		}

		// we can bypass several checks as GenDecl always consists of at least one spec
		// and type name is always the first one, at least all cases I'm aware of this
		// logic works
		if gd.Specs[0] == obj.Decl {
			return gd
		}
	}

	return nil
}
