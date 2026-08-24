package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_coreStruct covers the constraint shapes a literal of a type parameter is
// written against. A constraint without one core type carries no literal at
// all, so no compiling fixture reaches that path.
func Test_coreStruct(t *testing.T) {
	t.Parallel()

	const src = `package p

type direct interface{ ~struct{ A int; B string } }
type nested interface{ direct }
type deeper interface{ nested }
type twoTerms interface{ ~struct{ A int } | ~struct{ B int } }
type s1 struct{ A int }
type s2 struct{ A int }
type sameUnderlying interface{ s1 | s2 }
type methodsOnly interface{ String() string }
type empty interface{}
type methodAndTerms interface{ ~struct{ A int; B string }; String() string }
type embeddedMethods interface{ ~struct{ A int; B string }; methodsOnly }
type mixedTerms interface{ ~struct{ A int } | ~int }

func fDirect[T direct]()               {}
func fNested[T nested]()               {}
func fDeeper[T deeper]()               {}
func fTwoTerms[T twoTerms]()           {}
func fSameUnderlying[T sameUnderlying]() {}
func fMethodsOnly[T methodsOnly]()     {}
func fEmpty[T empty]()                 {}
func fMethodAndTerms[T methodAndTerms]() {}
func fEmbeddedMethods[T embeddedMethods]() {}
func fMixedTerms[T mixedTerms]()       {}
`

	pkg := checkSource(t, src)

	tests := []struct {
		name      string
		give      string
		wantCore  bool
		wantField string
	}{
		{"terms named directly", "fDirect", true, "A"},
		{"terms inherited from an embedded interface", "fNested", true, "A"},
		{"terms inherited two levels up", "fDeeper", true, "A"},
		{"distinct terms sharing an underlying struct", "fSameUnderlying", true, "A"},
		{"a method declared beside the terms", "fMethodAndTerms", true, "A"},
		{"a method-only interface embedded beside the terms", "fEmbeddedMethods", true, "A"},
		{"terms disagree", "fTwoTerms", false, ""},
		{"a term that is not a struct", "fMixedTerms", false, ""},
		{"no terms, only methods", "fMethodsOnly", false, ""},
		{"no terms at all", "fEmpty", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			core := coreStruct(token.NewFileSet(), nil, typeParamOf(t, pkg, tt.give))

			if !tt.wantCore {
				assert.Nil(t, core)

				return
			}

			require.NotNil(t, core)
			assert.Equal(t, tt.wantField, core.Field(0).Name())
		})
	}
}

// Test_canNamePromotedIn covers the version gate on promoted keys. An unknown
// version has to read as the newest one: go/types allows every feature under
// it, so a literal it accepted with promoted keys would otherwise be reported
// for naming them.
func Test_canNamePromotedIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		give string
		want bool
	}{
		{"go1.26", false},
		{"go1.27", true},
		{"go1.28", true},
		{"", true},
		{"1.27", true},
	}

	for _, tt := range tests {
		t.Run(tt.give, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, canNamePromotedIn(tt.give))
		})
	}
}

func checkSource(tb testing.TB, src string) *types.Package {
	tb.Helper()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "constraints.go", src, 0)
	require.NoError(tb, err)

	conf := types.Config{}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}

	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	require.NoError(tb, err, "type-check failed")

	return pkg
}

// typeParamOf returns the single type parameter of the named function.
func typeParamOf(tb testing.TB, pkg *types.Package, fn string) *types.TypeParam {
	tb.Helper()

	obj, ok := pkg.Scope().Lookup(fn).(*types.Func)
	require.True(tb, ok)

	sig, ok := obj.Type().(*types.Signature)
	require.True(tb, ok)

	params := sig.TypeParams()
	require.Equal(tb, 1, params.Len())

	return params.At(0)
}
