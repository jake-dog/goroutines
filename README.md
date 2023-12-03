# goroutines

Concurrency package emulating a tiny subset of Python's threading module. Implements safe and simple concurrency patterns with abort-on-error and cancellation. Functionality similar to [sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) and [sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight) with simpler syntax using golang generics.  Depends only on the standard library.

[![GoDoc](https://pkg.go.dev/badge/github.com/jake-dog/goroutines)](https://pkg.go.dev/github.com/jake-dog/goroutines)
[![Go Report Card](https://goreportcard.com/badge/github.com/jake-dog/goroutines)](https://goreportcard.com/report/github.com/jake-dog/goroutines)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/jake-dog/goroutines/blob/master/LICENSE)

## Key features

* Concurrent context/error-aware [map/collect](https://en.wikipedia.org/wiki/Map_(higher-order_function)), [inject/reduce](https://en.wikipedia.org/wiki/Fold_(higher-order_function)), etc.
* Reentrant function coalescing and result cache
* Mutex with lock timeout and context cancellation

## Installation

Use `go get` to install this library:
```
$ go get github.com/jake-dog/goroutines
```

## Usage example

#### Map

```go
package main

import (
	"fmt"

	"github.com/jake-dog/goroutines"
)

func main() {
	data := []string{"one", "two", "three", "four", "five", "six", "seven"}

	results := goroutines.Map(2, func(s string) int {
		fmt.Println("Processing:", s)
		return len(s)
	}, data)

	var sum int
	for r := range results { // All results must be consumed
		fmt.Println("Received:", r)
		sum += r
	}
	fmt.Println("Sum of all strings processed:", sum)
}
```

Try it on [playground](https://go.dev/play/p/E1glA7I1TCL).

#### Reduce

```go
package main

import (
	"fmt"

	"github.com/jake-dog/goroutines"
)

func main() {
	data := []string{"one", "two", "three", "four", "five", "six", "seven"}

	sum, _ := goroutines.Reduce(2, func(s string) (int, error) {
		fmt.Println("Processing:", s)
		return len(s), nil
	}, func(a, b int) (int, error) {
		fmt.Println("Received:", b)
		return a + b, nil
	}, data)

	fmt.Println("Sum of all strings processed:", sum)
}
```

Try it on [playground](https://go.dev/play/p/BIgC0sUrUnu).

## Performance notes

The size of the goroutine pool provided to `Map`, `Inject`, `Reduce`, `Search`, etc. also limits the internal buffer size. Unordered functions require less buffering, and are frequently much faster, as the results do not need to be reordered.

Unless a context is provided error aware mapping functions can only terminate in-between processing of "slow" functions.  Context cancellation is preferred to terminate quickly.  Error aware mapping functions will return the first error encountered as the result of a function call or context cancellation.
