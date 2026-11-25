package domain

import "sync"

// Progress is how far a job in the background got, to be read by the screen on the event loop while
// another goroutine advances it. It counts steps (the release, the catalog, one per icon), not
// bytes: what is small and quick is not worth a byte count. Every method is safe on a nil *Progress,
// so a caller that does not care passes none.
type Progress struct {
	mu          sync.Mutex
	done, total int
	active      bool
}

// Begin starts a job of total steps.
func (p *Progress) Begin(total int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done, p.total, p.active = 0, total, true
}

// Add makes the job longer by n steps (what it has to do was learned on the way).
func (p *Progress) Add(n int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total += n
}

// Step is one more step done.
func (p *Progress) Step() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done < p.total {
		p.done++
	}
}

// Advance is n more units done (bytes, when the steps are bytes).
func (p *Progress) Advance(n int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done += n; p.done > p.total {
		p.done = p.total
	}
}

// End finishes the job, whatever was left.
func (p *Progress) End() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done, p.active = p.total, false
}

// State is the fraction done (0..1) and whether a job is running.
func (p *Progress) State() (fraction float64, active bool) {
	if p == nil {
		return 0, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.total <= 0 {
		return 0, p.active
	}
	return float64(p.done) / float64(p.total), p.active
}
