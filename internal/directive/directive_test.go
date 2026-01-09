package directive_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

func Test_Directive_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		d    directive.Directive
		want bool
	}{
		{directive.Ignore, true},
		{directive.Enforce, true},
		{directive.Optional, true},
		{directive.Directive("invalid"), false},
		{directive.Directive(""), false},
		{directive.Directive("IGNORE"), false}, // case sensitive
		{directive.Directive("Ignore"), false},
		{directive.Directive(" ignore"), false}, // with space
	}

	for _, tt := range tests {
		t.Run(string(tt.d), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.d.IsValid())
		})
	}
}

func Test_Directives_Contains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ds   directive.Directives
		d    directive.Directive
		want bool
	}{
		{"nil slice", nil, directive.Ignore, false},
		{"empty slice", directive.Directives{}, directive.Ignore, false},
		{"contains single", directive.Directives{directive.Ignore}, directive.Ignore, true},
		{"not contains", directive.Directives{directive.Enforce, directive.Optional}, directive.Ignore, false},
		{"contains in multiple", directive.Directives{directive.Ignore, directive.Enforce}, directive.Enforce, true},
		{"first element", directive.Directives{directive.Ignore, directive.Enforce}, directive.Ignore, true},
		{
			"last element",
			directive.Directives{directive.Ignore, directive.Enforce, directive.Optional},
			directive.Optional,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.ds.Contains(tt.d))
		})
	}
}

func Test_Parse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		text       string
		directives directive.Directives
		found      bool
		wantErrs   []error
	}{
		{
			name:       "no prefix",
			text:       "// regular comment",
			directives: nil,
			found:      false,
			wantErrs:   nil,
		},
		{
			name:       "ignore directive",
			text:       "//exhaustruct:ignore",
			directives: directive.Directives{directive.Ignore},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "enforce directive",
			text:       "//exhaustruct:enforce",
			directives: directive.Directives{directive.Enforce},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "optional directive",
			text:       "//exhaustruct:optional",
			directives: directive.Directives{directive.Optional},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "directive with trailing comment",
			text:       "//exhaustruct:ignore some reason",
			directives: directive.Directives{directive.Ignore},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "invalid directive name",
			text:       "//exhaustruct:invalid",
			directives: nil,
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective},
		},
		{
			name:       "partial directive name",
			text:       "//exhaustruct:opt",
			directives: nil,
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective},
		},
		{
			name:       "empty directive",
			text:       "//exhaustruct:",
			directives: nil,
			found:      true,
			wantErrs:   []error{directive.ErrEmptyDirective},
		},
		{
			name:       "space after slashes",
			text:       "// exhaustruct:ignore",
			directives: nil,
			found:      false,
			wantErrs:   nil,
		},
		{
			name:       "block comment",
			text:       "/*exhaustruct:ignore*/",
			directives: nil,
			found:      false,
			wantErrs:   nil,
		},
		{
			name:       "multiple directives comma-separated",
			text:       "//exhaustruct:enforce,optional",
			directives: directive.Directives{directive.Enforce, directive.Optional},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "multiple directives with invalid",
			text:       "//exhaustruct:enforce,invalid,optional",
			directives: directive.Directives{directive.Enforce, directive.Optional},
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective},
		},
		{
			name:       "all directives",
			text:       "//exhaustruct:ignore,enforce,optional",
			directives: directive.Directives{directive.Ignore, directive.Enforce, directive.Optional},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "multiple directives with trailing comment",
			text:       "//exhaustruct:enforce,optional some reason",
			directives: directive.Directives{directive.Enforce, directive.Optional},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "duplicate directives",
			text:       "//exhaustruct:enforce,enforce",
			directives: directive.Directives{directive.Enforce},
			found:      true,
			wantErrs:   []error{directive.ErrDuplicateDirectives},
		},
		{
			name:       "duplicate and invalid",
			text:       "//exhaustruct:enforce,invalid,enforce",
			directives: directive.Directives{directive.Enforce},
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective, directive.ErrDuplicateDirectives},
		},
		// Edge cases for comma handling
		{
			name:       "leading comma",
			text:       "//exhaustruct:,ignore",
			directives: directive.Directives{directive.Ignore},
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective}, // empty string is invalid
		},
		{
			name:       "trailing comma",
			text:       "//exhaustruct:ignore,",
			directives: directive.Directives{directive.Ignore},
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective}, // empty string is invalid
		},
		{
			name:       "double comma",
			text:       "//exhaustruct:ignore,,enforce",
			directives: directive.Directives{directive.Ignore, directive.Enforce},
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective}, // empty string in middle
		},
		{
			name:       "tab as separator after directive",
			text:       "//exhaustruct:ignore\treason for ignoring",
			directives: directive.Directives{directive.Ignore},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "newline as separator after directive",
			text:       "//exhaustruct:ignore\nreason for ignoring",
			directives: directive.Directives{directive.Ignore},
			found:      true,
			wantErrs:   nil,
		},
		{
			name:       "uppercase directive",
			text:       "//exhaustruct:IGNORE",
			directives: nil,
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective},
		},
		{
			name:       "mixed case directive",
			text:       "//exhaustruct:Ignore",
			directives: nil,
			found:      true,
			wantErrs:   []error{directive.ErrUnknownDirective},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			found, d, errs := directive.Parse(tt.text)
			assert.Equal(t, tt.found, found, "found mismatch")
			assert.Equal(t, tt.directives, d, "directives mismatch")
			require.Len(t, errs, len(tt.wantErrs), "error count mismatch")

			for i, wantErr := range tt.wantErrs {
				assert.ErrorIs(t, errs[i], wantErr)
			}
		})
	}
}
