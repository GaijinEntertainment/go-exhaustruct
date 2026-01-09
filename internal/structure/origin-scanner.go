package structure

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"

	"dev.gaijin.team/go/exhaustruct/v4/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v4/internal/cache"
)

type TypeOrigin uint8

const (
	OriginUnknown TypeOrigin = iota
	OriginStruct
	OriginAlias
	OriginDerived
)

type fileOrigins map[string]TypeOrigin

// OriginScanner extracts type origin information from AST files.
// Registers with FileParser to process files as they are parsed.
// Thread-safe.
type OriginScanner struct {
	parser *astutil.FileParser
	cache  *cache.Cache[string, fileOrigins]
}

const originCachePrealloc = 64

func NewOriginScanner(parser *astutil.FileParser) *OriginScanner {
	o := &OriginScanner{
		parser: parser,
		cache:  cache.New[string, fileOrigins](originCachePrealloc),
	}

	parser.OnFileParsed(o.onFileParsed)

	return o
}

func (o *OriginScanner) onFileParsed(
	fset *token.FileSet,
	file *ast.File,
) []analysis.Diagnostic {
	filename := fset.Position(file.Pos()).Filename

	origins := extractTypeOrigins(file)

	o.cache.Set(filename, origins)

	return nil
}

func (o *OriginScanner) ProcessFiles(fset *token.FileSet, files ...*ast.File) {
	o.parser.ProcessFiles(fset, files...)
}

// Lookup returns the type origin for a named type in the given file.
// Triggers on-demand parsing if file is not cached.
func (o *OriginScanner) Lookup(
	fset *token.FileSet,
	filename string,
	typeName string,
) TypeOrigin {
	if filename == "" || typeName == "" {
		return OriginUnknown
	}

	if origins, ok := o.cache.Get(filename); ok {
		return origins[typeName]
	}

	o.parser.ProcessFilename(fset, filename)

	if origins, ok := o.cache.Get(filename); ok {
		return origins[typeName]
	}

	return OriginUnknown
}

func (o *OriginScanner) Stats() (hits, misses, size uint64) {
	return o.cache.Stats()
}

func extractTypeOrigins(file *ast.File) fileOrigins {
	origins := make(fileOrigins)

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			origins[ts.Name.Name] = classifyTypeSpec(ts)
		}
	}

	return origins
}

func classifyTypeSpec(ts *ast.TypeSpec) TypeOrigin {
	if ts.Assign != token.NoPos {
		return OriginAlias
	}

	if _, isStruct := ts.Type.(*ast.StructType); isStruct {
		return OriginStruct
	}

	return OriginDerived
}
