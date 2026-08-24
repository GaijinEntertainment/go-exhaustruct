package structure_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"dev.gaijin.team/go/exhaustruct/v5/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v5/internal/directive"
	"dev.gaijin.team/go/exhaustruct/v5/internal/structure"
)

// BenchmarkSkippedFields_Promoted measures a literal that names promoted fields
// rather than the embedded fields carrying them. Promotion is resolved against
// an index built with the type's metadata, and the literal's keys are arranged
// once, so the cost of a literal grows with the keys it writes rather than with
// the depth of the tree behind them. The first case embeds nothing and stands
// for the common literal, which must not pay for any of this.
func BenchmarkSkippedFields_Promoted(b *testing.B) {
	cases := []struct{ depth, width int }{{1, 10}, {2, 5}, {3, 10}, {5, 30}}

	for _, c := range cases {
		b.Run(fmt.Sprintf("depth%d_width%d", c.depth, c.width), func(b *testing.B) {
			strct, lit := benchFixture(b, c.depth, c.width)

			b.ReportAllocs()

			for b.Loop() {
				benchSkipped = strct.SkippedFields(lit, benchPkgPath, promotedKeys)
			}
		})
	}
}

// BenchmarkSkippedFields_PartialAtEveryLevel measures a literal that names one
// field of every level, so the descent enters each of them and each returns
// fields of its own. What every level reports has to reach the caller once,
// not be copied again at each step of the unwind.
func BenchmarkSkippedFields_PartialAtEveryLevel(b *testing.B) {
	for _, depth := range []int{2, 4, 8, 16} {
		b.Run(fmt.Sprintf("depth%d", depth), func(b *testing.B) {
			const width = 10

			strct, _ := benchFixture(b, depth, width)

			keys := make([]string, 0, depth)
			for level := 1; level <= depth; level++ {
				keys = append(keys, fmt.Sprintf("f%d_0: 1", level))
			}

			lit := benchParse(b, "L1{"+strings.Join(keys, ", ")+"}")

			b.ReportAllocs()

			for b.Loop() {
				benchSkipped = strct.SkippedFields(lit, benchPkgPath, promotedKeys)
			}
		})
	}
}

// benchParse parses one composite literal written out in full.
func benchParse(b *testing.B, src string) *ast.CompositeLit {
	b.Helper()

	expr, err := parser.ParseExpr(src)
	if err != nil {
		b.Fatal(err)
	}

	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		b.Fatalf("expected a composite literal, got %T", expr)
	}

	return lit
}

const benchPkgPath = "example.com/dep"

// benchFixture builds a chain of depth structs, each embedding the next and
// declaring width fields of its own, and a literal naming every field in the
// chain by the name a literal of the outermost type writes.
func benchFixture(b *testing.B, depth, width int) (*structure.Struct, *ast.CompositeLit) {
	b.Helper()

	fset := token.NewFileSet()
	pos := fset.AddFile("bench.go", -1, 1<<20).Pos(0)
	pkg := types.NewPackage(benchPkgPath, "dep")

	var (
		inner     *types.Named
		innerName string
	)

	for level := depth; level >= 1; level-- {
		fields := make([]*types.Var, 0, width+1)

		if inner != nil {
			fields = append(fields, types.NewField(pos, pkg, innerName, inner, true))
		}

		for i := range width {
			name := fmt.Sprintf("f%d_%d", level, i)

			fields = append(fields, types.NewField(pos, pkg, name, types.Typ[types.Int], false))
		}

		strct := types.NewStruct(fields, nil)
		typeName := types.NewTypeName(pos, pkg, fmt.Sprintf("L%d", level), nil)
		named := types.NewNamed(typeName, strct, nil)

		if level > 1 {
			inner, innerName = named, typeName.Name()

			continue
		}

		fp := astutil.NewFileParser()
		proc := structure.NewProcessor(directive.NewScanner(fp), structure.NewOriginScanner(fp))

		resolved := proc.ResolveStruct(fset, typeName, strct, pos, pkg)
		if resolved == nil {
			b.Fatal("fixture did not resolve")
		}

		return resolved, benchLiteral(b, depth, width)
	}

	b.Fatal("depth must be at least one")

	return nil, nil
}

func benchLiteral(b *testing.B, depth, width int) *ast.CompositeLit {
	b.Helper()

	keys := make([]string, 0, depth*width)

	for level := 1; level <= depth; level++ {
		for i := range width {
			keys = append(keys, fmt.Sprintf("f%d_%d: 1", level, i))
		}
	}

	expr, err := parser.ParseExpr("L1{" + strings.Join(keys, ", ") + "}")
	if err != nil {
		b.Fatal(err)
	}

	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		b.Fatalf("expected a composite literal, got %T", expr)
	}

	return lit
}

//nolint:gochecknoglobals // sink, so the skipped fields cannot be optimized away
var benchSkipped []structure.Field

// BenchmarkResolveStruct_SharedEmbedding measures the metadata of a type whose
// embedding graph reaches the layer below it through two wrappers, so the paths
// through it double with every layer while the declarations grow by three. The
// walk opens each struct once, so the cost has to follow the declarations.
func BenchmarkResolveStruct_SharedEmbedding(b *testing.B) {
	for _, layers := range []int{4, 8, 12, 16} {
		b.Run(fmt.Sprintf("layers%d", layers), func(b *testing.B) {
			fset := token.NewFileSet()
			pos := fset.AddFile("bench.go", -1, 1<<20).Pos(0)
			pkg := types.NewPackage(benchPkgPath, "dep")
			top := sharedEmbeddingFixture(pos, pkg, layers)

			strct, ok := top.Underlying().(*types.Struct)
			if !ok {
				b.Fatal("fixture is not a struct")
			}

			b.ReportAllocs()

			for b.Loop() {
				// A processor answers a type once and remembers it, so each run
				// needs one that has not seen this type. Building it is setup,
				// and it allocates several times what the walk does.
				b.StopTimer()

				fp := astutil.NewFileParser()
				proc := structure.NewProcessor(directive.NewScanner(fp), structure.NewOriginScanner(fp))

				b.StartTimer()

				if proc.ResolveStruct(fset, top.Obj(), strct, pos, pkg) == nil {
					b.Fatal("fixture did not resolve")
				}
			}
		})
	}
}

// sharedEmbeddingFixture builds layers of L{n}a and L{n}b, each embedding
// L{n-1}, under an L{n} embedding both.
func sharedEmbeddingFixture(pos token.Pos, pkg *types.Package, layers int) *types.Named {
	named := func(name string, fields ...*types.Var) *types.Named {
		typeName := types.NewTypeName(pos, pkg, name, nil)

		return types.NewNamed(typeName, types.NewStruct(fields, nil), nil)
	}

	embeds := func(inner *types.Named) *types.Var {
		return types.NewField(pos, pkg, inner.Obj().Name(), inner, true)
	}

	level := named("L0", types.NewField(pos, pkg, "F0", types.Typ[types.Int], false))

	for i := 1; i <= layers; i++ {
		left := named(fmt.Sprintf("L%da", i), embeds(level))
		right := named(fmt.Sprintf("L%db", i), embeds(level))

		level = named(fmt.Sprintf("L%d", i), embeds(left), embeds(right))
	}

	return level
}
