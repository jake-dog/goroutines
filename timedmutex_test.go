package goroutines

import (
	"context"
	"testing"
	"time"
)

const allowedVariance = 100 * time.Millisecond

func TestTimedLock(t *testing.T) {
	ttests := []struct {
		name    string
		op      func(*TimedMutex)
		elapsed time.Duration
	}{
		{
			name: "lock no timeout succeeds",
			op: func(tl *TimedMutex) {
				tl.Lock()
			},
			elapsed: 100 * time.Millisecond,
		},
		{
			name: "lock with no timeout fails",
			op: func(tl *TimedMutex) {
				if tl.TryLock() {
					t.Errorf("Expected a timeout waiting for lock")
				}
			},
			elapsed: 100 * time.Millisecond,
		},
		{
			name: "lock with timeout fails",
			op: func(tl *TimedMutex) {
				if tl.LockTimeout(1 * time.Second) {
					t.Errorf("Expected a timeout waiting for lock")
				}
			},
			elapsed: 1 * time.Second,
		},
		{
			name: "lock with context fails",
			op: func(tl *TimedMutex) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
				if err := tl.LockWithContext(ctx); err == nil {
					t.Errorf("Expected a timeout waiting for lock")
				}
			},
			elapsed: 1 * time.Second,
		},
		{
			name: "unlock immediate",
			op: func(tl *TimedMutex) {
				tl.Unlock()
			},
			elapsed: 100 * time.Millisecond,
		},
		{
			name: "immediate lock succeeds",
			op: func(tl *TimedMutex) {
				if !tl.TryLock() {
					t.Errorf("Expected immediate lock")
				}
			},
			elapsed: 100 * time.Millisecond,
		},
		{
			name: "unlock immediate",
			op: func(tl *TimedMutex) {
				tl.Unlock()
			},
			elapsed: 100 * time.Millisecond,
		},
		{
			name: "lock with timeout succeeds",
			op: func(tl *TimedMutex) {
				if !tl.LockTimeout(1 * time.Second) {
					t.Errorf("Expected lock")
				}
			},
			elapsed: 100 * time.Millisecond,
		},
		{
			name: "unlock immediate",
			op: func(tl *TimedMutex) {
				tl.Unlock()
			},
			elapsed: 100 * time.Millisecond,
		},
		{
			name: "lock with context succeeds",
			op: func(tl *TimedMutex) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
				if err := tl.LockWithContext(ctx); err != nil {
					t.Errorf("Expected immediate lock with context")
				}
			},
			elapsed: 100 * time.Millisecond,
		},
	}
	tl := NewTimedMutex()
	for _, tt := range ttests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			tt.op(tl)
			elapsed := time.Since(start)
			if elapsed > tt.elapsed+allowedVariance || elapsed < tt.elapsed-allowedVariance {
				t.Errorf("Expected elapsed time to be %v (+/- %v) but instead it was %v", tt.elapsed, allowedVariance, elapsed)
			}
		})
	}
}

func TestPanics(t *testing.T) {
	ttests := []struct {
		name string
		fn   func()
	}{
		{
			name: "invalid mutex panic lock",
			fn: func() {
				mu := &TimedMutex{}
				if ok := mu.TryLock(); ok {
					t.Errorf("Unexpected trylock success")
				}
			},
		},
		{
			name: "invalid mutex panic lock with context",
			fn: func() {
				mu := &TimedMutex{}
				if err := mu.LockWithContext(context.Background()); err != nil {
					t.Errorf("Unexpected lock err%v", err)
				}
			},
		},
		{
			name: "invalid mutex panic unlock",
			fn: func() {
				mu := &TimedMutex{}
				mu.Unlock()
			},
		},
		{
			name: "panic unlock of unlocked mutex",
			fn: func() {
				mu := NewVariableTimedMutex(-1)
				mu.Unlock()
			},
		},
	}

	for _, tt := range ttests {
		t.Run(tt.name, func(t *testing.T) {
			func(fni func()) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Panic expected but none occurred")
					}
				}()
				fni()
			}(tt.fn)
		})
	}
}
