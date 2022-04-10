package analyzer

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var (
	ErrEmptyPattern = errors.New("pattern can't be empty")
)

type analyzer struct {
	include PatternsList
	exclude PatternsList
}

// MustNewAnalyzer returns a go/analysis-compatible analyzer.
//   -i arguments adds include patterns
//   -e arguments adds exclude patterns
func MustNewAnalyzer(include []string, exclude []string) *analysis.Analyzer {
	a := analyzer{
		include: mustNewPatternsList(include),
		exclude: mustNewPatternsList(exclude),
	}

	return &analysis.Analyzer{
		Name:     "exhaustruct",
		Doc:      "Checks if all structure fields are initialized",
		Run:      a.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Flags:    a.newFlagSet(),
	}
}

func (a *analyzer) newFlagSet() flag.FlagSet {
	fs := flag.NewFlagSet("exhaustruct flags", flag.PanicOnError)

	fs.Var(
		&reListVar{values: &a.include},
		"i",
		"Regular expression to match struct packages and names, can receive multiple flags",
	)
	fs.Var(
		&reListVar{values: &a.exclude},
		"e",
		"Regular expression to exclude struct packages and names, can receive multiple flags",
	)

	return *fs
}

func (a *analyzer) run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector) //nolint:forcetypeassert

	nodeTypes := []ast.Node{
		(*ast.CompositeLit)(nil),
		(*ast.ReturnStmt)(nil),
	}

	insp.Preorder(nodeTypes, a.newVisitor(pass))

	return nil, nil //nolint:nilnil
}

//nolint:funlen,cyclop
func (a *analyzer) newVisitor(pass *analysis.Pass) func(node ast.Node) {
	var ret *ast.ReturnStmt

	return func(node ast.Node) {
		if retLit, ok := node.(*ast.ReturnStmt); ok {
			// save return statement for future (to detect error-containing returns)
			ret = retLit

			return
		}

		lit, _ := node.(*ast.CompositeLit)
		if lit.Type == nil {
			// we're not interested in non-typed literals
			return
		}

		typ := pass.TypesInfo.TypeOf(lit.Type)
		if typ == nil {
			return
		}

		strct, ok := typ.Underlying().(*types.Struct)
		if !ok {
			// we also not interested in non-structure literals
			return
		}

		strctName := exprName(lit.Type)
		if strctName == "" {
			return
		}

		if len(a.include) > 0 {
			if !a.include.MatchesAny(typ.String()) {
				return
			}
		}

		if a.exclude.MatchesAny(typ.String()) {
			return
		}

		if len(lit.Elts) == 0 && ret != nil {
			if ret.End() < lit.Pos() {
				// we're outside last return statement
				ret = nil
			} else if returnContainsLiteral(ret, lit) && returnContainsError(ret, pass) {
				// we're okay with empty literals in return statements with non-nil errors, like
				// `return my.Struct{}, fmt.Errorf("non-nil error!")`
				return
			}
		}

		missingFields := structMissingFields(lit, strct, typ, pass)

		if len(missingFields) == 1 {
			pass.Reportf(node.Pos(), "%s is missing in %s", missingFields[0], strctName)
		} else if len(missingFields) > 1 {
			pass.Reportf(node.Pos(), "%s are missing in %s", strings.Join(missingFields, ", "), strctName)
		}
	}
}

func returnContainsLiteral(ret *ast.ReturnStmt, lit *ast.CompositeLit) bool {
	for _, result := range ret.Results {
		if l, ok := result.(*ast.CompositeLit); ok {
			if lit == l {
				return true
			}
		}
	}

	return false
}

func returnContainsError(ret *ast.ReturnStmt, pass *analysis.Pass) bool {
	for _, result := range ret.Results {
		if pass.TypesInfo.TypeOf(result).String() == "error" {
			return true
		}
	}

	return false
}

func structMissingFields(lit *ast.CompositeLit, strct *types.Struct, typ types.Type, pass *analysis.Pass) []string {
	isSamePackage := strings.HasPrefix(typ.String(), pass.Pkg.Path()+".")

	keys, unnamed := literalKeys(lit)
	fields := structFields(strct, isSamePackage)

	if unnamed {
		return fields[len(keys):]
	}

	return difference(fields, keys)
}

func literalKeys(lit *ast.CompositeLit) (keys []string, unnamed bool) {
	for _, elt := range lit.Elts {
		if k, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := k.Key.(*ast.Ident); ok {
				keys = append(keys, ident.Name)
			}

			continue
		}

		// in case we deal with unnamed initialization - no need to iterate over all
		// elements - simply create slice with proper size
		unnamed = true
		keys = make([]string, len(lit.Elts))

		break
	}

	return keys, unnamed
}

func structFields(strct *types.Struct, withPrivate bool) (keys []string) {
	for i := 0; i < strct.NumFields(); i++ {
		fieldName := strct.Field(i).Name()

		if !withPrivate && !strct.Field(i).Exported() {
			continue
		}

		keys = append(keys, fieldName)
	}

	return keys
}

// difference returns elements that are in `a` and not in `b`.
func difference(a, b []string) (diff []string) {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}

	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func exprName(expr ast.Expr) string {
	if i, ok := expr.(*ast.Ident); ok {
		return i.Name
	}

	s, ok := expr.(*ast.SelectorExpr)

	if !ok {
		return ""
	}

	return s.Sel.Name
}

type PatternsList []*regexp.Regexp

// MatchesAny matches provided string against all regexps in a slice.
func (l PatternsList) MatchesAny(str string) bool {
	for _, r := range l {
		if r.MatchString(str) {
			return true
		}
	}

	return false
}

// mustNewPatternsList parses slice of strings to a slice of compiled regular
// expressions.
func mustNewPatternsList(in []string) (list PatternsList) {
	for _, reStr := range in {
		if reStr == "" {
			panic(ErrEmptyPattern)
		}

		re, err := regexp.Compile(reStr)
		if err != nil {
			panic(fmt.Errorf("unable to compile %s as regular expression: %w", reStr, err))
		}

		list = append(list, re)
	}

	return list
}

type reListVar struct {
	values *PatternsList
}

func (v *reListVar) Set(value string) error {
	if value == "" {
		return ErrEmptyPattern
	}

	re, err := regexp.Compile(value)
	if err != nil {
		return fmt.Errorf("unable to compile %s as regular expression: %w", value, err)
	}

	*v.values = append(*v.values, re)

	return nil
}

func (v *reListVar) String() string {
	return ""
}
