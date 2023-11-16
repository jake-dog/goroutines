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

var zeroTime = time.Time{}

// ErrRunnerTimedout means a timeout occured waiting for result of "fn"
var ErrRunnerTimedout = errors.New("runner timed out")

// QueuedRunner serializes access to a function, allowing multiple goroutines
// to queue returning identical result to all waiters.  The caller must
// not retain or modify result unless safe to do so given function "fn".
type QueuedRunner[T any] struct {
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

// NewQueuedRunner for the given function "fn"
func NewQueuedRunner[T any](fn func() (T, error)) *QueuedRunner[T] {
	return &QueuedRunner[T]{
		mu: new(sync.Mutex),
		fn: fn,
	}
}

// NewCachedQueuedRunner for the given "fn" with cache ttl/grace for result.
// A cached result is returned if available and the result is not older than
// ttl+grace.  If the result is older than ttl but younger than grace+ttl,
// a cached result is returned and a call to "fn" is made concurrently,
// refreshing the cached result.  When run if error is non-nil then a result
// will not be cached.  To cache an error, the returned error must be nil and
// the value of result an error.
// The result may be concurrently accessed and must not be modified.
func NewCachedQueuedRunner[T any](fn func() (T, error), ttl time.Duration, grace time.Duration) *QueuedRunner[T] {
	return &QueuedRunner[T]{
		mu:    new(sync.Mutex),
		fn:    fn,
		ttl:   ttl,
		grace: grace,
	}
}

// UncachedQueuedRunner wraps a QueuedRunner and bypasses caching
type UncachedQueuedRunner[T any] struct {
	qr *QueuedRunner[T]
}

// TryRun but do not return cached values
func (u UncachedQueuedRunner[T]) TryRun() (T, error) {
	return u.qr.run(context.Background(), 0, true)
}

// Run but do not return cached values
func (u UncachedQueuedRunner[T]) Run() (T, error) {
	return u.qr.run(context.Background(), -1, true)
}

// RunWithContext but do not return cached values
func (u UncachedQueuedRunner[T]) RunWithContext(ctx context.Context) (T, error) {
	return u.qr.run(ctx, -1, true)
}

// RunWithTimeout but do not return cached values
func (u UncachedQueuedRunner[T]) RunWithTimeout(timeout time.Duration) (T, error) {
	return u.qr.run(context.Background(), timeout, true)
}

// TryRun returns immediately with a result if available or ErrRunnerTimedout.
// Identical to TryWithTimeout(0)
func (qr *QueuedRunner[T]) TryRun() (T, error) {
	return qr.run(context.Background(), 0, false)
}

// Run or queue for the next result of "fn"
// Can be called from mulitple goroutines ensuring only one invocation of "fn"
// is active at a time.
func (qr *QueuedRunner[T]) Run() (T, error) {
	return qr.run(context.Background(), -1, false)
}

// RunWithContextruns or queues for the next result of "fn".
// Can be called from mulitple goroutines ensuring only one invocation of "fn"
// is active at a time.
func (qr *QueuedRunner[T]) RunWithContext(ctx context.Context) (T, error) {
	return qr.run(ctx, -1, false)
}

// RunTimeout runs or queues until timeout for the next result of "fn". If
// timeout is zero, return immediately with response or ErrRunnerTimeout when
// no result is available.  If timeout is positive return ErrRunnerTimeout
// when timeout occurs.  Wait for result when timeout is negative.
// Can be called from mulitple goroutines ensuring only one invocation of "fn"
// is active at a time.
func (qr *QueuedRunner[T]) RunWithTimeout(timeout time.Duration) (T, error) {
	return qr.run(context.Background(), timeout, false)
}

// UncachedQueuedRunner returns the same QueuedRunner with cache bypass enabled
func (qr *QueuedRunner[T]) NoCache() UncachedQueuedRunner[T] {
	return UncachedQueuedRunner[T]{qr}
}

func (qr *QueuedRunner[T]) run(ctx context.Context, timeout time.Duration, noCache bool) (T, error) {
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

func (qr *QueuedRunner[T]) rvalue(v [2]any) (T, error) {
	if v[1] != nil {
		return v[0].(T), v[1].(error)
	}
	return v[0].(T), nil
}

func (qr *QueuedRunner[T]) Flush() {
	if qr.ttl > 0 || qr.grace > 0 {
		qr.mu.Lock()
		qr.added = zeroTime
		qr.mu.Unlock()
	}
}

func (qr *QueuedRunner[T]) IsRunning() bool {
	var isrunning bool
	qr.mu.Lock()
	isrunning = qr.state == running
	qr.mu.Unlock()
	return isrunning
}

func (qr *QueuedRunner[T]) pump() {
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

func (qr *QueuedRunner[T]) abort(gen int, r chan [2]any) {
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
