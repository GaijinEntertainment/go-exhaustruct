package fileutil

import (
	"go/ast"
	"go/parser"
	"go/token"

	"dev.gaijin.team/go/golib/e"
	"dev.gaijin.team/go/golib/fields"
)

// Parser provides Go source file parsing with comments.
type Parser struct {
	reader *Reader
}

// NewParser creates a Parser that uses the provided Reader for file access.
func NewParser(reader *Reader) *Parser {
	return &Parser{reader: reader}
}

// ParseFile parses a Go source file and returns its AST with comments.
// Only the specified file is parsed - no dependencies are resolved.
func (p *Parser) ParseFile(fset *token.FileSet, filename string) (*ast.File, error) {
	content, err := p.reader.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	file, err := parser.ParseFile(fset, filename, content, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, e.NewFrom("parse file", err, fields.F("filename", filename))
	}

	return file, nil
}
