package goroutines

import (
	"context"
	"time"
)

var s = struct{}{}

// TimedMutex implements mutex-like interface but adds lock timeouts.
// The zero value cannot be used.
type TimedMutex struct {
	c chan struct{}
}

// NewVariableTimedMutex returns a new TimedMutex.
// Limit determines how many consumers can obtain the mutex at once.
func NewVariableTimedMutex(limit int) *TimedMutex {
	p := limit
	if p <= 0 {
		p = 1
	}
	l := &TimedMutex{
		c: make(chan struct{}, p),
	}
	for i := 0; i < p; i++ {
		l.c <- s
	}
	return l
}

// NewTimedMutex returns a TimedMutex similar to sync.Mutex.
func NewTimedMutex() *TimedMutex {
	return NewVariableTimedMutex(1)
}

func (l *TimedMutex) internalLock(t time.Duration) bool {
	if l.c == nil {
		panic("Uninitialized TimedMutex")
	}
	if t < 0 {
		<-l.c
		return true
	}
	if t == 0 {
		select {
		case <-l.c:
			return true
		default:
			return false
		}
	}
	timer := time.NewTimer(t)
	select {
	case <-l.c:
		return true
	case <-timer.C:
	}
	return false
}

// LockTimeout returns true if the lock succeeded before timeout.
func (l *TimedMutex) LockTimeout(timeout time.Duration) bool {
	return l.internalLock(timeout)
}

// LockTimeout returns an error if context is cancelled before lock succeeds.
func (l *TimedMutex) LockWithContext(ctx context.Context) error {
	if l.c == nil {
		panic("Uninitialized TimedMutex")
	}
	select {
	case <-l.c:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Lock locks the mutex.
func (l *TimedMutex) Lock() {
	l.internalLock(-1)
}

// TryLock tries to lock and reports whether it succeeded.
func (l *TimedMutex) TryLock() bool {
	return l.internalLock(0)
}

// Unlock unlocks the mutex.
func (l *TimedMutex) Unlock() {
	if l.c == nil {
		panic("Uninitialized TimedMutex")
	}
	select {
	case l.c <- s:
	default:
		panic("TimedMutex unlock of unlocked mutex")
	}
}
