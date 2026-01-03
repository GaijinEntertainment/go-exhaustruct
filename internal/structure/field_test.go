package structure_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

func Test_Fields_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields structure.Fields
		want   string
	}{
		{
			name:   "empty",
			fields: structure.Fields{},
			want:   "",
		},
		{
			name: "single",
			fields: structure.Fields{
				{Name: "Foo", Exported: true},
			},
			want: "Foo",
		},
		{
			name: "multiple",
			fields: structure.Fields{
				{Name: "Foo", Exported: true},
				{Name: "Bar", Exported: true},
				{Name: "Baz", Exported: true},
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
