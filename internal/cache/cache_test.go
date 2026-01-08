package cache_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/cache"
)

func Test_Cache(t *testing.T) {
	t.Parallel()

	t.Run("ResolveStruct miss", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](8)

		v, ok := c.Get("missing")

		assert.False(t, ok)
		assert.Zero(t, v)

		hits, misses, size := c.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(0), misses)
		assert.Equal(t, uint64(0), size)
	})

	t.Run("Set and ResolveStruct", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](8)

		c.Set("key", 42)

		v, ok := c.Get("key")

		require.True(t, ok)
		assert.Equal(t, 42, v)

		hits, misses, size := c.Stats()
		assert.Equal(t, uint64(1), hits)
		assert.Equal(t, uint64(1), misses) // Set records miss
		assert.Equal(t, uint64(1), size)
	})

	t.Run("GetOrSet miss", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](8)
		computed := false

		v := c.GetOrSet("key", func() int {
			computed = true
			return 42
		})

		assert.True(t, computed)
		assert.Equal(t, 42, v)

		hits, misses, size := c.Stats()
		assert.Equal(t, uint64(0), hits)
		assert.Equal(t, uint64(1), misses)
		assert.Equal(t, uint64(1), size)
	})

	t.Run("GetOrSet hit", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](8)

		c.Set("key", 42)

		computed := false

		v := c.GetOrSet("key", func() int {
			computed = true
			return 99
		})

		assert.False(t, computed)
		assert.Equal(t, 42, v)

		hits, misses, size := c.Stats()
		assert.Equal(t, uint64(1), hits)
		assert.Equal(t, uint64(1), misses) // from Set
		assert.Equal(t, uint64(1), size)
	})

	t.Run("concurrent access", func(t *testing.T) {
		t.Parallel()

		c := cache.New[int, int](64)

		var wg sync.WaitGroup

		for i := range 100 {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				c.GetOrSet(i%10, func() int {
					return i
				})
			}(i)
		}

		wg.Wait()

		_, _, size := c.Stats()
		assert.Equal(t, uint64(10), size)
	})

	t.Run("GetOrSet same key concurrent", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](10)

		var computeCount atomic.Int32

		var wg sync.WaitGroup

		for range 100 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				c.GetOrSet("same-key", func() int {
					computeCount.Add(1)
					time.Sleep(10 * time.Millisecond)

					return 42
				})
			}()
		}

		wg.Wait()

		assert.Equal(t, int32(1), computeCount.Load(),
			"compute should be called exactly once")

		_, _, size := c.Stats()
		assert.Equal(t, uint64(1), size)
	})

	t.Run("zero size prealloc", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](0)

		c.Set("key", 42)

		v, ok := c.Get("key")

		require.True(t, ok)
		assert.Equal(t, 42, v)
	})

	t.Run("store zero value", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](8)

		c.Set("zero", 0)

		v, ok := c.Get("zero")

		require.True(t, ok)
		assert.Equal(t, 0, v)
	})

	t.Run("empty string key", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](8)

		c.Set("", 42)

		v, ok := c.Get("")

		require.True(t, ok)
		assert.Equal(t, 42, v)
	})
}
