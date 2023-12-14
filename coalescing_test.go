package goroutines

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

type mystruct struct {
	biz string
	boz int
}

func TestCoalesce(t *testing.T) {
	finished := new(atomic.Uint64)
	tests := []struct {
		name  string
		fn    func(*Coalescer[*mystruct])
		sleep time.Duration
		gen   int
		qlen  int
	}{
		{
			name: "no run when cancelled",
			fn: func(qr *Coalescer[*mystruct]) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				v, err := qr.RunWithContext(ctx)
				if err != context.Canceled {
					t.Errorf("Expected err=%v received=%v", context.Canceled, err)
				}
				if v != nil {
					t.Errorf("Expected nil received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen:   0,
			qlen:  0,
		},
		{
			name: "simple run",
			fn: func(qr *Coalescer[*mystruct]) {
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
			gen:   1,
			qlen:  1,
		},
		{
			name: "queued simple run",
			fn: func(qr *Coalescer[*mystruct]) {
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
			gen:   1,
			qlen:  2,
		},
		{
			name: "run immediate timeout",
			fn: func(qr *Coalescer[*mystruct]) {
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
			gen:   1,
			qlen:  2,
		},
		{
			name: "run with brief timeout",
			fn: func(qr *Coalescer[*mystruct]) {
				v, err := qr.RunTimeout(5 * time.Millisecond)
				if err != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", err)
				}
				if v != nil {
					t.Errorf("Expected v=nil but received v=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen:   1,
			qlen:  2,
		},
		{
			name: "another queued simple run",
			fn: func(qr *Coalescer[*mystruct]) {
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
			gen:   1,
			qlen:  3,
		},
		{
			name: "gen2 fresh queue run with long timeout",
			fn: func(qr *Coalescer[*mystruct]) {
				v, err := qr.RunTimeout(1 * time.Second)
				if v == nil || v.biz != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen:   2,
			qlen:  1,
		},
		{
			name: "gen2 context timeout",
			fn: func(qr *Coalescer[*mystruct]) {
				ctx, _ := context.WithTimeout(context.Background(), 5*time.Millisecond)
				v, err := qr.RunWithContext(ctx)
				if err != context.DeadlineExceeded {
					t.Errorf("Expected err=%v received=%v", context.DeadlineExceeded, err)
				}
				if v != nil {
					t.Errorf("Expected nil received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen:   2,
			qlen:  1,
		},
		{
			name: "gen2 cancelled context",
			fn: func(qr *Coalescer[*mystruct]) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				v, err := qr.RunWithContext(ctx)
				if err != context.Canceled {
					t.Errorf("Expected err=%v received=%v", context.Canceled, err)
				}
				if v != nil {
					t.Errorf("Expected nil received=%v", v)
				}
				finished.Add(1)
			},
			sleep: 0,
			gen:   2,
			qlen:  1,
		},
		{
			name: "gen2 queue normal run",
			fn: func(qr *Coalescer[*mystruct]) {
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
			gen:   2,
			qlen:  2,
		},
		{
			name: "end",
			fn: func(_ *Coalescer[*mystruct]) {
			},
			sleep: 0,
			gen:   2,
			qlen:  0,
		},
	}
	q := Coalesce(func() (*mystruct, error) {
		time.Sleep(100 * time.Millisecond)
		return &mystruct{"foo", 10}, nil
	})
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
	if numResults := finished.Load(); numResults != uint64(len(tests)-1) {
		t.Errorf("Expected results in tests=%d, but received results=%d", len(tests)-1, numResults)
	}
}

func TestCacheCoalesce(t *testing.T) {
	finished := new(atomic.Uint64)
	tests := []struct {
		name    string
		fn      func(*Coalescer[string])
		sleep   time.Duration
		gen     int
		qlen    int
		running bool
	}{
		{
			name: "t1_uncached",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     1,
			qlen:    1,
			running: true,
		},
		{
			name: "t2_uncached",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     1,
			qlen:    2,
			running: true,
		},
		{
			name: "t3_uncached_immediate",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.RunTimeout(0)
				if err != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", err)
				}
				if v != "" {
					t.Errorf("Expected v=nil but received v=%v", v)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     1,
			qlen:    2,
			running: true,
		},
		{
			name: "t4_context_timeout",
			fn: func(qr *Coalescer[string]) {
				ctx, _ := context.WithTimeout(context.Background(), 1*time.Millisecond)
				v, err := qr.RunWithContext(ctx)
				if err != context.DeadlineExceeded {
					t.Errorf("Expected err=%v received=%v", context.DeadlineExceeded, err)
				}
				if v != "" {
					t.Errorf("Expected empty received=%v", v)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     1,
			qlen:    2,
			running: true,
		},
		{
			name: "t5_cancelled_context",
			fn: func(qr *Coalescer[string]) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				v, err := qr.RunWithContext(ctx)
				if err != context.Canceled {
					t.Errorf("Expected err=%v received=%v", context.Canceled, err)
				}
				if v != "" {
					t.Errorf("Expected empty received=%v", v)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     1,
			qlen:    2,
			running: true,
		},
		{
			name: "t6_uncached_nearimmediate",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.RunTimeout(20 * time.Millisecond)
				if err != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", err)
				}
				if v != "" {
					t.Errorf("Expected v=nil but received v=%v", v)
				}
				finished.Add(1)
			},
			sleep:   250 * time.Millisecond,
			gen:     1,
			qlen:    3,
			running: true,
		},
		{
			name: "t7_cached",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.TryRun()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     1,
			qlen:    0,
			running: false,
		},
		{
			name: "t8_cached",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep:   50 * time.Millisecond,
			gen:     1,
			qlen:    0,
			running: false,
		},
		{
			name: "t9_cached_grace_but_cancelled",
			fn: func(qr *Coalescer[string]) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				v, err := qr.RunWithContext(ctx)
				if err != context.Canceled {
					t.Errorf("Expected err=%v received=%v", context.Canceled, err)
				}
				if v != "" {
					t.Errorf("Expected nil received=%v", v)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     1,
			qlen:    0,
			running: false,
		},
		{
			name: "t9_cached_grace",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     2,
			qlen:    0,
			running: true,
		},
		{
			name: "t9_cached_grace_running",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep:   0,
			gen:     2,
			qlen:    0,
			running: true,
		},
		{
			name: "t10_flushed",
			fn: func(qr *Coalescer[string]) {
				qr.Flush()
				v, err := qr.TryRun()
				if err != ErrRunnerTimedout {
					t.Errorf("Expected ErrRunnerTimedout received=%v", err)
				}
				if v != "" {
					t.Errorf("Expected v=nil but received v=%v", v)
				}
				finished.Add(1)
			},
			sleep:   250 * time.Millisecond,
			gen:     2,
			qlen:    0,
			running: true,
		},
		{
			name: "t11_cached",
			fn: func(qr *Coalescer[string]) {
				v, err := qr.Run()
				if v != "foo" {
					t.Errorf("Expected foo received=%v", v)
				}
				if err != nil {
					t.Error(err)
				}
				finished.Add(1)
			},
			sleep:   250 * time.Millisecond,
			gen:     2,
			qlen:    0,
			running: false,
		},
		{
			name: "t12",
			fn: func(_ *Coalescer[string]) {
			},
			sleep: 0,
			gen:   2,
			qlen:  0,
		},
	}
	q := CacheCoalesce(func() (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "foo", nil
	}, 100*time.Millisecond, 100*time.Millisecond)
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
	if numResults := finished.Load(); numResults != uint64(len(tests)-1) {
		t.Errorf("Expected results in tests=%d, but received results=%d", len(tests)-1, numResults)
	}
}

func TestAbort(t *testing.T) {
	tests := []struct {
		name      string
		e         chan *F[string]
		gen       int
		cfn       func(chan *F[string]) *Coalescer[string]
		expectLen int
	}{
		{
			name: "not the last",
			e:    make(chan *F[string], 1),
			gen:  2,
			cfn: func(e chan *F[string]) *Coalescer[string] {
				return &Coalescer[string]{
					l: []chan *F[string]{
						make(chan *F[string], 1),
						make(chan *F[string], 1),
						e,
						make(chan *F[string], 1),
					},
					gen: 2,
				}
			},
			expectLen: 3,
		},
		{
			name: "the first",
			e:    make(chan *F[string], 1),
			gen:  2,
			cfn: func(e chan *F[string]) *Coalescer[string] {
				return &Coalescer[string]{
					l: []chan *F[string]{
						e,
						make(chan *F[string], 1),
						make(chan *F[string], 1),
						make(chan *F[string], 1),
					},
					gen: 2,
				}
			},
			expectLen: 3,
		},
		{
			name: "the last",
			e:    make(chan *F[string], 1),
			gen:  2,
			cfn: func(e chan *F[string]) *Coalescer[string] {
				return &Coalescer[string]{
					l: []chan *F[string]{
						make(chan *F[string], 1),
						make(chan *F[string], 1),
						make(chan *F[string], 1),
						e,
					},
					gen: 2,
				}
			},
			expectLen: 3,
		},
		{
			name: "wrong gen",
			e:    make(chan *F[string], 1),
			gen:  2,
			cfn: func(e chan *F[string]) *Coalescer[string] {
				return &Coalescer[string]{
					l: []chan *F[string]{
						e,
						make(chan *F[string], 1),
						make(chan *F[string], 1),
						make(chan *F[string], 1),
					},
					gen: 3,
				}
			},
			expectLen: 4,
		},
		{
			name: "only",
			e:    make(chan *F[string], 1),
			gen:  2,
			cfn: func(e chan *F[string]) *Coalescer[string] {
				return &Coalescer[string]{
					l: []chan *F[string]{
						e,
					},
					gen: 2,
				}
			},
			expectLen: 0,
		},
		{
			name: "empty",
			e:    make(chan *F[string], 1),
			gen:  2,
			cfn: func(_ chan *F[string]) *Coalescer[string] {
				return &Coalescer[string]{
					gen: 3,
				}
			},
			expectLen: 0,
		},
		{
			name: "empty same gen",
			e:    make(chan *F[string], 1),
			gen:  2,
			cfn: func(_ chan *F[string]) *Coalescer[string] {
				return &Coalescer[string]{
					gen: 2,
				}
			},
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			c := tt.cfn(tt.e)
			c.abort(tt.gen, tt.e)
			if len(c.l) != tt.expectLen {
				t.Errorf("[%v] expected qlen=%v but received qlen=%v", tt.name, tt.expectLen, len(c.l))
			}
		})
	}
}
