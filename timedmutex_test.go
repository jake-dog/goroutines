package goroutines

import (
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
			name: "lock with timeout fails",
			op: func(tl *TimedMutex) {
				if tl.LockTimeout(1 * time.Second) {
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
