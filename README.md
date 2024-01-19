# goroutines

[![GoDoc](https://pkg.go.dev/badge/github.com/jake-dog/goroutines)](https://pkg.go.dev/github.com/jake-dog/goroutines)
[![Go Report Card](https://goreportcard.com/badge/github.com/jake-dog/goroutines)](https://goreportcard.com/report/github.com/jake-dog/goroutines)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/jake-dog/goroutines/blob/master/LICENSE)
![tests](https://github.com/jake-dog/goroutines/actions/workflows/go.yml/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/jake-dog/goroutines/graph/badge.svg?token=5VCRFWBTXN)](https://codecov.io/gh/jake-dog/goroutines)

Safe and simple "batteries included" concurrency with abort-on-error and cancellation. Functionality is similar to [sourcegraph/conc](https://github.com/sourcegraph/conc), providing ergonomic alternatives to [sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) and [sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight). Only dependency is on the standard library.

This package is not [yet](https://github.com/panjf2000/ants) [another](https://github.com/Jeffail/tunny) [goroutine](https://github.com/gammazero/workerpool) [pool](https://github.com/go-tomb/tomb), but instead a generic concurrency library taking full advantage of idiomatic go. Goroutines are lightweight compared to system threads and rarely require pool or lifecycle management. However, it may still be useful in conjunction with goroutine pools for specific patterns and style preferences.

#### Install

```shell
go get github.com/jake-dog/goroutines
```

#### Why the name `goroutines`?

[_A package’s name provides context for its contents._](https://go.dev/blog/package-names) It follows from [`sync.Map`](https://pkg.go.dev/sync#Map), a synchronized map implementation, that `goroutines.Map` maps a function into one or more goroutines.

## Key features

* [Concurrent mapping](#concurrent-mapping)
* [Timed mutex](#timed-mutex)
* [Function coalescing](#function-coalescing)

## Concurrent Mapping

Mapping functions are [higher-order functions](https://en.wikipedia.org/wiki/Higher-order_function) where each element is processed in a separate goroutine. The number of goroutines used is limited to the number specified as the first argument to all mapping functions, or second argument for context aware variants. Mapping functions provide safer alternatives to [`sync/errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup).

```go
groceries, _ := goroutines.Collect(3, func(a string) (int, error) {
	return len(a), nil
}, []string{"eggs", "onion", "cheese"})
// groceries == []int{4, 5, 6}
```

Some mapping functions, like `Inject` and `Reduce`, take a second function argument which is run serially to aggregate or transform results from the mapped function. While this style is atypical it provides great control over compute parallelism and result transformations (eg. [flatMap](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/flatMap)). For instance it may be efficient to send HTTP requests concurrently to retrieve large JSON responses, but parse and extract values in a single goroutine.

```go
chars, _ := goroutines.Reduce(3, func(a string) ([]byte, error) {
	return []byte(a), nil        // run concurrently
}, func(a []byte, b []byte) ([]byte, error) {
	return append(a, b...), nil  // run serially (to flatten)
}, []string{"eggs", "onion", "cheese"})
// string(chars) == "eggsonioncheese"
```

Error aware mapping functions will ignore unprocessed items and wait for active goroutines to terminate if the mapped function returns an error. Additionally, context aware mapping functions will terminate all active goroutines when the context is cancelled. Unordered function variants are [faster](#benchmarks) than ordered functions, but return results in the order they arrive rather than the order of the input. See: [performance notes](#performance-notes).

**Generic**

* [Map](#map)
* MapUnordered

**Error aware**

* [Collect](#collect)
* CollectUnordered
* [ForEach](#foreach)
* ForEachUnordered
* [Inject](#inject)
* InjectUnordered
* [MapErr](#maperr)
* MapErrUnordered
* [Reduce](#reduce)
* ReduceUnordered
* [Search](#search)
* SearchUnordered

**Error and context aware**

* CollectWithContext
* CollectUnorderedWithContext
* ForEachWithContext
* ForEachUnorderedWithContext
* InjectWithContext
* InjectUnorderedWithContext
* MapErrWithContext
* MapErrUnorderedWithContext
* ReduceWithContext
* ReduceUnorderedWithContext
* SearchWithContext
* SearchUnorderedWithContext

## Timed Mutex

`TimedMutex` has no direct programming analog though similar implementations can be found in many golang packages (including [`sync/errgroup`](https://cs.opensource.google/go/x/sync/+/refs/tags/v0.6.0:errgroup/errgroup.go;l=70-72)). It is neither a reentrant mutex, nor a traditional mutex, though it includes the `sync.Mutex` interface. What makes `TimedMutex` distinct is that it can be used when an operation has a concurrency limit and/or the operation under lock takes an indeterminate amount of time. This is particularly useful, for instance, in limiting connections via a custom `net.Listener`, or when serializing access to a resource while serving HTTP.

```go
// LimitHandler wraps a handler and limits concurrent execution.
func LimitHandler(h http.Handler, limit int, timeout time.Duration) http.Handler {
	mu := goroutines.NewVariableTimedMutex(limit)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := context.WithTimeout(r.Context(), timeout)
		if err := mu.LockWithContext(ctx); err != nil {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		h.ServeHTTP(w, r)
		mu.Unlock()
	})
}
```

## Function Coalescing

Coalescing combines concurrent calls to a function, returning the same result to all callers. As the result may be accessed by multiple callers simultaneously, it should not be modified. Coalescing behavior is comparable to [`sync/singleflight`](https://pkg.go.dev/golang.org/x/sync/singleflight) except:

* results can be cached, and optionally include [grace time](https://info.varnish-software.com/blog/two-minute-tech-tuesdays-grace-mode)
* each coalesced function is a separate `Coalescer`
* results use golang generics and do not need to be unwrapped

```go
invocations := new(atomic.Uint64)
callers := new(atomic.Uint64)

fn := goroutines.CacheCoalesce(func() ([]byte, error) {
	time.Sleep(1 * time.Second) // some expensive operation
	invocations.Add(1)
	return []byte("some expensive payload"), nil
}, 10 * time.Second, 5 * time.Second)

for i := 0; i < 10; i++ {
	go func() {
		_, _ = fn.Run() // concurrently call function
		callers.Add(1)
	}()
	time.Sleep(200 * time.Millisecond)
}

invoked := invocations.Load();
called := callers.Load()
fmt.Println("Coalesced function invoked", invoked, "time(s), but called", called, "times")
// "Coalesced function invoked 1 time(s), but called 10 times\n"
```

Caching is passive so old values will continue to occupy memory unless `Flush()` is called. Calling `NoCache()` on a `Coalescer` returns a wrapped instance of the same `Coalescer` with caching disabled. It may be useful to employ a cached coalescer for read operations, and an uncached wrapper of the same coalescer for write or update operations.

## Examples

### Collect

Concurrently invokes function on elements in slice, returning a new slice of the given type and the first error, if any, that occurs.

```go
r, _ := goroutines.Collect(2, func(s string) (int, error) {
	return len(s), nil
}, []string{"eggs", "onion", "cheese"})
// []int{4, 5, 6}
```

### ForEach

Concurrently invokes function on each element in slice returning the first error, if any, that occurs.

```go
_ = goroutines.ForEach(2, func(s string) error {
	fmt.Println(s)
	return nil
}, []string{"eggs","onion","cheese"})
// "eggs\nonion\ncheese\n"
```

### Inject

Inject is like [`Reduce`](#reduce) but allows an initial value of an arbitrary type to be specified.

```go
r, _ := goroutines.Inject(2, []string{"grocery"}, func(s string) ([2]string, error) {
	return [2]string{s, strconv.Itoa(len(s))}, nil
}, func(a []string, b [2]string) ([]string, error) {
	return append(a, b[:]...), nil
}, []string{"eggs", "onion", "cheese"})
// []string{"grocery", "eggs", "4", "onion", "5", "cheese", "6"}
```

### Map

Concurrently invokes function on each element in slice. Returns a channel of results which must be read until close to avoid leaking goroutines. Consider instead a cancellable variant, like `MapErrWithContext`.

```go
results := goroutines.Map(2, func(i int) rune {
	return rune(1<<i)
}, []int{4, 5, 6, 7, 8})

var sum rune
for r := range results { // all results must be consumed
	sum += r
}
// 496
```

### MapErr

Concurrently execute function on each element in slice. Returns a function which must be called until a non-nil error,  or the last bool value is false, to avoid leaking goroutines.

```go
next := goroutines.MapErr(2, func(i int) (rune, error) {
	return rune(1<<i), nil
}, []int{4, 5, 6, 7, 8})

var sum rune
for r, _, ok := next(); ok; r, _, ok = next() {
	sum += r
}
// 496
```

### Reduce

Concurrently invokes the first function with each element of slice, then runs the second function serially to aggregate results.

```go
sum, _ := goroutines.Reduce(2, func(s string) (int, error) {
	return len(s), nil
}, func(a, b int) (int, error) {
	return a + b, nil
}, []string{"eggs", "onion", "cheese"})
// 15
```

### Search

Concurrently invokes function with each element of slice, returning the first result when `ErrSearchSuccess`, or the first non-nil error is encountered. If all elements are processed without an error, `ErrSearchFailure` is returned.

```go
r, _ := goroutines.Search(2, func(s string) (string, error) {
	if len(s) == 6 {
		return "I love "+s, goroutines.ErrSearchSuccess
	}
	return "", nil
}, []string{"eggs","onion","cheese"})
// "I love cheese"
```

## Performance notes

The size of the goroutine pool provided to concurrent mapping functions (`Map`, `Inject`, `Reduce`, `Search`, etc.) also limits the internal buffer size. Unordered functions require less buffering, and are frequently much faster, as the results do not need to be reordered.

Unless a context is provided, error aware mapping functions can only terminate in-between processing of "slow" functions.  Context cancellation is preferred to terminate quickly. Error aware mapping functions will return the first error encountered as the result of a function call or context cancellation.

Mapping functions spawn up to the number of goroutines requested for the pool size, plus one goroutine to stoke the input buffer channel. The pattern maintains a consistent performance profile by reducing GC and scheduler overhead, due to goroutine churn, at the cost of extra channel operations.

Concurrent mapping functions are particularly useful for I/O bound activities-- within limits. Initiating too many I/O requests simultaneously can cause resource exhaustion on devices and external systems. CPU bound workloads do not benefit from increasing mapping pool size beyond the number of cores, or threads (`GOMAXPROCS`) when lower.

## Benchmarks

Two types of benchmarks are included utilizing the technique of [sambe/lo](https://github.com/samber/lo/blob/3283fe13ea3134bd8bca074c24d353211916d834/README.md#-benchmark), pretty much just:

```go
_, _ = goroutines.Collect(runtime.NumCPU(), func(x int64) (string, error) {
	return strconv.FormatInt(x, 10), nil
}, arr)
```

 The first set of benchmarks is parallel compute optimized (`BenchmarkMapEfficient`), running a for-loop to process list items in each system thread. The second set (`BenchmarkMap`), which is identical to that of `samber/lo`, tests the speed of mapping work onto goroutines.  Benchmarks are accessible on the [benchmarking branch](https://github.com/jake-dog/goroutines/blob/benchmarking/benchmark_test.go).

```shell
$ BENCHTIME=6s make bench
goos: linux
goarch: amd64
pkg: github.com/jake-dog/goroutines
cpu: AMD Ryzen 7 PRO 5850U with Radeon Graphics
BenchmarkMapEfficient/goroutines.Map-16               770     8206191 ns/op    39998869 B/op   1000003 allocs/op
BenchmarkMapEfficient/goroutines.MapUnordered-16      840     8693458 ns/op    39998856 B/op   1000003 allocs/op
BenchmarkMapEfficient/goroutines.Collect-16           650    10910569 ns/op    39998950 B/op   1000004 allocs/op
BenchmarkMapEfficient/goroutines.CollectUnordered-16  782    10339897 ns/op    39998915 B/op   1000003 allocs/op
BenchmarkMapEfficient/lo/parallel.Map-16              759     8386293 ns/op    39999403 B/op   1000004 allocs/op
BenchmarkMapEfficient/sync/errgroup-16                812     9576724 ns/op    39998883 B/op   1000003 allocs/op
BenchmarkMap/goroutines.Map-16                          8   811953972 ns/op    80001012 B/op   3000034 allocs/op
BenchmarkMap/goroutines.MapUnordered-16                15   460896523 ns/op    64000413 B/op   3000027 allocs/op
BenchmarkMap/goroutines.Collect-16                      6  1141937176 ns/op   120010392 B/op   4000063 allocs/op
BenchmarkMap/goroutines.CollectUnordered-16             9   667950231 ns/op    96008560 B/op   3000042 allocs/op
BenchmarkMap/sync/errgroup-16                          10   633464807 ns/op   111998742 B/op   3000005 allocs/op
BenchmarkMap/lo/parallel.Map-16                        15   456819421 ns/op   135998440 B/op   3000002 allocs/op
BenchmarkMap/lo.Map-singlethreaded-16                 144    49399065 ns/op    39998424 B/op   1000001 allocs/op
BenchmarkMap/funk.Map-singlethreaded-16                20   351858069 ns/op   176027896 B/op   4000041 allocs/op
BenchmarkMap/for-singlethreaded-16                    147    48856922 ns/op    39998424 B/op   1000001 allocs/op
PASS
ok      github.com/jake-dog/goroutines  145.602s
```

**Warning:** each second of benchmark time takes about 5GB of memory for _most_ compute optimized benchmarks.

#### Results

* both compute optimized and multi-threaded benchmarks show generally insignificant performance differences
* single-threaded `lo.Map`, or a for-loop, vastly outperform concurrent functions when concurrency is not needed
* `lop.Map` incurs additional scheduler/GC overhead which may limit CPU bound workloads, as it is the only mapping function included that lacks a concurrency limit
* when system memory limits result size, `Map`, `MapUnordered`, and `MapErr` provide a consistent memory profile by streaming results
* `errgroup` can also limit memory consumption but it is not inherent to the implementation and would increase CPU cost

## Why?

While it would be a stretch to suggest that golang makes concurrency difficult, it isn't always easy either. Channels are, for instance, [notoriously finicky](https://www.jtolio.com/2016/03/go-channels-are-bad-and-you-should-feel-bad/). Many languages have built-in libraries for common concurrency patterns. Python has the [`multiprocessing`](https://docs.python.org/3/library/multiprocessing.html#multiprocessing.pool.ThreadPool) module, the original inspiration for this package, as well as a newer [`concurrent.futures`](https://docs.python.org/3/library/concurrent.futures.html#concurrent.futures.Executor.map) module. Then again, Python also has [`asyncio`](https://docs.python.org/3.10/library/asyncio.html) which suffers from the [color of functions](https://journal.stuffwithstuff.com/2015/02/01/what-color-is-your-function/) issue. So a direct port of some other concurrency library to golang isn't necessarily logical, nor practical when considering golang's unusual type system and concurrency model.

These points are surely what drove google engineers to develop [sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) and [sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight), though some rough edges in the current implementations of said packages may be why they are not yet incorporated into the standard library. Golang 1.18 added [generics](https://go.googlesource.com/proposal/+/HEAD/design/43651-type-parameters.md) which, despite introducing an entirely new set of quirks, provides the basis for new concurrency tools in golang. Amongst those packages attempting to fill the concurrency tooling gap using generics are, [sourcegraph/conc](https://pkg.go.dev/github.com/sourcegraph/conc#section-readme) and this library. (If there are others, I can't find them.)
