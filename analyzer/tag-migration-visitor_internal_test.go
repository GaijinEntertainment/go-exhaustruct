package analyzer

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_buildTagDiagnostic_carriesAFix pins that every deprecated tag reported
// comes with the edit that migrates it. A golden file cannot answer this:
// analysistest compares the files a fix rewrote, so a diagnostic carrying none
// leaves nothing to compare against.
func Test_buildTagDiagnostic_carriesAFix(t *testing.T) {
	t.Parallel()

	field := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("Field")},
		Type:  ast.NewIdent("string"),
		Tag:   &ast.BasicLit{ValuePos: token.Pos(1), Kind: token.STRING, Value: "`exhaustruct:\"optional\"`"},
	}

	placement := tagPlacement{
		canAppend:          true,
		anchors:            []nameAnchor{{pos: token.Pos(1), startsLine: true}},
		optionalityDecided: false,
		gapBeforeTag:       false,
	}

	diag := buildTagDiagnostic(field, placement, optionalTagValue)

	require.Len(t, diag.SuggestedFixes, 1)
	assert.NotEmpty(t, diag.SuggestedFixes[0].TextEdits)
}

// Test_buildTagFix_appendedDirective pins the bytes an appended directive is
// written with. Every driver formats the file it fixes and the golden files are
// compared formatted, so only this test sees the space between the type and the
// directive.
func Test_buildTagFix_appendedDirective(t *testing.T) {
	t.Parallel()

	const (
		typePos = token.Pos(10)
		tagPos  = token.Pos(17)
	)

	tests := []struct {
		name        string
		giveTag     string
		giveGap     bool
		wantPos     token.Pos
		wantNewText string
	}{
		{
			name:        "the tag dropped whole gives way to the directive",
			giveTag:     "`exhaustruct:\"optional\"`",
			giveGap:     false,
			wantPos:     typePos + token.Pos(len("string")),
			wantNewText: " //exhaustruct:optional",
		},
		{
			name:        "the tag's remainder stays ahead of the directive",
			giveTag:     "`json:\"f\" exhaustruct:\"optional\"`",
			giveGap:     false,
			wantPos:     tagPos,
			wantNewText: "`json:\"f\"` //exhaustruct:optional",
		},
		{
			name:        "a comment before the tag keeps the whitespace after it",
			giveTag:     "`exhaustruct:\"optional\"`",
			giveGap:     true,
			wantPos:     tagPos,
			wantNewText: "//exhaustruct:optional",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field := &ast.Field{
				Names: []*ast.Ident{ast.NewIdent("Field")},
				Type:  &ast.Ident{NamePos: typePos, Name: "string"},
				Tag:   &ast.BasicLit{ValuePos: tagPos, Kind: token.STRING, Value: tt.giveTag},
			}

			placement := tagPlacement{
				canAppend:          true,
				anchors:            []nameAnchor{{pos: token.Pos(1), startsLine: true}},
				optionalityDecided: false,
				gapBeforeTag:       tt.giveGap,
			}

			fix := buildTagFix(field, placement, optionalTagValue)

			require.Len(t, fix.TextEdits, 1)
			assert.Equal(t, tt.wantPos, fix.TextEdits[0].Pos)
			assert.Equal(t, tt.wantNewText, string(fix.TextEdits[0].NewText))
		})
	}
}

// Test_removeExhaustructFromTag pins what is left of a tag once the deprecated
// key is gone, down to the delimiter the remainder is written with.
func Test_removeExhaustructFromTag(t *testing.T) {
	t.Parallel()

	const (
		backtick = "`"
		repeated = `exhaustruct:"optional"`
	)

	tests := []struct {
		name string
		give string
		want string
	}{
		{
			name: "the deprecated key alone leaves no tag",
			give: backtick + `exhaustruct:"optional"` + backtick,
			want: "",
		},
		{
			name: "keys after the deprecated one survive",
			give: backtick + `exhaustruct:"optional" json:"f"` + backtick,
			want: backtick + `json:"f"` + backtick,
		},
		{
			name: "keys around the deprecated one survive",
			give: backtick + `json:"f" exhaustruct:"optional" yaml:"f"` + backtick,
			want: backtick + `json:"f" yaml:"f"` + backtick,
		},
		{
			name: "an escaped quote does not end the deprecated value",
			give: backtick + `json:"x" exhaustruct:"foo\"bar" yaml:"y"` + backtick,
			want: backtick + `json:"x" yaml:"y"` + backtick,
		},
		{
			name: "a repeated deprecated key is cut out whole",
			give: backtick + `json:"f" ` + repeated + " " + repeated + backtick,
			want: backtick + `json:"f"` + backtick,
		},
		{
			name: "a tail that stops parsing survives the cut",
			give: backtick + `exhaustruct:"optional" malformed` + backtick,
			want: backtick + `malformed` + backtick,
		},
		{
			name: "a key ending in the deprecated one is a different key",
			give: backtick + `fooexhaustruct:"keep" exhaustruct:"optional"` + backtick,
			want: backtick + `fooexhaustruct:"keep"` + backtick,
		},
		{
			name: "a backtick in the remaining tag forces an interpreted literal",
			give: `"exhaustruct:\"optional\" json:\"a` + backtick + `b\""`,
			want: `"json:\"a` + backtick + `b\""`,
		},
		{
			name: "a literal that does not unquote is left alone",
			give: `exhaustruct:"optional"`,
			want: `exhaustruct:"optional"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, removeExhaustructFromTag(tt.give))
		})
	}
}
