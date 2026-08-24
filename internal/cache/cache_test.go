package cache_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v5/internal/cache"
)

// Test_Cache_GetOrSet_Concurrent covers the promise GetOrSet makes to callers
// racing for one key: it computes once, every caller receives what was
// computed, and filling one key holds up no other.
func Test_Cache_GetOrSet_Concurrent(t *testing.T) {
	t.Parallel()

	t.Run("GetOrSet same key concurrent", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			const waiters = 100

			c := cache.New[string, int](10)

			var computeCount atomic.Int32

			release := make(chan struct{})

			// Every caller records what it received: one of them computes, the
			// rest find the claim and wait on it, and all have to end up
			// holding the value that was computed.
			got := make([]int, waiters+1)

			var wg sync.WaitGroup

			wg.Go(func() {
				got[0] = c.GetOrSet("same-key", func() int {
					computeCount.Add(1)
					<-release

					return 42
				})
			})

			// Wait returns once every other goroutine of the bubble is durably
			// blocked: here, the caller parked on release inside compute.
			synctest.Wait()

			for i := 1; i <= waiters; i++ {
				wg.Go(func() {
					got[i] = c.GetOrSet("same-key", func() int {
						computeCount.Add(1)

						return -1
					})
				})
			}

			// Settled again, every waiter is parked on the claim it found. A
			// caller released before it joins reads the filled entry instead
			// and leaves the claim untested.
			synctest.Wait()

			close(release)
			wg.Wait()

			assert.Equal(t, int32(1), computeCount.Load(),
				"compute should be called exactly once")

			for i, v := range got {
				assert.Equal(t, 42, v, "caller %d received a value nobody computed", i)
			}

			_, _, size := c.Stats()
			assert.Equal(t, uint64(1), size)
		})
	})

	t.Run("GetOrSet different keys concurrent", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](2)

		first, second := make(chan struct{}), make(chan struct{})

		var wg sync.WaitGroup

		// Each compute waits for the other to have started, so the two finish
		// only if one key can be filled while the other is being computed.
		wg.Go(func() {
			c.GetOrSet("first", func() int {
				close(first)
				<-second

				return 1
			})
		})

		wg.Go(func() {
			c.GetOrSet("second", func() int {
				close(second)
				<-first

				return 2
			})
		})

		awaitGroup(t, &wg, "one key being filled while another is")
	})

	t.Run("GetOrSet keeps a Set that lands while it computes", func(t *testing.T) {
		t.Parallel()

		c := cache.New[string, int](1)

		computing, release := make(chan struct{}), make(chan struct{})

		var (
			got int
			wg  sync.WaitGroup
		)

		wg.Go(func() {
			got = c.GetOrSet("key", func() int {
				close(computing)
				<-release

				return 1
			})
		})

		await(t, computing, "the caller reaching compute")
		c.Set("key", 2)
		close(release)
		awaitGroup(t, &wg, "the computing caller returning")

		assert.Equal(t, 2, got, "the value Set while computing has to stand")

		stored, ok := c.Get("key")
		require.True(t, ok)
		assert.Equal(t, 2, stored)
	})

	t.Run("GetOrSet releases a waiter with the Set that landed", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			c := cache.New[string, int](1)

			release := make(chan struct{})

			var (
				computed, waited int
				wg               sync.WaitGroup
			)

			wg.Go(func() {
				computed = c.GetOrSet("key", func() int {
					<-release

					return 1
				})
			})

			synctest.Wait()

			wg.Go(func() {
				waited = c.GetOrSet("key", func() int {
					t.Error("a waiter computed the value a second time")

					return 3
				})
			})

			// Settled, the waiter is parked on the claim rather than computing
			// a second value.
			synctest.Wait()

			c.Set("key", 2)
			close(release)
			wg.Wait()

			assert.Equal(t, 2, computed, "the value Set while computing has to stand")
			assert.Equal(t, 2, waited, "a waiter has to receive the value the key holds")
		})
	})

	t.Run("GetOrSet releases its waiters when the fill panics", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			c := cache.New[string, int](1)

			release := make(chan struct{})

			var (
				waited int
				wg     sync.WaitGroup
			)

			failingFill := func() int {
				<-release

				panic("fill failed")
			}

			wg.Go(func() {
				defer func() {
					assert.NotNil(t, recover(), "the panic has to reach the caller that computed")
				}()

				c.GetOrSet("key", failingFill)
			})

			synctest.Wait()

			wg.Go(func() {
				waited = c.GetOrSet("key", func() int { return 2 })
			})

			// Settled, the waiter is parked on the claim the panicking fill
			// holds. Releasing that fill has to release the waiter with it.
			synctest.Wait()

			close(release)
			wg.Wait()

			assert.Equal(t, 2, waited, "a waiter has to compute for itself once the fill panicked")

			stored, ok := c.Get("key")
			require.True(t, ok)
			assert.Equal(t, 2, stored)
		})
	})
}

func Test_Cache(t *testing.T) {
	t.Parallel()

	t.Run("Get miss", func(t *testing.T) {
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

	t.Run("Set and Get", func(t *testing.T) {
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
			wg.Go(func() {
				c.GetOrSet(i%10, func() int {
					return i
				})
			})
		}

		awaitGroup(t, &wg, "every caller returning")

		_, _, size := c.Stats()
		assert.Equal(t, uint64(10), size)
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

// syncTimeout bounds a step one test goroutine waits for another to reach. A
// step that never runs then names itself, where an unbounded receive would take
// the whole suite down with it.
const syncTimeout = 5 * time.Second

// awaitGroup waits for the group, and fails the test when a caller it holds
// never returns.
func awaitGroup(t *testing.T, wg *sync.WaitGroup, step string) {
	t.Helper()

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	await(t, done, step)
}

// await waits for c to deliver, and fails the test when the step that would
// have delivered never runs.
func await[T any](t *testing.T, c <-chan T, step string) {
	t.Helper()

	select {
	case <-c:
	case <-time.After(syncTimeout):
		t.Fatalf("%s did not happen", step)
	}
}
