package workers

import (
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
type QueuedRunner struct {
	mu     *sync.Mutex
	fn     func() any
	l      []chan any
	state  int
	gen    int
	result any
	ttl    time.Duration
	added  time.Time
}

// NewQueuedRunner for the given function "fn"
func NewQueuedRunner(fn func() any) *QueuedRunner {
	return &QueuedRunner{
		mu: new(sync.Mutex),
		fn: fn,
	}
}

// NewCachedQueuedRunner for the given "fn" with cache ttl for results
func NewCachedQueuedRunner(fn func() any, ttl time.Duration) *QueuedRunner {
	return &QueuedRunner{
		mu:  new(sync.Mutex),
		fn:  fn,
		ttl: ttl,
	}
}

// RunTimeout runs or queues until timeout for the next result of "fn". If
// timeout is zero, return immediately with response or ErrRunnerTimeout when
// no result is available.  If timeout is positive return ErrRunnerTimeout
// when timeout occurs.  Wait for result when timeout is negative.
// Can be called from mulitple goroutines ensuring only one invocation of "fn"
// is active at a time.
func (qr *QueuedRunner) RunTimeout(timeout time.Duration) any {
	return qr.run(timeout, false)
}

// RunCachedTimeout mimics RunTimeout but returns a cached result if available
// Can be called from mulitple goroutines ensuring only one invocation of "fn"
// is active at a time.
func (qr *QueuedRunner) RunCachedTimeout(timeout time.Duration) any {
	return qr.run(timeout, true)
}

// Run or queue for the next result of "fn"
// Can be called from mulitple goroutines ensuring only one invocation of "fn"
// is active at a time.
func (qr *QueuedRunner) Run() any {
	return qr.run(-1, false)
}

// RunCached returns the next result of "fn", or cached value if available
// Can be called from mulitple goroutines ensuring only one invocation of "fn"
// is active at a time.
func (qr *QueuedRunner) RunCached() any {
	return qr.run(-1, true)
}

func (qr *QueuedRunner) run(timeout time.Duration, allowCached bool) any {
	var gen int
	qr.mu.Lock()

	if allowCached && qr.ttl > 0 && qr.result != nil && time.Since(qr.added) <= qr.ttl {
		defer qr.mu.Unlock()
		return qr.result
	}

	r := make(chan any, 1)
	if qr.state == running {
		qr.l = append(qr.l, r)
		gen = qr.gen
	} else {
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
			return v
		case <-t.C:
			qr.abort(gen, r)
			return ErrRunnerTimedout
		}
	} else if timeout == 0 {
		select {
		case v := <-r:
			return v
		default:
			qr.abort(gen, r)
			return ErrRunnerTimedout
		}
	}

	return <-r
}

func (qr *QueuedRunner) Flush() {
	qr.mu.Lock()
	qr.result = nil
	qr.added = zeroTime
	qr.mu.Unlock()
}

func (qr *QueuedRunner) pump() {
	v := qr.fn()

	qr.mu.Lock()
	defer qr.mu.Unlock()

	if qr.ttl > 0 {
		qr.result = v
		qr.added = time.Now()
	}

	for _, l := range qr.l {
		l <- v
		close(l)
	}
	qr.l = qr.l[:0]
	qr.state = stopped
}

func (qr *QueuedRunner) abort(gen int, r chan any) {
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
