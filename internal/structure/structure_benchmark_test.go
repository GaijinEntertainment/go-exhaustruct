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
// a map built with the type's metadata, so the cost of a literal grows with the
// keys it writes rather than with the depth of the tree behind them. The first
// case embeds nothing and stands for the common literal, which must not pay for
// any of this.
func BenchmarkSkippedFields_Promoted(b *testing.B) {
	cases := []struct{ depth, width int }{{1, 10}, {2, 5}, {3, 10}, {5, 30}}

	for _, c := range cases {
		b.Run(fmt.Sprintf("depth%d_width%d", c.depth, c.width), func(b *testing.B) {
			strct, lit := benchFixture(b, c.depth, c.width)

			b.ReportAllocs()

			for b.Loop() {
				_ = strct.SkippedFields(lit, benchPkgPath, promotedKeys)
			}
		})
	}
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
