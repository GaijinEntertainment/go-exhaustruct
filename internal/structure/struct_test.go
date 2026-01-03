package structure_test

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

func Test_NewStruct(t *testing.T) {
	t.Parallel()

	td := loadTestdata(t)

	t.Run("basic structs", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			structName string
			wantFields int
		}{
			{"empty struct", "Empty", 0},
			{"single field", "SingleField", 1},
			{"multiple fields", "MultiField", 3},
			{"mixed exported", "MixedExported", 3},
			{"all unexported", "AllUnexported", 2},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				strct := td.getStruct(t, tt.structName)
				pos := td.getStructPos(t, tt.structName)

				info, diags := structure.NewStruct(td.fset, strct, tt.structName, td.pkg, pos, nil)

				require.NotNil(t, info)
				assert.Empty(t, diags)
				assert.Equal(t, tt.structName, info.Name)
				assert.Equal(t, "testdata", info.PackagePath)
				assert.Len(t, info.Fields, tt.wantFields)
			})
		}
	})

	t.Run("exported fields", func(t *testing.T) {
		t.Parallel()

		strct := td.getStruct(t, "MixedExported")
		pos := td.getStructPos(t, "MixedExported")

		info, diags := structure.NewStruct(td.fset, strct, "MixedExported", td.pkg, pos, nil)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		require.Len(t, info.Fields, 3)

		assert.Equal(t, "Exported", info.Fields[0].Name)
		assert.True(t, info.Fields[0].Exported)

		assert.Equal(t, "unexported", info.Fields[1].Name)
		assert.False(t, info.Fields[1].Exported)

		assert.Equal(t, "Another", info.Fields[2].Name)
		assert.True(t, info.Fields[2].Exported)
	})

	t.Run("struct level directives", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			structName string
			enforced   bool
			ignored    bool
			optional   bool
		}{
			{"ignored struct", "IgnoredStruct", false, true, false},
			{"enforced struct", "EnforcedStruct", true, false, false},
			{"optional struct", "OptionalStruct", false, false, true},
			{"no directives", "MultiField", false, false, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				strct := td.getStruct(t, tt.structName)
				pos := td.getStructPos(t, tt.structName)

				info, diags := structure.NewStruct(td.fset, strct, tt.structName, td.pkg, pos, td.cache)

				require.NotNil(t, info)
				assert.Empty(t, diags)
				assert.Equal(t, tt.enforced, info.Enforced, "Enforced mismatch")
				assert.Equal(t, tt.ignored, info.Ignored, "Ignored mismatch")
				assert.Equal(t, tt.optional, info.Optional, "Optional mismatch")
			})
		}
	})

	t.Run("field level directives", func(t *testing.T) {
		t.Parallel()

		t.Run("optional via doc comment", func(t *testing.T) {
			t.Parallel()

			strct := td.getStruct(t, "WithOptionalDoc")
			pos := td.getStructPos(t, "WithOptionalDoc")

			info, diags := structure.NewStruct(td.fset, strct, "WithOptionalDoc", td.pkg, pos, td.cache)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields, 2)

			assert.Equal(t, "Required", info.Fields[0].Name)
			assert.False(t, info.Fields[0].Optional)

			assert.Equal(t, "Optional", info.Fields[1].Name)
			assert.True(t, info.Fields[1].Optional)
		})

		t.Run("optional via inline comment", func(t *testing.T) {
			t.Parallel()

			strct := td.getStruct(t, "WithOptionalInline")
			pos := td.getStructPos(t, "WithOptionalInline")

			info, diags := structure.NewStruct(td.fset, strct, "WithOptionalInline", td.pkg, pos, td.cache)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields, 2)

			assert.Equal(t, "Required", info.Fields[0].Name)
			assert.False(t, info.Fields[0].Optional)

			assert.Equal(t, "Optional", info.Fields[1].Name)
			assert.True(t, info.Fields[1].Optional)
		})

		t.Run("enforced field", func(t *testing.T) {
			t.Parallel()

			strct := td.getStruct(t, "WithEnforcedField")
			pos := td.getStructPos(t, "WithEnforcedField")

			info, diags := structure.NewStruct(td.fset, strct, "WithEnforcedField", td.pkg, pos, td.cache)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields, 2)

			assert.Equal(t, "Normal", info.Fields[0].Name)
			assert.False(t, info.Fields[0].Enforced)

			assert.Equal(t, "Enforced", info.Fields[1].Name)
			assert.True(t, info.Fields[1].Enforced)
		})

		t.Run("mixed directives", func(t *testing.T) {
			t.Parallel()

			strct := td.getStruct(t, "WithMixedDirectives")
			pos := td.getStructPos(t, "WithMixedDirectives")

			info, diags := structure.NewStruct(td.fset, strct, "WithMixedDirectives", td.pkg, pos, td.cache)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields, 3)

			assert.Equal(t, "Normal", info.Fields[0].Name)
			assert.False(t, info.Fields[0].Optional)
			assert.False(t, info.Fields[0].Enforced)

			assert.Equal(t, "Optional", info.Fields[1].Name)
			assert.True(t, info.Fields[1].Optional)
			assert.False(t, info.Fields[1].Enforced)

			assert.Equal(t, "Enforced", info.Fields[2].Name)
			assert.False(t, info.Fields[2].Optional)
			assert.True(t, info.Fields[2].Enforced)
		})
	})

	t.Run("anonymous struct", func(t *testing.T) {
		t.Parallel()

		strct := td.getStruct(t, "SingleField")

		info, diags := structure.NewStruct(td.fset, strct, structure.AnonymousName, td.pkg, 0, nil)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.Equal(t, structure.AnonymousName, info.Name)
	})

	t.Run("nil lookup", func(t *testing.T) {
		t.Parallel()

		strct := td.getStruct(t, "IgnoredStruct")
		pos := td.getStructPos(t, "IgnoredStruct")

		info, diags := structure.NewStruct(td.fset, strct, "IgnoredStruct", td.pkg, pos, nil)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.False(t, info.Ignored, "should be false with nil lookup")
	})

	t.Run("embedded fields", func(t *testing.T) {
		t.Parallel()

		t.Run("exported embedded", func(t *testing.T) {
			t.Parallel()

			strct := td.getStruct(t, "WithEmbedded")
			pos := td.getStructPos(t, "WithEmbedded")

			info, diags := structure.NewStruct(td.fset, strct, "WithEmbedded", td.pkg, pos, nil)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields, 2)

			assert.Equal(t, "Embedded", info.Fields[0].Name)
			assert.True(t, info.Fields[0].Exported)

			assert.Equal(t, "Own", info.Fields[1].Name)
			assert.True(t, info.Fields[1].Exported)
		})

		t.Run("unexported embedded", func(t *testing.T) {
			t.Parallel()

			strct := td.getStruct(t, "WithUnexportedEmbedded")
			pos := td.getStructPos(t, "WithUnexportedEmbedded")

			info, diags := structure.NewStruct(td.fset, strct, "WithUnexportedEmbedded", td.pkg, pos, nil)

			require.NotNil(t, info)
			assert.Empty(t, diags)
			require.Len(t, info.Fields, 2)

			assert.Equal(t, "unexported", info.Fields[0].Name)
			assert.False(t, info.Fields[0].Exported)

			assert.Equal(t, "Own", info.Fields[1].Name)
			assert.True(t, info.Fields[1].Exported)
		})
	})
}

