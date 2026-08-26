package cache_test

import (
	"strconv"
	"testing"

	"dev.gaijin.team/go/exhaustruct/v5/internal/cache"
)

// BenchmarkGetOrSet_HitParallel measures the answer a filled key gives while
// every other thread asks for a key of its own. The Processor holding this
// cache is shared by the passes an analysis driver runs at once, and a resolved
// literal reaches this path on every hit.
func BenchmarkGetOrSet_HitParallel(b *testing.B) {
	const keys = 64

	c := cache.New[string, int](keys)

	names := make([]string, keys)
	for i := range names {
		names[i] = strconv.Itoa(i)
		c.Set(names[i], i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0

		for pb.Next() {
			if v := c.GetOrSet(names[i%keys], func() int { return -1 }); v < 0 {
				b.Fatal("the key was not filled")
			}

			i++
		}
	})
}
