package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/internal/fields"
)

type Visitor struct {
	analyzer *analyzer
	pass     *analysis.Pass
}

func (v *Visitor) Visit(n ast.Node, push bool, stack []ast.Node) bool {
	if !push {
		return true
	}

	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		// this should never happen, but better be prepared
		return true
	}

	structTyp, typeInfo, ok := v.getStructType(lit)
	if !ok {
		return true
	}

	if len(lit.Elts) == 0 {
		if ret, ok := stackParentIsReturn(stack); ok {
			if v.returnContainsNonNilError(ret) {
				// it is okay to return uninitialized structure in case struct's direct parent is
				// a return statement containing non-nil error
				//
				// we're unable to check if returned error is custom, but at least we're able to
				// cover str [error] type.
				return true
			}
		}
	}

	pos, msg := v.processStruct(lit, structTyp, typeInfo)
	if pos != nil {
		v.pass.Reportf(*pos, msg)
	}

	return true
}

func (v *Visitor) getStructType(lit *ast.CompositeLit) (*types.Struct, *TypeInfo, bool) {
	switch typ := v.pass.TypesInfo.TypeOf(lit).(type) {
	case *types.Named: // named type
		if structTyp, ok := typ.Underlying().(*types.Struct); ok {
			pkg := typ.Obj().Pkg()
			ti := TypeInfo{
				Name:        typ.Obj().Name(),
				PackageName: pkg.Name(),
				PackagePath: pkg.Path(),
			}

			return structTyp, &ti, true
		}

		return nil, nil, false

	case *types.Struct: // anonymous struct
		ti := TypeInfo{
			Name:        "<anonymous>",
			PackageName: v.pass.Pkg.Name(),
			PackagePath: v.pass.Pkg.Path(),
		}

		return typ, &ti, true

	default:
		return nil, nil, false
	}
}

func stackParentIsReturn(stack []ast.Node) (*ast.ReturnStmt, bool) {
	// it is safe to skip boundary check, since stack always has at least one element
	// - whole file.
	ret, ok := stack[len(stack)-2].(*ast.ReturnStmt)

	return ret, ok
}

func (v *Visitor) returnContainsNonNilError(ret *ast.ReturnStmt) bool {
	// errors are mostly located at the end of return statement, so we're starting
	// from the end.
	for i := len(ret.Results) - 1; i >= 0; i-- {
		if v.pass.TypesInfo.TypeOf(ret.Results[i]).String() == "error" {
			return true
		}
	}

	return false
}

func (v *Visitor) processStruct(
	lit *ast.CompositeLit,
	structTyp *types.Struct,
	info *TypeInfo,
) (*token.Pos, string) {
	if !v.shouldProcessType(info) {
		return nil, ""
	}

	// unnamed structures are only defined in same package, along with types that has
	// prefix identical to current package name.
	isSamePackage := info.PackagePath == v.pass.Pkg.Path()

	if f := v.litSkippedFields(lit, structTyp, !isSamePackage); len(f) > 0 {
		pos := lit.Pos()

		if len(f) == 1 {
			return &pos, fmt.Sprintf("%s is missing field %s", info.ShortString(), f.String())
		}

		return &pos, fmt.Sprintf("%s is missing fields %s", info.ShortString(), f.String())
	}

	return nil, ""
}

// shouldProcessType returns true if type should be processed basing off include
// and exclude patterns, defined though constructor and\or flags.
func (v *Visitor) shouldProcessType(info *TypeInfo) bool {
	if len(v.analyzer.include) == 0 && len(v.analyzer.exclude) == 0 {
		return true
	}

	name := info.String()

	v.analyzer.typeProcessingNeedMu.RLock()
	res, ok := v.analyzer.typeProcessingNeed[name]
	v.analyzer.typeProcessingNeedMu.RUnlock()

	if !ok {
		v.analyzer.typeProcessingNeedMu.Lock()
		res = true

		if v.analyzer.include != nil && !v.analyzer.include.MatchFullString(name) {
			res = false
		}

		if res && v.analyzer.exclude != nil && v.analyzer.exclude.MatchFullString(name) {
			res = false
		}

		v.analyzer.typeProcessingNeed[name] = res
		v.analyzer.typeProcessingNeedMu.Unlock()
	}

	return res
}

//revive:disable-next-line:unused-receiver
func (v *Visitor) litSkippedFields(
	lit *ast.CompositeLit,
	typ *types.Struct,
	onlyExported bool,
) fields.StructFields {
	v.analyzer.fieldsCacheMu.RLock()
	f, ok := v.analyzer.fieldsCache[typ]
	v.analyzer.fieldsCacheMu.RUnlock()

	if !ok {
		v.analyzer.fieldsCacheMu.Lock()
		f = fields.NewStructFields(typ)
		v.analyzer.fieldsCache[typ] = f
		v.analyzer.fieldsCacheMu.Unlock()
	}

	return f.SkippedFields(lit, onlyExported)
}

type TypeInfo struct {
	Name        string
	PackageName string
	PackagePath string
}

func (t TypeInfo) String() string {
	return t.PackagePath + "." + t.Name
}

func (t TypeInfo) ShortString() string {
	return t.PackageName + "." + t.Name
}
