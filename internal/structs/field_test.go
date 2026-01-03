package structs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"dev.gaijin.team/go/exhaustruct/v4/internal/structs"
)

func Test_Fields_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields structs.Fields
		want   string
	}{
		{
			name:   "empty",
			fields: structs.Fields{},
			want:   "",
		},
		{
			name: "single",
			fields: structs.Fields{
				{Name: "Foo", Exported: true},
			},
			want: "Foo",
		},
		{
			name: "multiple",
			fields: structs.Fields{
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
