package structure

import (
	"go/token"
	"go/types"

	"dev.gaijin.team/go/exhaustruct/v5/internal/cache"
	"dev.gaijin.team/go/exhaustruct/v5/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v5/internal/pattern"
)

// structKey identifies a struct type in the struct cache.
//
// A source position cannot serve as that identity. Export data is not required
// to carry a faithful one: gcimporter discards the column outright and clamps
// any line past 64Ki to 1, so distinct declarations inside one dependency file
// collapse onto a single token.Position. go/types objects, in contrast, are
// unique per declaration per loaded package.
type structKey struct {
	// name is the type's TypeName, nil for anonymous structs.
	name *types.TypeName
	// strct is the underlying struct type. It alone identifies an anonymous
	// struct, and guards against a shared TypeName for a named one.
	strct *types.Struct
	// callerPkg is set for anonymous structs only, whose metadata is derived
	// from the package that uses them. Named types are caller-independent and
	// share one entry across every package that references them.
	callerPkg *types.Package
}

type Processor struct {
	directives  *directive.Scanner
	origins     *OriginScanner
	fieldsCache *cache.Cache[*types.Struct, structFields]
	structCache *cache.Cache[structKey, *Struct]

	enforce    pattern.List //exhaustruct:optional
	ignore     pattern.List //exhaustruct:optional
	optional   pattern.List //exhaustruct:optional
	allowEmpty pattern.List //exhaustruct:optional
}

type Option func(*Processor)

func WithEnforce(patterns pattern.List) Option {
	return func(p *Processor) { p.enforce = patterns }
}

func WithIgnore(patterns pattern.List) Option {
	return func(p *Processor) { p.ignore = patterns }
}

func WithOptional(patterns pattern.List) Option {
	return func(p *Processor) { p.optional = patterns }
}

func WithAllowEmpty(patterns pattern.List) Option {
	return func(p *Processor) { p.allowEmpty = patterns }
}

const cachePreallocSize = 64

