package inkview

import (
	"sync"
	"time"
)

// timers is the list of things to run later on the event loop. InkView has one timer per name, so
// the app keeps its own list and arms a single timer for the earliest one.
type timers struct {
	mu      sync.Mutex
	pending []timer
}

type timer struct {
	due time.Time
	run func()
}

// add queues f to run d from now and returns how long until the earliest entry is due.
func (t *timers) add(now time.Time, d time.Duration, f func()) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pending = append(t.pending, timer{due: now.Add(d), run: f})
	return t.next(now)
}

// next is how long until the earliest entry is due (-1 when there is none). The caller holds the lock.
func (t *timers) next(now time.Time) time.Duration {
	if len(t.pending) == 0 {
		return -1
	}
	min := t.pending[0].due
	for _, p := range t.pending[1:] {
		if p.due.Before(min) {
			min = p.due
		}
	}
	if d := min.Sub(now); d > 0 {
		return d
	}
	return 0
}

// runDue runs, in the order they were added, the entries that are due and returns how long until
// the next one (-1 when none is left). Entries added while they run wait for the next round.
func (t *timers) runDue(now time.Time) time.Duration {
	t.mu.Lock()
	var due []func()
	left := t.pending[:0:0]
	for _, p := range t.pending {
		if p.due.After(now) {
			left = append(left, p)
		} else {
			due = append(due, p.run)
		}
	}
	t.pending = left
	t.mu.Unlock()
	for _, f := range due {
		f()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.next(time.Now())
}
