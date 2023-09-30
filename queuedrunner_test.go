package workers

import (
	"testing"
	"time"
	"sync/atomic"
)

func TestQueuedRunner(t *testing.T) {
	finished := new(atomic.Uint64)
	tests := []struct {
		fn    func(*QueuedRunner)
		sleep time.Duration
		gen   int
		qlen  int
	}{
		{
			fn: func(qr *QueuedRunner) {
				v := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 1,
		},
		{
			fn: func(qr *QueuedRunner) {
				v := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 2,
		},
		{
			fn: func(qr *QueuedRunner) {
				v := qr.RunTimeout(0)
				if v != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 2,
		},
		{
			fn: func(qr *QueuedRunner) {
				v := qr.RunTimeout(5*time.Millisecond)
				if v != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 2,
		},
		{
			fn: func(qr *QueuedRunner) {
				v := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 500 * time.Millisecond,
			gen: 1,
			qlen: 3,
		},
		{
			fn: func(qr *QueuedRunner) {
				v := qr.RunTimeout(1 * time.Second)
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 2,
			qlen: 1,
		},
		{
			fn: func(qr *QueuedRunner) {
				v := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 200 * time.Millisecond,
			gen: 2,
			qlen: 2,
		},
		{
			fn: func(_ *QueuedRunner) {
			},
			sleep: 0,
			gen: 2,
			qlen: 0,
		},
	}
	q := NewQueuedRunner(func() any {
		time.Sleep(100 * time.Millisecond)
		return "foo"
	})
	for _, tt := range tests {
		go tt.fn(q)
		time.Sleep(10 * time.Millisecond) // Sleep to let goroutine run
		q.mu.Lock()
		if tt.gen != q.gen {
			t.Errorf("Expected generation=%v but received generation=%v", tt.gen, q.gen)
		}
		if tt.qlen != len(q.l) {
			t.Errorf("Expected qlen=%v but received qlen=%v", tt.qlen, len(q.l))
		}
		q.mu.Unlock()
		if tt.sleep > 0 {
			time.Sleep(tt.sleep)
		}
	}
	if numResults := finished.Load(); numResults != uint64(len(tests)-1) {
		t.Errorf("Expected results in tests=%d, but received results=%d", len(tests)-1, numResults)
	}
}