func Test_Struct_SkippedFields(t *testing.T) {
	t.Parallel()

	td := loadTestdata(t)

	// Get info for LiteralTest struct with directive lookup
	strct := td.getStruct(t, "LiteralTest")
	pos := td.getStructPos(t, "LiteralTest")
	info, _ := structure.NewStruct(td.fset, strct, "LiteralTest", td.pkg, pos, td.cache)

	// Verify fields are parsed correctly
	require.Len(t, info.Fields, 4)
	assert.False(t, info.Fields[0].Optional) // ExportedRequired
	assert.False(t, info.Fields[1].Optional) // unexportedRequired
	assert.True(t, info.Fields[2].Optional)  // ExportedOptional
	assert.True(t, info.Fields[3].Optional)  // unexportedOptional

	t.Run("positional complete", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_positionalComplete")
		assert.Nil(t, info.SkippedFields(lit, true))
		assert.Nil(t, info.SkippedFields(lit, false))
	})

	t.Run("positional incomplete", func(t *testing.T) {
		t.Parallel()

		// Create AST manually - positional literals with fewer values are invalid Go
		// but we still need to test the logic
		lit := &ast.CompositeLit{ //nolint:exhaustruct
			Elts: []ast.Expr{
				&ast.BasicLit{Kind: token.INT, Value: "1"}, //nolint:exhaustruct
			},
		}

		// externalPkg=false: returns all remaining fields
		skipped := info.SkippedFields(lit, false)
		require.Len(t, skipped, 3)
		assert.Equal(t, "unexportedRequired", skipped[0].Name)
		assert.Equal(t, "ExportedOptional", skipped[1].Name)
		assert.Equal(t, "unexportedOptional", skipped[2].Name)

		// externalPkg=true: returns only exported remaining fields
		skipped = info.SkippedFields(lit, true)
		require.Len(t, skipped, 1)
		assert.Equal(t, "ExportedOptional", skipped[0].Name)
	})

	t.Run("named complete", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_namedComplete")
		assert.Nil(t, info.SkippedFields(lit, true))
		assert.Nil(t, info.SkippedFields(lit, false))
	})

	t.Run("named missing unexported", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_namedMissingUnexported")

		// externalPkg=true: unexported not required (inaccessible)
		assert.Nil(t, info.SkippedFields(lit, true))

		// externalPkg=false: unexported required
		skipped := info.SkippedFields(lit, false)
		require.Len(t, skipped, 1)
		assert.Equal(t, "unexportedRequired", skipped[0].Name)
	})

	t.Run("named missing exported", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_namedMissingExported")

		// externalPkg=true: only exported required reported
		skipped := info.SkippedFields(lit, true)
		require.Len(t, skipped, 1)
		assert.Equal(t, "ExportedRequired", skipped[0].Name)

		// externalPkg=false: both required fields reported
		skipped = info.SkippedFields(lit, false)
		require.Len(t, skipped, 2)
		assert.Equal(t, "ExportedRequired", skipped[0].Name)
		assert.Equal(t, "unexportedRequired", skipped[1].Name)
	})

	t.Run("empty literal", func(t *testing.T) {
		t.Parallel()

		lit := td.getLiteral(t, "_empty")

		// externalPkg=true: only exported required
		skipped := info.SkippedFields(lit, true)
		require.Len(t, skipped, 1)
		assert.Equal(t, "ExportedRequired", skipped[0].Name)

		// externalPkg=false: all required
		skipped = info.SkippedFields(lit, false)
		require.Len(t, skipped, 2)
		assert.Equal(t, "ExportedRequired", skipped[0].Name)
		assert.Equal(t, "unexportedRequired", skipped[1].Name)
	})
}

func Test_Struct_SkippedFields_EmptyStruct(t *testing.T) {
	t.Parallel()

	td := loadTestdata(t)

	strct := td.getStruct(t, "Empty")
	pos := td.getStructPos(t, "Empty")
	info, _ := structure.NewStruct(td.fset, strct, "Empty", td.pkg, pos, nil)

	lit := &ast.CompositeLit{Elts: []ast.Expr{}} //nolint:exhaustruct

	assert.Nil(t, info.SkippedFields(lit, true))
	assert.Nil(t, info.SkippedFields(lit, false))
}
