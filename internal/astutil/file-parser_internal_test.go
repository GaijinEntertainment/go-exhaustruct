package astutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// hasPathPrefix is tested directly because the prefix it guards against comes
// from build.Default.GOROOT, which cannot be injected without mutating a global
// shared by the parallel tests of this package.
func Test_hasPathPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		givePath   string
		givePrefix string
		want       bool
	}{
		{
			name:       "element boundary",
			givePath:   "$GOROOT/src/strings/builder.go",
			givePrefix: "$GOROOT",
			want:       true,
		},
		{
			name:       "exact match",
			givePath:   "/usr/local/go",
			givePrefix: "/usr/local/go",
			want:       true,
		},
		{
			name:       "trailing separator in prefix",
			givePath:   "/usr/local/go/src/net/tcpsock.go",
			givePrefix: "/usr/local/go/",
			want:       true,
		},
		{
			// GOROOT carries native separators, the compiler records slashes.
			name:       "native separators in prefix",
			givePath:   "/usr/local/go/src/net/tcpsock.go",
			givePrefix: filepath.FromSlash("/usr/local/go"),
			want:       true,
		},
		{
			name:       "mid-element match",
			givePath:   "/usr/local/gopher/src/main.go",
			givePrefix: "/usr/local/go",
			want:       false,
		},
		{
			name:       "unrelated path",
			givePath:   "/home/user/project/main.go",
			givePrefix: "/usr/local/go",
			want:       false,
		},
		{
			// An unset GOROOT must match nothing, not everything.
			name:       "empty prefix",
			givePath:   "/home/user/project/main.go",
			givePrefix: "",
			want:       false,
		},
		{
			// A root GOROOT normalizes to an empty prefix.
			name:       "root prefix",
			givePath:   "/home/user/project/main.go",
			givePrefix: "/",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, hasPathPrefix(tt.givePath, tt.givePrefix))
		})
	}
}
