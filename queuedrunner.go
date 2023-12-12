package goroutines

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	stopped = iota
	running
)

var (
	zeroTime = time.Time{}

	// ErrRunnerTimedout waiting for result
	ErrRunnerTimedout = errors.New("runner timed out")
)

// Coalescer is an instance of a coalesced function, ensuring only one
// invocation is running at a time. Behavior is similar to sync/singleflight
// with optional caching, and callers may individually abort early.
type Coalescer[T any] struct {
	mu     *sync.Mutex
	fn     func() (T, error)
	l      []chan [2]any
	state  int
	gen    int
	result T
	ttl    time.Duration
	grace  time.Duration
	added  time.Time
}

// Coalesce the given function.
func Coalesce[T any](fn func() (T, error)) *Coalescer[T] {
	return &Coalescer[T]{
		mu: new(sync.Mutex),
		fn: fn,
	}
}

// CacheCoalesce coalesces the given function with result cache ttl/grace.
// A cached result is returned if available and the result is not older than
// ttl+grace. If the result is older than ttl but younger than grace+ttl,
// a cached result is returned and a call to function is made asynchronously,
// refreshing the cached result. If returned error is non-nil then a result
// will not be cached.
func CacheCoalesce[T any](fn func() (T, error), ttl time.Duration, grace time.Duration) *Coalescer[T] {
	return &Coalescer[T]{
		mu:    new(sync.Mutex),
		fn:    fn,
		ttl:   ttl,
		grace: grace,
	}
}

// UncachedCoalescer wraps a Coalescer and bypasses caching.
type UncachedCoalescer[T any] struct {
	qr *Coalescer[T]
}

// TryRun but do not return cached values.
func (u UncachedCoalescer[T]) TryRun() (T, error) {
	return u.qr.run(context.Background(), 0, true)
}

// Run but do not return cached values.
func (u UncachedCoalescer[T]) Run() (T, error) {
	return u.qr.run(context.Background(), -1, true)
}

// RunWithContext but do not return cached values.
func (u UncachedCoalescer[T]) RunWithContext(ctx context.Context) (T, error) {
	return u.qr.run(ctx, -1, true)
}

// RunTimeout but do not return cached values.
func (u UncachedCoalescer[T]) RunTimeout(timeout time.Duration) (T, error) {
	return u.qr.run(context.Background(), timeout, true)
}

// TryRun returns immediately with a result if available or ErrRunnerTimedout.
//
// Identical to TryWithTimeout(0).
func (qr *Coalescer[T]) TryRun() (T, error) {
	return qr.run(context.Background(), 0, false)
}

// Run or queue for the next result.
func (qr *Coalescer[T]) Run() (T, error) {
	return qr.run(context.Background(), -1, false)
}

// RunWithContext runs or queues for the next result.
func (qr *Coalescer[T]) RunWithContext(ctx context.Context) (T, error) {
	return qr.run(ctx, -1, false)
}

// RunTimeout runs or queues until timeout for the next result. If
// timeout is zero, return immediately with response or ErrRunnerTimeout when
// no result is available.  If timeout is positive return ErrRunnerTimeout
// when timeout occurs.  Wait for result when timeout is negative.
func (qr *Coalescer[T]) RunTimeout(timeout time.Duration) (T, error) {
	return qr.run(context.Background(), timeout, false)
}

// NoCache returns the same Coalescer with cache bypass enabled
func (qr *Coalescer[T]) NoCache() UncachedCoalescer[T] {
	return UncachedCoalescer[T]{qr}
}

func (qr *Coalescer[T]) run(ctx context.Context, timeout time.Duration, noCache bool) (T, error) {
	var gen int
	qr.mu.Lock()

	if !noCache && qr.ttl > 0 && time.Since(qr.added) <= qr.ttl {
		defer qr.mu.Unlock()
		return qr.result, nil
	}

	if !noCache && qr.grace > 0 && time.Since(qr.added) <= qr.ttl+qr.grace {
		defer qr.mu.Unlock()
		if qr.state == running {
			return qr.result, nil
		}

		select {
		case <-ctx.Done():
			v := new(T)
			return *v, ctx.Err()
		default:
		}

		qr.state = running
		qr.gen = qr.gen + 1
		go qr.pump()
		return qr.result, nil
	}

	r := make(chan [2]any, 1)
	if qr.state == running {
		qr.l = append(qr.l, r)
		gen = qr.gen
	} else {
		select {
		case <-ctx.Done():
			v := new(T)
			return *v, ctx.Err()
		default:
		}

		qr.state = running
		qr.l = append(qr.l, r)
		qr.gen = qr.gen + 1
		gen = qr.gen
		go qr.pump()
	}
	qr.mu.Unlock()

	if timeout > 0 {
		t := time.NewTimer(timeout)
		select {
		case v := <-r:
			return qr.rvalue(v)
		case <-t.C:
			qr.abort(gen, r)
			v := new(T)
			return *v, ErrRunnerTimedout
		case <-ctx.Done():
			v := new(T)
			return *v, ctx.Err()
		}
	} else if timeout == 0 {
		select {
		case v := <-r:
			return qr.rvalue(v)
		case <-ctx.Done():
			v := new(T)
			return *v, ctx.Err()
		default:
			qr.abort(gen, r)
			v := new(T)
			return *v, ErrRunnerTimedout
		}
	}

	select {
	case v := <-r:
		return qr.rvalue(v)
	case <-ctx.Done():
		v := new(T)
		return *v, ctx.Err()
	}
}

func (qr *Coalescer[T]) rvalue(v [2]any) (T, error) {
	if v[1] != nil {
		return v[0].(T), v[1].(error)
	}
	return v[0].(T), nil
}

// Flush cached result.
func (qr *Coalescer[T]) Flush() {
	if qr.ttl > 0 || qr.grace > 0 {
		qr.mu.Lock()
		qr.added = zeroTime
		qr.mu.Unlock()
	}
}

// IsRunning returns true if function is running.
func (qr *Coalescer[T]) IsRunning() bool {
	var isrunning bool
	qr.mu.Lock()
	isrunning = qr.state == running
	qr.mu.Unlock()
	return isrunning
}

func (qr *Coalescer[T]) pump() {
	v, err := qr.fn()

	qr.mu.Lock()
	defer qr.mu.Unlock()

	if err == nil && (qr.ttl > 0 || qr.grace > 0) {
		qr.result = v
		qr.added = time.Now()
	}

	for _, l := range qr.l {
		l <- [2]any{v, err}
		close(l)
	}
	qr.l = qr.l[:0]
	qr.state = stopped
}

func (qr *Coalescer[T]) abort(gen int, r chan [2]any) {
	// Best effort cleanup if client aborts, otherwise GC handles it
	if qr.mu.TryLock() {
		defer qr.mu.Unlock()
		if gen != qr.gen || len(qr.l) == 0 {
			return
		}
		if len(qr.l) == 1 && qr.l[0] == r {
			qr.l = qr.l[:0]
			close(r)
		} else if qr.l[len(qr.l)-1] == r {
			qr.l = qr.l[:len(qr.l)-1]
			close(r)
		} else {
			n := -1
			for i, l := range qr.l {
				if l == r {
					n = i
					break
				}
			}
			if n > 0 {
				qr.l[n] = qr.l[len(qr.l)-1]
				qr.l = qr.l[:len(qr.l)-1]
				close(r)
			}
		}
	}
}
