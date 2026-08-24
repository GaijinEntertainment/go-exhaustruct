package analyzer

import (
	"fmt"
	"go/token"
	"strings"
	"testing"
)

// benchDiamondLayers stacks enough diamonds that the paths through them
// outnumber the declarations by six orders of magnitude.
const benchDiamondLayers = 20

// BenchmarkCoreStruct_LayeredDiamonds resolves a constraint whose interfaces
// form stacked diamonds: every layer embeds two interfaces that both embed the
// layer below it. That leaves 2^n paths down to the terms and only 3n+1
// interfaces declaring them, so a walk paying per path cannot finish while a
// walk paying per interface is immediate.
func BenchmarkCoreStruct_LayeredDiamonds(b *testing.B) {
	pkg := checkSource(b, layeredDiamonds(benchDiamondLayers))
	tp := typeParamOf(b, pkg, "fDiamond")
	fset := token.NewFileSet()

	b.ReportAllocs()

	for b.Loop() {
		benchCore = coreStruct(fset, nil, tp)
	}
}

//nolint:gochecknoglobals // sink, so the resolved core cannot be optimized away
var benchCore any

// layeredDiamonds writes a package where layer i embeds two interfaces that
// both embed layer i-1, and the bottom layer names the struct terms.
func layeredDiamonds(layers int) string {
	var b strings.Builder

	b.WriteString("package p\n\ntype l0 interface{ ~struct{ A int; B string } }\n")

	for i := 1; i <= layers; i++ {
		fmt.Fprintf(&b, "type l%da interface{ l%d }\n", i, i-1)
		fmt.Fprintf(&b, "type l%db interface{ l%d }\n", i, i-1)
		fmt.Fprintf(&b, "type l%d interface{ l%da; l%db }\n", i, i, i)
	}

	fmt.Fprintf(&b, "\nfunc fDiamond[T l%d]() {}\n", layers)

	return b.String()
}
