package directive_test

import (
	"slices"
	"strconv"
	"testing"

	"dev.gaijin.team/go/exhaustruct/v4/internal/directive"
)

func uniqStableMap[S ~[]T, T comparable](slice S) S { //nolint:ireturn
	if len(slice) <= 1 {
		return slice
	}

	m := make(map[T]struct{}, len(slice))
	s := make(S, 0, len(slice))

	for _, entry := range slice {
		if _, ok := m[entry]; !ok {
			m[entry] = struct{}{}

			s = append(s, entry)
		}
	}

	return s
}

func uniqStableContains[S ~[]T, T comparable](slice S) S { //nolint:ireturn
	if len(slice) <= 1 {
		return slice
	}

	result := make(S, 0, len(slice))

	for _, entry := range slice {
		if !slices.Contains(result, entry) {
			result = append(result, entry)
		}
	}

	return result
}

//nolint:modernize // intentionally using manual loop for benchmark comparison
func uniqStableLoop[S ~[]T, T comparable](slice S) S { //nolint:ireturn
	if len(slice) <= 1 {
		return slice
	}

	result := make(S, 0, len(slice))

	for _, entry := range slice {
		found := false

		for _, r := range result {
			if r == entry {
				found = true

				break
			}
		}

		if !found {
			result = append(result, entry)
		}
	}

	return result
}

func generateTestData(size int) directive.Directives {
	directives := []directive.Directive{directive.Ignore, directive.Enforce, directive.Optional}
	result := make(directive.Directives, size)

	for i := range size {
		result[i] = directives[i%len(directives)]
	}

	return result
}

/*
*
goos: darwin
goarch: arm64
pkg: dev.gaijin.team/go/exhaustruct/v4/internal/directive
cpu: Apple M4 Max
Benchmark_UniqStable
Benchmark_UniqStable/Map/10
Benchmark_UniqStable/Map/10-16         	 4793421	       247.7 ns/op	     616 B/op	       4 allocs/op
Benchmark_UniqStable/Contains/10
Benchmark_UniqStable/Contains/10-16    	20663246	        57.82 ns/op	     160 B/op	       1 allocs/op
Benchmark_UniqStable/Loop/10
Benchmark_UniqStable/Loop/10-16        	18978907	        62.82 ns/op	     160 B/op	       1 allocs/op
Benchmark_UniqStable/Map/100
Benchmark_UniqStable/Map/100-16        	  822258	      1307 ns/op	    5288 B/op	       4 allocs/op
Benchmark_UniqStable/Contains/100
Benchmark_UniqStable/Contains/100-16   	 2897716	       413.6 ns/op	    1792 B/op	       1 allocs/op
Benchmark_UniqStable/Loop/100
Benchmark_UniqStable/Loop/100-16       	 2689054	       446.7 ns/op	    1792 B/op	       1 allocs/op
Benchmark_UniqStable/Map/1000
Benchmark_UniqStable/Map/1000-16       	   96571	     12402 ns/op	   70992 B/op	       6 allocs/op
Benchmark_UniqStable/Contains/1000
Benchmark_UniqStable/Contains/1000-16  	  335102	      3543 ns/op	   16384 B/op	       1 allocs/op
Benchmark_UniqStable/Loop/1000
Benchmark_UniqStable/Loop/1000-16      	  302566	      3999 ns/op	   16384 B/op	       1 allocs/op
PASS.
*/
func Benchmark_UniqStable(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		data := generateTestData(size)

		b.Run("Map/"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = uniqStableMap(data)
			}
		})

		b.Run("Contains/"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = uniqStableContains(data)
			}
		})

		b.Run("Loop/"+strconv.Itoa(size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_ = uniqStableLoop(data)
			}
		})
	}
}
