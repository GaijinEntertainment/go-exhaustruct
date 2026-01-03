package structure_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/structure"
)

func Test_Cache(t *testing.T) {
	t.Parallel()

	td := loadTestdata(t)

	t.Run("miss", func(t *testing.T) {
		t.Parallel()

		cache := &structure.Cache{}

		strct := td.getStruct(t, "SingleField")
		pos := td.getStructPos(t, "SingleField")

		info, diags := cache.Get(td.fset, strct, "SingleField", td.pkg, pos, nil)

		require.NotNil(t, info)
		assert.Empty(t, diags)
		assert.Equal(t, "SingleField", info.Name)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(1), misses)
	})

	t.Run("hit", func(t *testing.T) {
		t.Parallel()

		cache := &structure.Cache{}

		strct := td.getStruct(t, "SingleField")
		pos := td.getStructPos(t, "SingleField")

		info1, _ := cache.Get(td.fset, strct, "SingleField", td.pkg, pos, nil)
		info2, diags := cache.Get(td.fset, strct, "SingleField", td.pkg, pos, nil)

		assert.Same(t, info1, info2)
		assert.Empty(t, diags)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(1), hits)
		assert.Equal(t, uint64(1), misses)
	})

	t.Run("different structs", func(t *testing.T) {
		t.Parallel()

		cache := &structure.Cache{}

		strct1 := td.getStruct(t, "SingleField")
		pos1 := td.getStructPos(t, "SingleField")

		strct2 := td.getStruct(t, "MultiField")
		pos2 := td.getStructPos(t, "MultiField")

		info1, _ := cache.Get(td.fset, strct1, "SingleField", td.pkg, pos1, nil)
		info2, _ := cache.Get(td.fset, strct2, "MultiField", td.pkg, pos2, nil)

		assert.NotSame(t, info1, info2)
		assert.Equal(t, "SingleField", info1.Name)
		assert.Equal(t, "MultiField", info2.Name)

		hits, misses := cache.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(2), misses)
	})

	t.Run("with directives", func(t *testing.T) {
		t.Parallel()

		cache := &structure.Cache{}

		strct := td.getStruct(t, "IgnoredStruct")
		pos := td.getStructPos(t, "IgnoredStruct")

		// First call with lookup - should parse directives
		info1, diags := cache.Get(td.fset, strct, "IgnoredStruct", td.pkg, pos, td.cache)

		require.NotNil(t, info1)
		assert.Empty(t, diags)
		assert.True(t, info1.Ignored)

		// Second call - cached, directives preserved
		info2, diags := cache.Get(td.fset, strct, "IgnoredStruct", td.pkg, pos, td.cache)

		assert.Same(t, info1, info2)
		assert.Empty(t, diags)
		assert.True(t, info2.Ignored)
	})
}
