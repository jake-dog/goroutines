package goroutines

import (
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"github.com/thoas/go-funk"
	"golang.org/x/sync/errgroup"
)

func sliceGenerator(size uint) []int64 {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	result := make([]int64, size)

	for i := uint(0); i < size; i++ {
		result[i] = r.Int63()
	}

	return result
}

func BenchmarkMapEfficient(b *testing.B) {
	arr := sliceGenerator(1000000)
	nproc := runtime.NumCPU()

	b.Run("goroutines.Map", func(b *testing.B) {
		m := make([][]int64, b.N)
		for i, _ := range m {
			m[i] = arr
		}
		r := Map(nproc, func(a []int64) []string {
			results := make([]string, len(a))

			for i, item := range a {
				result := strconv.FormatInt(item, 10)
				results[i] = result
			}
			return results
		}, m)
		for _ = range r {
		}
	})

	b.Run("goroutines.MapUnordered", func(b *testing.B) {
		m := make([][]int64, b.N)
		for i, _ := range m {
			m[i] = arr
		}
		r := MapUnordered(nproc, func(a []int64) []string {
			results := make([]string, len(a))

			for i, item := range a {
				result := strconv.FormatInt(item, 10)
				results[i] = result
			}
			return results
		}, m)
		for _ = range r {
		}
	})

	b.Run("goroutines.Collect", func(b *testing.B) {
		m := make([][]int64, b.N)
		for i, _ := range m {
			m[i] = arr
		}
		_, _ = Collect(nproc, func(a []int64) ([]string, error) {
			results := make([]string, len(a))

			for i, item := range a {
				result := strconv.FormatInt(item, 10)
				results[i] = result
			}
			return results, nil
		}, m)
	})

	b.Run("goroutines.CollectUnordered", func(b *testing.B) {
		m := make([][]int64, b.N)
		for i, _ := range m {
			m[i] = arr
		}
		_, _ = CollectUnordered(nproc, func(a []int64) ([]string, error) {
			results := make([]string, len(a))

			for i, item := range a {
				result := strconv.FormatInt(item, 10)
				results[i] = result
			}
			return results, nil
		}, m)
	})

	b.Run("lo/parallel.Map", func(b *testing.B) {
		m := make([][]int64, b.N)
		for i, _ := range m {
			m[i] = arr
		}
		_ = lop.Map(m, func(a []int64, i int) []string {
			results := make([]string, len(a))

			for i, item := range a {
				result := strconv.FormatInt(item, 10)
				results[i] = result
			}
			return results
		})
	})

	b.Run("sync/errgroup", func(b *testing.B) {
		r := make([][]string, b.N)
		g := &errgroup.Group{}
		g.SetLimit(nproc)
		for n := 0; n < b.N; n++ {
			n := n // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func() error {
				results := make([]string, len(arr))

				for i, item := range arr {
					result := strconv.FormatInt(item, 10)
					results[i] = result
				}
				r[n] = results
				return nil
			})
		}
		_ = g.Wait()
	})
}

func BenchmarkMap(b *testing.B) {
	arr := sliceGenerator(1000000)
	nproc := runtime.NumCPU()

	b.Run("goroutines.Map", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			results := make([]string, 0, len(arr))
			r := Map(nproc, func(x int64) string {
				return strconv.FormatInt(x, 10)
			}, arr)
			for v := range r {
				results = append(results, v)
			}
		}
	})

	b.Run("goroutines.MapUnordered", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			results := make([]string, 0, len(arr))
			r := MapUnordered(nproc, func(x int64) string {
				return strconv.FormatInt(x, 10)
			}, arr)
			for v := range r {
				results = append(results, v)
			}
		}
	})

	b.Run("goroutines.Collect", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_, _ = Collect(nproc, func(x int64) (string, error) {
				return strconv.FormatInt(x, 10), nil
			}, arr)
		}
	})

	b.Run("goroutines.CollectUnordered", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_, _ = CollectUnordered(nproc, func(x int64) (string, error) {
				return strconv.FormatInt(x, 10), nil
			}, arr)
		}
	})

	b.Run("sync/errgroup", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			results := make([]string, len(arr))
			g := &errgroup.Group{}
			for i, x := range arr {
				g.Go(func() error {
					results[i] = strconv.FormatInt(x, 10)
					return nil
				})
			}
		}
	})

	b.Run("lo/parallel.Map", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_ = lop.Map(arr, func(x int64, i int) string {
				return strconv.FormatInt(x, 10)
			})
		}
	})

	b.Run("lo.Map", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_ = lo.Map(arr, func(x int64, i int) string {
				return strconv.FormatInt(x, 10)
			})
		}
	})

	b.Run("funk.Map", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_ = funk.Map(arr, func(x int64) string {
				return strconv.FormatInt(x, 10)
			})
		}
	})

	b.Run("for", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			results := make([]string, len(arr))

			for i, item := range arr {
				result := strconv.FormatInt(item, 10)
				results[i] = result
			}
		}
	})
}
