package engineering

import (
	"fmt"
	"image"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// benchIterations is how many times each update mode redraws the test area.
const benchIterations = 6

// widgetPad is the padding of a page.
func widgetPad() geom.Insets { return geom.XY(12, 24) }

func geomRect(x, y, w, h float64) geom.Rect { return geom.R(x, y, w, h) }

// SamplePatterns are the pictures the update modes are compared on by eye: a smooth ramp, color
// bars, a fine checker and the grid.
func SamplePatterns() []Pattern {
	return []Pattern{{"gray256", gray256}, {"bars", bars}, {"checker8", checker(8)}, {"grid", grid}}
}

// benchmark is a run of the update benchmark: each mode in turn redraws the same two pictures,
// alternately, over half the screen, and the time of every call is kept.
type benchmark struct {
	mode  Mode
	iter  int
	draws []time.Duration // the call that draws and asks for the refresh
	total []time.Duration // the call and the wait until the panel is done
	imgs  [2]*image.RGBA
	rect  image.Rectangle
}

func (s *Screen) showUpdate() {
	intro := widget.NewLabel(s.t("update.intro"))
	intro.Style.Color = widget.Muted
	s.note = widget.NewLabel("")
	s.note.ID = "update-note"
	s.note.Style.FontSize = widget.FontSmall
	s.table = widget.NewTable(Modes+1,
		widget.TableColumn{Title: s.t("update.col.mode"), Flex: 2},
		widget.TableColumn{Title: s.t("update.col.mean"), Flex: 2},
		widget.TableColumn{Title: s.t("update.col.total"), Flex: 2},
		widget.TableColumn{Title: s.t("update.col.p95"), Flex: 2})
	s.table.ID = "update-table"
	run := s.button("update-run", s.t("update.run"), s.runBench)
	samples := s.button("update-samples", s.t("update.samples"), func() { s.showPatterns("sample", SamplePatterns()) })
	s.show(pageUpdate, s.t("eng.update"), intro, run, samples, s.table, s.note)
}

// runBench starts the benchmark: one step per tick so the screen keeps answering touches.
func (s *Screen) runBench() {
	if s.cfg.Panel == nil {
		s.note.Text = s.t("update.nopanel")
		s.doc.Invalidate()
		return
	}
	if s.bench != nil {
		return
	}
	w, h := s.cfg.Width, s.cfg.Height
	// The pictures are as big as the screen, as the panel's rectangles are in screen coordinates.
	s.bench = &benchmark{rect: image.Rect(0, h/2, w, h)}
	s.bench.imgs = [2]*image.RGBA{gray256(w, h), checker(8)(w, h)}
	s.table.Rows = nil
	s.note.Text = s.t("update.running")
	s.cfg.Rec.Event("bench_start", "iterations", benchIterations)
	s.doc.Invalidate()
	s.cfg.After(100*time.Millisecond, s.benchStep)
}

// benchStep makes one redraw, or closes the mode that has had all of its own.
func (s *Screen) benchStep() {
	b := s.bench
	if b == nil {
		return
	}
	p := s.cfg.Panel
	if b.iter == benchIterations {
		s.benchMode(b, p)
		if b.mode++; int(b.mode) == Modes {
			s.benchDone()
			return
		}
		b.iter, b.draws, b.total = 0, nil, nil
	}
	t0 := s.now()
	p.DrawColor(b.imgs[b.iter%2], b.rect, b.mode)
	d := s.now().Sub(t0)
	p.Wait()
	total := s.now().Sub(t0)
	b.draws, b.total = append(b.draws, d), append(b.total, total)
	s.cfg.Rec.Event("bench", "mode", b.mode.String(), "i", b.iter, "draw_ms", Millis(d), "total_ms", Millis(total))
	b.iter++
	s.cfg.After(20*time.Millisecond, s.benchStep)
}

// benchMode records the statistics of the mode that just finished and how long the panel took to
// settle after it.
func (s *Screen) benchMode(b *benchmark, p Panel) {
	t0 := s.now()
	p.Settle()
	settle := s.now().Sub(t0)
	st, tot := Summarize(b.draws), Summarize(b.total)
	s.cfg.Rec.Event("bench_mode", "mode", b.mode.String(), "n", st.Count, "mean_ms", Millis(st.Mean),
		"p95_ms", Millis(st.P95), "min_ms", Millis(st.Min), "max_ms", Millis(st.Max),
		"total_mean_ms", Millis(tot.Mean), "total_p95_ms", Millis(tot.P95), "total_max_ms", Millis(tot.Max),
		"settle_ms", Millis(settle))
	s.cfg.Rec.Flush()
	s.table.Rows = append(s.table.Rows, []string{
		b.mode.String(), fmt.Sprintf("%.1f ms", Millis(st.Mean)), fmt.Sprintf("%.1f ms", Millis(tot.Mean)),
		fmt.Sprintf("%.1f ms", Millis(tot.P95)),
	})
}

func (s *Screen) benchDone() {
	s.bench = nil
	s.note.Text = s.t("update.done", s.cfg.LogPath)
	s.cfg.Rec.Event("bench_end")
	s.cfg.Rec.Flush()
	s.doc.ResetFrame()
	s.doc.Invalidate()
}
