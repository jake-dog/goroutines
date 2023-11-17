package goroutines

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrSearchSuccess exits Search when desired element is found
	ErrSearchSuccess = errors.New("search concluded successfully")

	// ErrSearchFailure occurs when Search completes without ErrSearchSuccess
	ErrSearchFailure = errors.New("failed to locate element")
)

const defaultPoolSize = 10

// ordE is used to order elements in Map
type ordE[T any] struct {
	e T
	n int
}

// F is a function result of the form (T, error) passed through a channel.
type F[T any] struct {
	V T
	E error
}

// NewF simplifies creating F for functions returning (T, error).
func NewF[T any](v T, e error) *F[T] {
	return &F[T]{v, e}
}

// Return the original result of calling function.
func (p *F[T]) Return() (T, error) {
	return p.V, p.E
}

type runnable[I any, R any] struct {
	f       func(any) any
	input   chan I
	output  chan R
	workers int
}

func (r *runnable[I, R]) run(ctx context.Context, wg *sync.WaitGroup) {
OuterLoop:
	for {
		var d I
		var ok bool
		select {
		case <-ctx.Done():
			break OuterLoop
		case d, ok = <-r.input:
			if !ok {
				break OuterLoop
			}
		}

		select {
		case <-ctx.Done():
			break OuterLoop
		case r.output <- r.f(d).(R):
		}
	}
	wg.Done()
}

func newRunnable[I any, R any](qlen int, fn func(I) R) *runnable[I, R] {
	return &runnable[I, R]{
		f: func(a any) any {
			return any(fn(a.(I)))
		},
		input:   make(chan I, qlen),
		output:  make(chan R, qlen),
		workers: qlen,
	}
}

// Map function to each element of slice returning a channel of results.
// All results must be consumed or goroutines may leak.
//
// MapWithContext is preferred in cases where all results are not consumed.
func Map[I any, R any](qlen int, fn func(I) R, args []I) <-chan R {
	return MapWithContext(context.Background(), qlen, fn, args)
}

// MapUnordered is Map but results are returned as they complete.
func MapUnordered[I any, R any](qlen int, fn func(I) R, args []I) <-chan R {
	return MapUnorderedWithContext(context.Background(), qlen, fn, args)
}

// MapErr is an error aware Map.
// All results must be consumed or goroutines may leak.
//
// MapErrWithContext is preferred in cases where all results are not consumed.
// Call the returned function until bool is false to consume all results.
// If an error is returned, new arguments will not be processed and execution
// will return when all goroutines finish.
func MapErr[I any, R any](qlen int, fn func(I) (R, error), args []I) func() (R, error, bool) {
	return MapErrWithContext(context.Background(), qlen, fn, args)
}

// MapErrUnordered is MapErr but results are returned as they complete.
func MapErrUnordered[I any, R any](qlen int, fn func(I) (R, error), args []I) func() (R, error, bool) {
	return MapErrUnorderedWithContext(context.Background(), qlen, fn, args)
}

// Search with Map, returning the result if ErrSearchSuccess.
//
// If an error is returned, new arguments will not be processed and execution
// will return when all goroutines finish.
func Search[I any, R any](qlen int, fn func(I) (R, error), args []I) (R, error) {
	return SearchWithContext(context.Background(), qlen, fn, args)
}

// SearchUnordered is Search but results are searched as they complete.
func SearchUnordered[I any, R any](qlen int, fn func(I) (R, error), args []I) (R, error) {
	return SearchUnorderedWithContext(context.Background(), qlen, fn, args)
}

// Reduce returns a single value as the result of Map
// The reduction function "fni" runs serially as results are returned.
//
// If an error is returned, new arguments will not be processed and execution
// will return when all goroutines finish.
func Reduce[I any, R any](qlen int, fn func(I) (R, error), fni func(R, R) (R, error), args []I) (R, error) {
	return ReduceWithContext(context.Background(), qlen, fn, fni, args)
}

// ReduceUnordered is Reduce but results are processed as they complete.
func ReduceUnordered[I any, R any](qlen int, fn func(I) (R, error), fni func(R, R) (R, error), args []I) (R, error) {
	return ReduceUnorderedWithContext(context.Background(), qlen, fn, fni, args)
}

// Collect is Map but returns a slice instead of a channel.
//
// If an error is returned, new arguments will not be processed and execution
// will return when all goroutines finish.
func Collect[I any, R any](qlen int, fn func(I) (R, error), args []I) ([]R, error) {
	return CollectWithContext(context.Background(), qlen, fn, args)
}

// CollectWithContext is Collect but with a context.
func CollectWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) (R, error), args []I) ([]R, error) {
	return InjectWithContext(ctx, qlen, make([]R, 0, len(args)), fn, func(a []R, b R) ([]R, error) {
		return append(a, b), nil
	}, args)
}

// CollectUnordered is MapUnordered but returns a slice instead of a channel.
//
// If an error is returned, new arguments will not be processed and execution
// will return when all goroutines finish.
func CollectUnordered[I any, R any](qlen int, fn func(I) (R, error), args []I) ([]R, error) {
	return CollectUnorderedWithContext(context.Background(), qlen, fn, args)
}

