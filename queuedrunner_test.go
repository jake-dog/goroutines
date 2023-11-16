package goroutines

import (
	"testing"
	"time"
	"runtime"
	"sync/atomic"
)

type mystruct struct{
	biz string
	boz int
}

func TestQueuedRunner(t *testing.T) {
	finished := new(atomic.Uint64)
	tests := []struct {
		fn    func(*QueuedRunner[*mystruct])
		sleep time.Duration
		gen   int
		qlen  int
	}{
		{
			fn: func(qr *QueuedRunner[*mystruct]) {
				v, err := qr.Run()
				if v == nil || v.biz != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 1,
		},
		{
			fn: func(qr *QueuedRunner[*mystruct]) {
				v, err := qr.Run()
				if v == nil || v.biz != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 2,
		},
		{
			fn: func(qr *QueuedRunner[*mystruct]) {
				v, err := qr.TryRun()
				if err != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", err)
				}
				if v != nil {
					t.Errorf("Expected v=nil but received v=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 2,
		},
		{
			fn: func(qr *QueuedRunner[*mystruct]) {
				v, err := qr.RunWithTimeout(5*time.Millisecond)
				if err != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", err)
				}
				if v != nil {
					t.Errorf("Expected v=nil but received v=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 2,
		},
		{
			fn: func(qr *QueuedRunner[*mystruct]) {
				v, err := qr.Run()
				if v == nil || v.biz != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 500 * time.Millisecond,
			gen: 1,
			qlen: 3,
		},
		{
			fn: func(qr *QueuedRunner[*mystruct]) {
				v, err := qr.RunWithTimeout(1 * time.Second)
				if v == nil || v.biz != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 2,
			qlen: 1,
		},
		{
			fn: func(qr *QueuedRunner[*mystruct]) {
				v, err := qr.Run()
				if v == nil || v.biz != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 200 * time.Millisecond,
			gen: 2,
			qlen: 2,
		},
		{
			fn: func(_ *QueuedRunner[*mystruct]) {
			},
			sleep: 0,
			gen: 2,
			qlen: 0,
		},
	}
	q := NewQueuedRunner(func() (*mystruct, error) {
		time.Sleep(100 * time.Millisecond)
		return &mystruct{"foo",10}, nil
	})
	for _, tt := range tests {
		go tt.fn(q)
		runtime.Gosched()
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

func TestCachedQueuedRunner(t *testing.T) {
	finished := new(atomic.Uint64)
	tests := []struct {
		name    string
		fn      func(*QueuedRunner[string])
		sleep   time.Duration
		gen     int
		qlen    int
		running bool
	}{
		{
			name: "t1_uncached",
			fn: func(qr *QueuedRunner[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 1,
			running: true,
		},
		{
			name: "t2_uncached",
			fn: func(qr *QueuedRunner[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 2,
			running: true,
		},
		{
			name: "t3_uncached_immediate",
			fn: func(qr *QueuedRunner[string]) {
				v, err := qr.RunWithTimeout(0)
				if err != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", err)
				}
				if v != "" {
					t.Errorf("Expected v=nil but received v=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 2,
			running: true,
		},
		{
			name: "t4_uncached_nearimmediate",
			fn: func(qr *QueuedRunner[string]) {
				v, err := qr.RunWithTimeout(20*time.Millisecond)
				if err != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", err)
				}
				if v != "" {
					t.Errorf("Expected v=nil but received v=%v", v)
				}
				finished.Add(1)
			},
			sleep: 250 * time.Millisecond,
			gen: 1,
			qlen: 3,
			running: true,
		},
		{
			name: "t5_cached",
			fn: func(qr *QueuedRunner[string]) {
				v, err := qr.TryRun()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen: 1,
			qlen: 0,
			running: false,
		},
		{
			name: "t6_cached",
			fn: func(qr *QueuedRunner[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 50 * time.Millisecond,
			gen: 1,
			qlen: 0,
			running: false,
		},
		{
			name: "t7_cached_grace",
			fn: func(qr *QueuedRunner[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 250 * time.Millisecond,
			gen: 2,
			qlen: 0,
			running: true,
		},
		{
			name: "t8_cached",
			fn: func(qr *QueuedRunner[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 250 * time.Millisecond,
			gen: 2,
			qlen: 0,
			running: false,
		},
		{
			name: "t9",
			fn: func(_ *QueuedRunner[string]) {
			},
			sleep: 0,
			gen: 2,
			qlen: 0,
		},
	}
	q := NewCachedQueuedRunner(func() (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "foo", nil
	}, 100 * time.Millisecond, 100 * time.Millisecond)
	for _, tt := range tests {
		go tt.fn(q)
		runtime.Gosched()
		time.Sleep(5 * time.Millisecond) // Sleep to let goroutine run
		q.mu.Lock()
		if tt.gen != q.gen {
			t.Errorf("[%v] Expected generation=%v but received generation=%v", tt.name, tt.gen, q.gen)
		}
		if tt.qlen != len(q.l) {
			t.Errorf("[%v] Expected qlen=%v but received qlen=%v", tt.name, tt.qlen, len(q.l))
		}
		q.mu.Unlock()
		if q.IsRunning() != tt.running {
			t.Errorf("[%v] Expected runner to be running=%v", tt.name, tt.running)
		}
		if tt.sleep > 0 {
			time.Sleep(tt.sleep)
		}
	}
	if numResults := finished.Load(); numResults != uint64(len(tests)-1) {
		t.Errorf("Expected results in tests=%d, but received results=%d", len(tests)-1, numResults)
	}
}
