package file

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sync"
)

var ErrNotFound = errors.New("typename declaration not found")

type ASTCache struct {
	Mode parser.Mode

	files map[string]*ast.File `exhaustruct:"optional"`
	mu    sync.RWMutex         `exhaustruct:"optional"`
}

// AddFiles adds a list of AST files to the cache.
func (c *ASTCache) AddFiles(fs *token.FileSet, files ...*ast.File) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.files == nil {
		c.files = make(map[string]*ast.File)
	}

	for _, f := range files {
		c.files[fs.PositionFor(f.Pos(), true).Filename] = f
	}
}

// Get returns an AST file for a given path. In case if a file is not found, it
// creates a new one by parsing it with [ASTCache.Mode] mode.
func (c *ASTCache) Get(fs *token.FileSet, path string) (*ast.File, error) {
	c.mu.RLock()
	f, ok := c.files[path]
	c.mu.RUnlock()

	if ok {
		return f, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.files == nil {
		c.files = make(map[string]*ast.File)
	}

	f, err := parser.ParseFile(fs, path, nil, c.Mode)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	c.files[path] = f

	return f, nil
}

// FindTypeNameGenDecl returns a GenDecl for a given type name. In case if a
// declaration is not found, it returns an error.
func (c *ASTCache) FindTypeNameGenDecl(fs *token.FileSet, tn *types.TypeName) (*ast.GenDecl, error) {
	typPos := fs.PositionFor(tn.Pos(), true)

	f, err := c.Get(fs, typPos.Filename)
	if err != nil {
		return nil, err
	}

	obj, ok := f.Scope.Objects[tn.Name()]
	if !ok {
		return nil, ErrNotFound
	}

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		// we can bypass several checks as GenDecl always consists of at least one spec
		// and type name is always the first one, at leas all cases I'm aware of this
		// logic works
		if gd.Specs[0] == obj.Decl {
			return gd, nil
		}
	}

	return nil, ErrNotFound
}