func NewProcessor(directives *directive.Scanner, origins *OriginScanner, opts ...Option) *Processor {
	p := &Processor{
		directives:  directives,
		origins:     origins,
		fieldsCache: cache.New[*types.Struct, structFields](cachePreallocSize),
		structCache: cache.New[structKey, *Struct](cachePreallocSize),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Directives returns the directive scanner the processor was constructed
// with, shared so callers can resolve use-site directives against the same
// cache.
func (p *Processor) Directives() *directive.Scanner {
	return p.directives
}

// ResolveStruct returns Struct metadata for the given type.
// Type resolution (pointers, aliases) is done by the caller.
//
// Parameters:
//   - typeName: the type's TypeName, or nil for anonymous structs
//   - strct: the underlying struct type (required)
//   - pos: position of type definition (from analyzer's AST inspection)
//   - callerPkg: package context, used for anonymous struct path
func (p *Processor) ResolveStruct(
	fset *token.FileSet,
	typeName *types.TypeName,
	strct *types.Struct,
	pos token.Pos,
	callerPkg *types.Package,
) *Struct {
	if strct == nil {
		return nil
	}

	key := structKey{name: typeName, strct: strct, callerPkg: nil}
	if typeName == nil {
		key.callerPkg = callerPkg
	}

	// Check cache before allocating
	if cached, ok := p.structCache.Get(key); ok {
		return cached
	}

	// Positions are taken unadjusted: they locate the declaration on disk, and
	// //line directives point at files that hold no Go source.
	s := p.buildStruct(typeName, fset.PositionFor(pos, false), callerPkg)

	p.populateFields(fset, s, strct)
	p.resolveStructOrigin(fset, s)
	p.resolveStructDirectives(fset, s)
	p.matchStructPatterns(s)

	p.structCache.Set(key, s)

	return s
}

// buildStruct creates Struct metadata from type info.
func (*Processor) buildStruct(typeName *types.TypeName, pos token.Position, callerPkg *types.Package) *Struct {
	if typeName != nil {
		pkg := typeName.Pkg()

		return &Struct{
			Name:        typeName.Name(),
			FullPath:    pkg.Path() + "." + typeName.Name(),
			PackageName: pkg.Name(),
			Position:    pos,
		}
	}

	// Anonymous struct
	return &Struct{
		Name:        AnonymousName,
		FullPath:    callerPkg.Path() + "." + AnonymousName,
		PackageName: callerPkg.Name(),
		Position:    pos,
	}
}

func (p *Processor) getStructFields(fset *token.FileSet, strct *types.Struct) structFields {
	if fields, ok := p.fieldsCache.Get(strct); ok {
		return fields
	}

	fields := p.resolveStructFields(fset, strct)

	p.fieldsCache.Set(strct, fields)

	return fields
}

func (p *Processor) resolveStructFields(fset *token.FileSet, strct *types.Struct) structFields {
	result := structFields{
		packagePath: "",
		fields:      make([]fieldInfo, 0, strct.NumFields()),
	}

	for f := range strct.Fields() {
		if result.packagePath == "" && f.Pkg() != nil {
			result.packagePath = f.Pkg().Path()
		}

		field := fieldInfo{
			name:     f.Name(),
			exported: f.Exported(),
		}

		if p.directives != nil {
			dirs := p.directives.Lookup(fset, fset.PositionFor(f.Pos(), false))

			field.enforced = dirs.Contains(directive.Enforce)
			field.optional = dirs.Contains(directive.Optional)
		}

		result.fields = append(result.fields, field)
	}

	return result
}

func (p *Processor) populateFields(fset *token.FileSet, s *Struct, strct *types.Struct) {
	resolved := p.getStructFields(fset, strct)

	// Fields are external when declared in a different package than the struct type.
	// This happens for derived types and aliases from external packages.
	//
	// Rationale behind that filtering is that noone except package that has declared
	// the struct can access unexported fields, therefore we can simply filter them
	// out to save up on storage. Usage of derived type from the package of structure
	// definition is simply impossible since it will cause import cycle - thus, such
	// filtering is safe.
	fieldsExternal := resolved.packagePath != s.PackagePath()

	s.Fields = Fields{
		PackagePath: resolved.packagePath,
		Items:       make([]Field, 0, len(resolved.fields)),
	}

	for _, sf := range resolved.fields {
		if fieldsExternal && !sf.exported {
			continue
		}

		fieldPath := s.FullPath + "#" + sf.name

		s.Fields.Items = append(s.Fields.Items, Field{
			Name:            sf.name,
			Exported:        sf.exported,
			Enforced:        sf.enforced,
			Optional:        sf.optional,
			PatternEnforced: p.enforce.MatchFullString(fieldPath),
			PatternOptional: p.optional.MatchFullString(fieldPath),
		})
	}
}

func (p *Processor) resolveStructOrigin(fset *token.FileSet, s *Struct) {
	if !s.Position.IsValid() || s.Name == AnonymousName {
		return
	}

	origin := p.origins.Lookup(fset, s.Position.Filename, s.Name)

	s.IsAlias = origin == OriginAlias
	s.IsDerived = origin == OriginDerived
}

func (p *Processor) resolveStructDirectives(fset *token.FileSet, s *Struct) {
	if p.directives == nil || !s.Position.IsValid() {
		return
	}

	dirs := p.directives.Lookup(fset, s.Position)

	s.Enforced = dirs.Contains(directive.Enforce)
	s.Ignored = dirs.Contains(directive.Ignore)
	s.Optional = dirs.Contains(directive.Optional)
}

func (p *Processor) matchStructPatterns(s *Struct) {
	s.PatternEnforced = p.enforce.MatchFullString(s.FullPath)
	s.PatternIgnored = p.ignore.MatchFullString(s.FullPath)
	s.PatternOptional = p.optional.MatchFullString(s.FullPath)
	s.AllowEmptyDecl = p.allowEmpty.MatchFullString(s.FullPath)
}

func (p *Processor) Stats() (hits, misses, size uint64) {
	return p.structCache.Stats()
}
