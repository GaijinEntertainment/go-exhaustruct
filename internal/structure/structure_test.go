package structure_test

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

func Test_Struct_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fullPath  string
		wantStr   string
		wantShort string
		wantPkg   string
	}{
		{
			name:      "simple package",
			fullPath:  "main.Config",
			wantStr:   "main.Config",
			wantShort: "main.Config",
			wantPkg:   "main",
		},
		{
			name:      "nested package",
			fullPath:  "net/http.Request",
			wantStr:   "net/http.Request",
			wantShort: "http.Request",
			wantPkg:   "net/http",
		},
		{
			name:      "deep nested",
			fullPath:  "github.com/user/repo/pkg.Type",
			wantStr:   "github.com/user/repo/pkg.Type",
			wantShort: "pkg.Type",
			wantPkg:   "github.com/user/repo/pkg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &structure.Struct{Name: "Type", FullPath: tt.fullPath, PackageName: "pkg"}

			assert.Equal(t, tt.wantStr, s.String())
			assert.Equal(t, tt.wantShort, s.ShortString())
			assert.Equal(t, tt.wantPkg, s.PackagePath())
		})
	}
}

func Test_Field_String(t *testing.T) {
	t.Parallel()

	f := structure.Field{Name: "MyField"}
	assert.Equal(t, "MyField", f.String())
}

func Test_Fields_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields structure.Fields
		want   string
	}{
		{
			name:   "empty",
			fields: structure.Fields{PackagePath: "test", Items: nil},
			want:   "",
		},
		{
			name: "single",
			fields: structure.Fields{
				PackagePath: "test",
				Items: []structure.Field{
					{Name: "Foo", Exported: true},
				},
			},
			want: "Foo",
		},
		{
			name: "multiple",
			fields: structure.Fields{
				PackagePath: "test",
				Items: []structure.Field{
					{Name: "Foo", Exported: true},
					{Name: "Bar", Exported: true},
					{Name: "Baz", Exported: true},
				},
			},
			want: "Foo, Bar, Baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.fields.String())
		})
	}
}

func Test_Struct_IsEnforced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		enforced        bool
		patternEnforced bool
		want            bool
	}{
		{"neither", false, false, false},
		{"directive only", true, false, true},
		{"pattern only", false, true, true},
		{"both", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &structure.Struct{
				Name:            "Test",
				FullPath:        "test.Test",
				PackageName:     "test",
				Enforced:        tt.enforced,
				PatternEnforced: tt.patternEnforced,
			}

			assert.Equal(t, tt.want, s.IsEnforced())
		})
	}
}

func Test_Struct_IsIgnored(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		ignored        bool
		patternIgnored bool
		want           bool
	}{
		{"neither", false, false, false},
		{"directive only", true, false, true},
		{"pattern only", false, true, true},
		{"both", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &structure.Struct{
				Name:           "Test",
				FullPath:       "test.Test",
				PackageName:    "test",
				Ignored:        tt.ignored,
				PatternIgnored: tt.patternIgnored,
			}

			assert.Equal(t, tt.want, s.IsIgnored())
		})
	}
}

func Test_Struct_IsOptional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		optional        bool
		patternOptional bool
		want            bool
	}{
		{"neither", false, false, false},
		{"directive only", true, false, true},
		{"pattern only", false, true, true},
		{"both", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &structure.Struct{
				Name:            "Test",
				FullPath:        "test.Test",
				PackageName:     "test",
				Optional:        tt.optional,
				PatternOptional: tt.patternOptional,
			}

			assert.Equal(t, tt.want, s.IsOptional())
		})
	}
}

func Test_Struct_isFieldRequired_combinations(t *testing.T) {
	t.Parallel()

	samePkg := "test"
	externalPkg := "other"

	tests := []struct {
		name           string
		fieldEnforced  bool
		fieldOptional  bool
		fieldExported  bool
		structOptional bool
		callerPkg      string
		wantRequired   bool
	}{
		// Enforced always required
		{
			name:           "enforced on normal struct",
			fieldEnforced:  true,
			fieldOptional:  false,
			structOptional: false,
			fieldExported:  true,
			callerPkg:      samePkg,
			wantRequired:   true,
		},
		{
			name:           "enforced on optional struct",
			fieldEnforced:  true,
			fieldOptional:  false,
			structOptional: true,
			fieldExported:  true,
			callerPkg:      samePkg,
			wantRequired:   true,
		},
		{
			name:           "enforced unexported external",
			fieldEnforced:  true,
			fieldOptional:  false,
			structOptional: false,
			fieldExported:  false,
			callerPkg:      externalPkg,
			wantRequired:   true, // enforced overrides external
		},
		// Optional field not required
		{
			name:           "optional field on normal struct",
			fieldEnforced:  false,
			fieldOptional:  true,
			structOptional: false,
			fieldExported:  true,
			callerPkg:      samePkg,
			wantRequired:   false,
		},
		// Struct optional makes all non-enforced fields optional
		{
			name:           "regular field on optional struct",
			fieldEnforced:  false,
			fieldOptional:  false,
			structOptional: true,
			fieldExported:  true,
			callerPkg:      samePkg,
			wantRequired:   false,
		},
		// Unexported external not required
		{
			name:           "unexported external not required",
			fieldEnforced:  false,
			fieldOptional:  false,
			structOptional: false,
			fieldExported:  false,
			callerPkg:      externalPkg,
			wantRequired:   false,
		},
		// Exported external is required
		{
			name:           "exported external is required",
			fieldEnforced:  false,
			fieldOptional:  false,
			structOptional: false,
			fieldExported:  true,
			callerPkg:      externalPkg,
			wantRequired:   true,
		},
		// Unexported same-package is required
		{
			name:           "unexported same package is required",
			fieldEnforced:  false,
			fieldOptional:  false,
			structOptional: false,
			fieldExported:  false,
			callerPkg:      samePkg,
			wantRequired:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &structure.Struct{
				Name:        "Test",
				FullPath:    "test.Test",
				PackageName: "test",
				Optional:    tt.structOptional,
				Fields: structure.Fields{
					PackagePath: samePkg,
					Items: []structure.Field{
						{
							Name:     "TestField",
							Exported: tt.fieldExported,
							Enforced: tt.fieldEnforced,
							Optional: tt.fieldOptional,
						},
					},
				},
			}

			// Empty literal to trigger check of all fields.
			lit := &ast.CompositeLit{Elts: []ast.Expr{}} //nolint:exhaustruct

			skipped := s.SkippedFields(lit, tt.callerPkg)

			if tt.wantRequired {
				require.Len(t, skipped, 1)
				assert.Equal(t, "TestField", skipped[0].Name)
			} else {
				assert.Empty(t, skipped)
			}
		})
	}
}
