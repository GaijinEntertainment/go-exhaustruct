package structure_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/astutil"
	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

func Test_OriginScanner(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/origins.go", nil, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	origin := structure.NewOriginScanner(fp)
	origin.ProcessFiles(fset, file)

	tests := []struct {
		name     string
		typeName string
		want     structure.TypeOrigin
	}{
		// Original structs
		{"empty struct", "OriginalEmpty", structure.OriginStruct},
		{"struct with fields", "OriginalWithFields", structure.OriginStruct},
		{"generic struct", "GenericStruct", structure.OriginStruct},

		// Aliases
		{"alias to struct", "AliasToOriginal", structure.OriginAlias},
		{"alias to int", "AliasToInt", structure.OriginAlias},
		{"generic alias", "GenericAlias", structure.OriginAlias},

		// Derived types
		{"derived from struct", "DerivedFromOriginal", structure.OriginDerived},
		{"derived from alias", "DerivedFromAlias", structure.OriginDerived},

		// Non-struct type definitions (classified as derived)
		{"interface", "MyInterface", structure.OriginDerived},
		{"func type", "MyFunc", structure.OriginDerived},
		{"slice type", "MySlice", structure.OriginDerived},
		{"map type", "MyMap", structure.OriginDerived},
		{"chan type", "MyChan", structure.OriginDerived},

		// Unknown
		{"not defined", "NotDefined", structure.OriginUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := origin.Lookup(fset, "testdata/origins.go", tt.typeName)
			assert.Equal(t, tt.want, got, "TypeOrigin mismatch for %s", tt.typeName)
		})
	}
}

func Test_OriginScanner_Stats(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testdata/origins.go", nil, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	origin := structure.NewOriginScanner(fp)

	// Initial stats: all zeros.
	hits, misses, size := origin.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, uint64(0), size)

	// ProcessFiles - triggers miss.
	origin.ProcessFiles(fset, file)

	_, misses, size = origin.Stats()
	assert.Equal(t, uint64(1), misses)
	assert.Equal(t, uint64(1), size)

	// Lookup - triggers hit.
	origin.Lookup(fset, "testdata/origins.go", "OriginalEmpty")

	hits, _, _ = origin.Stats()
	assert.Equal(t, uint64(1), hits)

	// Another lookup - another hit.
	origin.Lookup(fset, "testdata/origins.go", "AliasToOriginal")

	hits, _, _ = origin.Stats()
	assert.Equal(t, uint64(2), hits)
}

func Test_OriginScanner_EmptyInputs(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	fp := astutil.NewFileParser()
	origin := structure.NewOriginScanner(fp)

	// Empty filename returns unknown.
	got := origin.Lookup(fset, "", "SomeType")
	assert.Equal(t, structure.OriginUnknown, got)

	// Empty type name returns unknown.
	got = origin.Lookup(fset, "test.go", "")
	assert.Equal(t, structure.OriginUnknown, got)

	// Should not increment stats.
	hits, misses, size := origin.Stats()
	assert.Equal(t, uint64(0), hits)
	assert.Equal(t, uint64(0), misses)
	assert.Equal(t, uint64(0), size)
}

func Test_OriginScanner_MultipleFiles(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()

	// Load both testdata files.
	file1, err := parser.ParseFile(fset, "testdata/structs.go", nil, parser.ParseComments)
	require.NoError(t, err)

	file2, err := parser.ParseFile(fset, "testdata/origins.go", nil, parser.ParseComments)
	require.NoError(t, err)

	fp := astutil.NewFileParser()
	origin := structure.NewOriginScanner(fp)

	origin.ProcessFiles(fset, file1, file2)

	_, misses, size := origin.Stats()
	assert.Equal(t, uint64(2), misses)
	assert.Equal(t, uint64(2), size)

	// Lookup from structs.go.
	assert.Equal(t, structure.OriginStruct, origin.Lookup(fset, "testdata/structs.go", "Empty"))
	assert.Equal(t, structure.OriginStruct, origin.Lookup(fset, "testdata/structs.go", "SingleField"))

	// Lookup from origins.go.
	assert.Equal(t, structure.OriginStruct, origin.Lookup(fset, "testdata/origins.go", "OriginalEmpty"))
	assert.Equal(t, structure.OriginAlias, origin.Lookup(fset, "testdata/origins.go", "AliasToOriginal"))

	// Type from one file not in the other.
	assert.Equal(t, structure.OriginUnknown, origin.Lookup(fset, "testdata/origins.go", "Empty"))
	assert.Equal(t, structure.OriginUnknown, origin.Lookup(fset, "testdata/structs.go", "OriginalEmpty"))
}
