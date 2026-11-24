package engineering

import (
	"fmt"
	"sort"
	"time"
)

// Stats summarizes a series of timings.
type Stats struct {
	Count               int
	Min, Max, Mean, P95 time.Duration
}

// Summarize computes the statistics of the samples (all zero when there are none).
func Summarize(samples []time.Duration) Stats {
	if len(samples) == 0 {
		return Stats{}
	}
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}
	idx := (len(sorted)*95 + 99) / 100 // nearest rank
	if idx < 1 {
		idx = 1
	}
	return Stats{
		Count: len(sorted), Min: sorted[0], Max: sorted[len(sorted)-1],
		Mean: sum / time.Duration(len(sorted)), P95: sorted[idx-1],
	}
}

// PerSecond is how many of the timed operations fit in a second at the mean pace.
func (s Stats) PerSecond() float64 {
	if s.Mean <= 0 {
		return 0
	}
	return float64(time.Second) / float64(s.Mean)
}

// Millis is a duration in milliseconds, with one decimal.
func Millis(d time.Duration) float64 { return float64(d.Microseconds()/100) / 10 }

// String is the summary on one line: n, mean, p95, min, max (milliseconds) and the rate.
func (s Stats) String() string {
	return fmt.Sprintf("n=%d mean=%.1fms p95=%.1fms min=%.1fms max=%.1fms rate=%.1f/s",
		s.Count, Millis(s.Mean), Millis(s.P95), Millis(s.Min), Millis(s.Max), s.PerSecond())
}

// onPaint keeps the last repaint of the platform and, on the pages that scroll or follow a finger,
// writes it down: what a frame cost, step by step, tells where the time goes.
func (s *Screen) onPaint(p Paint) {
	s.paint = p
	if s.page == pageScroll || s.page == pageTouch {
		s.cfg.Rec.Event("paint", append([]interface{}{"page", s.page}, p.fields()...)...)
	}
}

// fields are the numbers of a repaint as fields of a log event.
func (p Paint) fields() []interface{} {
	return []interface{}{
		"layout_ms", Millis(p.Layout), "snapshot_ms", Millis(p.Snapshot), "diff_ms", Millis(p.Diff),
		"paint_ms", Millis(p.Paint), "show_ms", Millis(p.Show),
		"full", p.Full, "slid", p.Slid, "painted_px", p.Painted, "sent_px", p.Sent, "areas", p.Areas,
	}
}