// CollectUnorderedWithContext is CollectUnordered but with a context.
func CollectUnorderedWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) (R, error), args []I) ([]R, error) {
	return InjectUnorderedWithContext(ctx, qlen, make([]R, 0, len(args)), fn, func(a []R, b R) ([]R, error) {
		return append(a, b), nil
	}, args)
}

// Inject is like Reduce except an initial value can be supplied.
// The reduction function "fni" runs serially as results are returned.
//
// If an error is returned, new arguments will not be processed and execution
// will return when all goroutines finish.
func Inject[I any, R any, A any](qlen int, a A, fn func(I) (R, error), fni func(A, R) (A, error), args []I) (A, error) {
	return InjectWithContext(context.Background(), qlen, a, fn, fni, args)
}

// InjectUnordered is Inject but results are processed as they complete.
func InjectUnordered[I any, R any, A any](qlen int, a A, fn func(I) (R, error), fni func(A, R) (A, error), args []I) (A, error) {
	return InjectUnorderedWithContext(context.Background(), qlen, a, fn, fni, args)
}

// MapWithContext is Map but with a context.
// In all cases where processing of result channel may abort early, the context
// should be cancelled to avoid goroutine leaks.
func MapWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) R, args []I) <-chan R {
	return mapI(ctx, qlen, fn, args, nil)
}

// MapUnorderedWithContext is an unordered version of MapWithContext
func MapUnorderedWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) R, args []I) <-chan R {
	return mapUnordered(ctx, qlen, fn, args, nil)
}

// MapErrWithContext is MapErr but with a context.
// In all cases where processing of result channel may abort early, the context
// should be cancelled to avoid goroutine leaks.
func MapErrWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) (R, error), args []I) func() (R, error, bool) {
	return mapErr(ctx, true, qlen, fn, args)
}

// MapErrUnorderedWithContext is an unordered version of MapErrWithContext
func MapErrUnorderedWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) (R, error), args []I) func() (R, error, bool) {
	return mapErr(ctx, false, qlen, fn, args)
}

// SearchWithContext is Search but with a context.
func SearchWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) (R, error), args []I) (R, error) {
	return search(ctx, true, qlen, fn, args)
}

// SearchUnorderedWithContext is an unordered version of SearchWithContext.
func SearchUnorderedWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) (R, error), args []I) (R, error) {
	return search(ctx, false, qlen, fn, args)
}

// ReduceWithContext is Reduce but with a context.
func ReduceWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) (R, error), fni func(R, R) (R, error), args []I) (R, error) {
	a := new(R)
	return inject(ctx, true, qlen, *a, fn, fni, args)
}

// ReduceUnorderedWithContext is an unordered version of ReduceWithContext.
func ReduceUnorderedWithContext[I any, R any](ctx context.Context, qlen int, fn func(I) (R, error), fni func(R, R) (R, error), args []I) (R, error) {
	a := new(R)
	return inject(ctx, false, qlen, *a, fn, fni, args)
}

// InjectWithContext is Inject but with a context.
func InjectWithContext[I any, R any, A any](ctx context.Context, qlen int, a A, fn func(I) (R, error), fni func(A, R) (A, error), args []I) (A, error) {
	return inject(ctx, true, qlen, a, fn, fni, args)
}

// InjectUnorderedWithContext is an unordered version of InjectWithContext.
func InjectUnorderedWithContext[I any, R any, A any](ctx context.Context, qlen int, a A, fn func(I) (R, error), fni func(A, R) (A, error), args []I) (A, error) {
	return inject(ctx, false, qlen, a, fn, fni, args)
}

func search[I any, R any](ctx context.Context, ordered bool, qlen int, fn func(I) (R, error), args []I) (R, error) {
	var v R
	var err error
	hasError := make(chan error, len(args))

	mapFn := mapUnordered[I, *F[R]]
	if ordered {
		mapFn = mapI[I, *F[R]]
	}

	results := mapFn(ctx, qlen, func(in I) *F[R] {
		vn, errn := fn(in)
		if errn != nil {
			hasError <- errn
		}
		return NewF(vn, errn)
	}, args, hasError)
	for r := range results {
		if err != nil {
			continue // consume all results
		} else {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				continue
			default:
			}
		}
		v, err = r.Return()
	}

	if err == nil {
		select {
		case <-ctx.Done():
			return v, ctx.Err()
		default:
		}
		return v, ErrSearchFailure
	} else if err == ErrSearchSuccess {
		return v, nil
	}
	return v, err
}

