# goroutines

Concurrency package emulating a tiny subset of Python's threading module. Implements safe and simple concurrency patterns with abort-on-error and cancellation. Provides simple syntax with golang generics, using only the standard library.

[![GoDoc](https://pkg.go.dev/badge/github.com/jake-dog/goroutines)](https://pkg.go.dev/github.com/jake-dog/goroutines)
[![Go Report Card](https://goreportcard.com/badge/github.com/jake-dog/goroutines)](https://goreportcard.com/report/github.com/jake-dog/goroutines)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/jake-dog/goroutines/blob/master/LICENSE)

## Key features

* Concurrent context/error-aware generic [map](https://en.wikipedia.org/wiki/Map_(higher-order_function)), [inject/reduce](https://en.wikipedia.org/wiki/Fold_(higher-order_function)), and more
* Reentrant generic function result cache and serializer
* Timed mutex implementation

## Installation

Use `go get` to install this library:
```
$ go get github.com/jake-dog/goroutines
```

## Usage example

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
		sum += r
	})
	fmt.Println("Sum of all strings processed:", sum)
}
```

## Performance notes

The size of the goroutine pool provided to `Map`, `Inject`, `Reduce`, `Search`, etc. also limits the internal buffer size. Unordered functions require less buffering, and are frequently much faster, as the results do not need to be reordered.

Unless a context is provided, error aware mapping functions can only terminate in-between processing of "slow" functions.  Context cancellation is preferred to terminate quickly.  Error aware mapping functions will return the first error encountered as the result of a function call or context cancellation.
