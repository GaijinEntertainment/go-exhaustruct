package directive_test

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dev.gaijin.team/go/exhaustruct/v5/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v5/internal/directive"
)

// BenchmarkLookup_OneDirectiveInALargeFile scans a file that declares a great
// deal and directs once. Only the lines a directive comment covers are ever
// asked about, so what the scan keeps has to follow the directives rather than
// the size of the file.
func BenchmarkLookup_OneDirectiveInALargeFile(b *testing.B) {
	for _, decls := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("decls%d", decls), func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "big.go")

			if err := os.WriteFile(path, []byte(largeSource(decls)), 0o600); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()

			for b.Loop() {
				scanner := directive.NewScanner(astutil.NewFileParser())

				d := scanner.Lookup(token.NewFileSet(), token.Position{Filename: path, Line: directedLine})
				if len(d) != 1 {
					b.Fatalf("the directive was not found: %v", d)
				}
			}
		})
	}
}

// directedLine is the line largeSource writes its one directed declaration on.
const directedLine = 4

func largeSource(decls int) string {
	var b strings.Builder

	b.WriteString("package big\n\n//exhaustruct:optional\nvar tagged int\n\n")

	for i := range decls {
		fmt.Fprintf(&b, "type T%d struct {\n\tA%d int\n\tB%d string\n}\n\n", i, i, i)
	}

	return b.String()
}