func inject[I any, R any, A any](ctx context.Context, ordered bool, qlen int, a A, fn func(I) (R, error), fni func(A, R) (A, error), args []I) (A, error) {
	var v R
	var err error
	hasError := make(chan error, len(args))

	mapFn := mapUnordered[I, *F[R]]
	if ordered {
		mapFn = mapI[I, *F[R]]
	}

	results := mapFn(ctx, qlen, func(in I) *F[R] {
		vn, errn := fn(in)
		if errn != nil {
			hasError <- errn
		}
		return NewF(vn, errn)
	}, args, hasError)
	for r := range results {
		if err != nil {
			continue // consume all results
		}
		v, err = r.Return()
		if err != nil {
			continue
		} else {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				continue
			default:
			}
		}
		a, err = fni(a, v)
	}
	if err == nil {
		select {
		case <-ctx.Done():
			return a, ctx.Err()
		default:
		}
	}
	return a, err
}

func mapErr[I any, R any](ctx context.Context, ordered bool, qlen int, fn func(I) (R, error), args []I) func() (R, error, bool) {
	hasError := make(chan error, len(args))

	mapFn := mapUnordered[I, *F[R]]
	if ordered {
		mapFn = mapI[I, *F[R]]
	}

	results := mapFn(ctx, qlen, func(in I) *F[R] {
		vn, errn := fn(in)
		if errn != nil {
			hasError <- errn
		}
		return NewF(vn, errn)
	}, args, hasError)

	return func() (vn R, errn error, ok bool) {
		var r *F[R]
		select {
		case <-ctx.Done():
			return vn, ctx.Err(), ok
		case r, ok = <-results:
		}
		if r == nil {
			return
		}
		vn, errn = r.Return()
		if errn == nil {
			select {
			case <-ctx.Done():
				return vn, ctx.Err(), ok
			default:
			}
		}
		return vn, errn, ok
	}
}

func mapUnordered[I any, R any](ctx context.Context, qlen int, fn func(I) R, args []I, hasError <-chan error) <-chan R {
	// Save a bit on recompute
	poolSize := qlen
	if poolSize <= 0 {
		poolSize = defaultPoolSize
	}

	rn := newRunnable(poolSize, fn)

	go func() {
		// Save a bit on recompute
		argsLen := len(args)

		// Only fill pool as much as needed
		startSize := poolSize
		if startSize > argsLen {
			startSize = argsLen
		}

		var wg sync.WaitGroup
		wg.Add(startSize)

		// Startup the pool and fill it with work
		for i := 0; i < startSize; i++ {
			go rn.run(ctx, &wg) // start runners
			select {
			case <-hasError:
				goto EarlyExit
			case <-ctx.Done():
				goto EarlyExit
			case rn.input <- args[i]:
			}
		}

		for _, arg := range args[startSize:argsLen] {
			select {
			case <-hasError:
				goto EarlyExit
			case <-ctx.Done():
				goto EarlyExit
			case rn.input <- arg: // send until done
			}
		}

	EarlyExit:
		close(rn.input)
		wg.Wait()
		close(rn.output)
	}()

	return rn.output
}

func mapI[I any, R any](ctx context.Context, qlen int, fn func(I) R, args []I, hasError <-chan error) <-chan R {
	// Save a bit on recompute
	poolSize := qlen
	if poolSize <= 0 {
		poolSize = defaultPoolSize
	}

	results := make(chan R, poolSize)

	rn := newRunnable(poolSize, func(in *ordE[I]) *ordE[R] {
		return &ordE[R]{
			e: fn(in.e),
			n: in.n,
		}
	})

	go func(buf []*ordE[R]) {
		_ = buf[poolSize-1] // Eliminate bounds check

		// Save a bit on recompute
		argsLen := len(args)

		// Only fill pool as much as needed
		startSize := poolSize
		if startSize > argsLen {
			startSize = argsLen
		}

		var wg sync.WaitGroup
		wg.Add(startSize)

		// Startup the pool and fill it with work
		for i := 0; i < startSize; i++ {
			go rn.run(ctx, &wg) // start runners
			rn.input <- &ordE[I]{args[i], i}
		}

		if startSize == argsLen {
			close(rn.input) // all inputs are buffered
		}

		var readn, cidx int
		idx := startSize
	OuterLoopContext:
		for readn < argsLen {
			var r *ordE[R]
			select {
			case <-ctx.Done():
				if idx < argsLen {
					close(rn.input)
				}
				break OuterLoopContext
			case r = <-rn.output:
			}
			readn++

			// Add current element to results, or buffer if out-of-sequence
			if r.n == cidx {
				results <- r.e
				cidx++
			} else {
				buf[r.n%poolSize] = r
			}

			// Check for any buffered results to return
			for buf[cidx%poolSize] != nil {
				results <- buf[cidx%poolSize].e
				buf[cidx%poolSize] = nil
				cidx++
			}

			// Top off the pool
			for idx < argsLen && cidx+poolSize > idx {
				select {
				case <-hasError:
					argsLen = idx
				case <-ctx.Done():
					close(rn.input)
					break OuterLoopContext
				case rn.input <- &ordE[I]{args[idx], idx}:
				}
				idx++
				if idx >= argsLen {
					close(rn.input) // Close early to terminate workers
				}
			}
		}

		wg.Wait() // only blocks when context is cancelled

		// Cleanup and signal readers
		close(rn.output)
		close(results)
	}(make([]*ordE[R], poolSize))

	return results
}
