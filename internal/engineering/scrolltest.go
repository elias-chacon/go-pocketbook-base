package engineering

import (
	"fmt"
	"time"

	"go-pocketbook-base/internal/toolkit/widget"
)

// scrollRows is how many rows the scroll test has: far more than a screen.
const scrollRows = 300

// Auto-scroll: the number of steps, and how far each one moves, in design pixels.
const (
	autoSteps = 40
	autoStep  = 160
)

// autoScroll is a run of the scroll test: the list moves by itself step by step, and the time of
// each step (the layout, the paint and the call that updates the panel) is kept.
type autoScroll struct {
	n       int
	dir     float64
	samples []time.Duration
	stopped bool
}

func (a *autoScroll) stop() { a.stopped = true }

func (s *Screen) showScroll() {
	rows := make([]widget.Widget, 0, scrollRows)
	for i := 1; i <= scrollRows; i++ {
		row := widget.NewLabel(fmt.Sprintf("%s %03d", s.t("scroll.row"), i))
		row.Style.Padding = widgetPad()
		row.Style.MinHeight = 90
		if i%2 == 0 {
			row.Style.Background = widget.Soft
		}
		if i%10 == 0 {
			row.Style.Bold, row.Style.Background = true, widget.Ink
			row.Style.Color = widget.Paper
		}
		rows = append(rows, row)
	}
	s.scroll = widget.NewScroll(rows...)
	s.scroll.ID = "scroll-list"
	s.scroll.Style.FlexGrow = 1
	last := s.now()
	s.scroll.OnScroll = func(*widget.Event) {
		if s.auto != nil {
			return // the run records its own steps
		}
		now := s.now()
		s.cfg.Rec.Event("scroll", "offset", int(s.scroll.Offset), "gap_ms", Millis(now.Sub(last)))
		last = now
	}
	s.note = widget.NewLabel(s.t("scroll.hint"))
	s.note.ID = "scroll-note"
	s.note.Style.FontSize, s.note.Style.Color = widget.FontSmall, widget.Muted
	var modeBtn *widget.Button
	modeBtn = s.button("scroll-mode", s.t("scroll.mode", s.mode), func() {
		s.mode = s.mode.Next()
		if s.cfg.Panel != nil {
			s.cfg.Panel.SetMode(s.mode)
		}
		modeBtn.Text = s.t("scroll.mode", s.mode)
		s.cfg.Rec.Event("scroll_mode", "mode", s.mode.String())
		s.doc.Invalidate()
	})
	auto := s.button("scroll-auto", s.t("scroll.auto"), s.startAuto)
	s.show(pageScroll, s.t("eng.scroll"), s.scroll, widget.NewRow(12, modeBtn, auto), s.note)
}

// startAuto starts the automatic scroll, one step per tick.
func (s *Screen) startAuto() {
	if s.auto != nil || s.scroll == nil {
		return
	}
	s.auto = &autoScroll{dir: 1}
	s.scroll.ScrollTo(0)
	s.cfg.Rec.Event("scroll_start", "mode", s.mode.String(), "steps", autoSteps, "step_px", autoStep)
	s.cfg.After(100*time.Millisecond, s.autoStep)
}

// autoStep moves the list one step and times what it takes to show it.
func (s *Screen) autoStep() {
	a := s.auto
	if a == nil || a.stopped || s.scroll == nil {
		return
	}
	t0 := s.now()
	if next := s.scroll.Offset + a.dir*autoStep; next < 0 || next > s.scroll.MaxOffset() {
		a.dir = -a.dir
	}
	s.scroll.ScrollBy(a.dir * autoStep)
	if s.cfg.Flush != nil {
		s.cfg.Flush()
	}
	d := s.now().Sub(t0)
	a.samples = append(a.samples, d)
	a.n++
	s.cfg.Rec.Event("scroll_step", "mode", s.mode.String(), "i", a.n, "offset", int(s.scroll.Offset), "ms", Millis(d))
	if a.n >= autoSteps {
		s.autoDone(a)
		return
	}
	s.cfg.After(time.Millisecond, s.autoStep)
}

func (s *Screen) autoDone(a *autoScroll) {
	st := Summarize(a.samples)
	var settle time.Duration
	if s.cfg.Panel != nil {
		t0 := s.now()
		s.cfg.Panel.Settle()
		settle = s.now().Sub(t0)
	}
	s.cfg.Rec.Event("scroll_run", "mode", s.mode.String(), "n", st.Count, "mean_ms", Millis(st.Mean),
		"p95_ms", Millis(st.P95), "min_ms", Millis(st.Min), "max_ms", Millis(st.Max),
		"fps", st.PerSecond(), "settle_ms", Millis(settle))
	s.cfg.Rec.Flush()
	s.auto = nil
	s.note.Text = s.t("scroll.result", s.mode, st.String())
	s.doc.Invalidate()
}
