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

func BenchmarkMap(b *testing.B) {
	arr := sliceGenerator(1000000)
	nproc := runtime.NumCPU()

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

	b.Run("lo.Map", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_ = lo.Map(arr, func(x int64, i int) string {
				return strconv.FormatInt(x, 10)
			})
		}
	})

	b.Run("lo/parallel.Map", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			_ = lop.Map(arr, func(x int64, i int) string {
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
