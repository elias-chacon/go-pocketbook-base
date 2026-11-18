package widget

import (
	"fmt"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// Locale holds the names a Calendar shows.
type Locale struct {
	Months       [12]string
	Weekdays     [7]string // Sunday first
	FirstWeekday time.Weekday
}

// LocaleEN is English, weeks starting on Sunday.
var LocaleEN = Locale{
	Months:   [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	Weekdays: [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"},
}

// LocalePTBR is Brazilian Portuguese, weeks starting on Sunday.
var LocalePTBR = Locale{
	Months:   [12]string{"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho", "Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro"},
	Weekdays: [7]string{"Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"},
}

// Calendar shows one month and lets the user pick a day.
type Calendar struct {
	Base
	Locale   Locale
	Year     int
	Month    time.Month
	Selected time.Time // zero when nothing is selected
	Today    time.Time // highlighted with a ring; defaults to now
	OnChange Handler
}

const (
	calHeaderH  = 88.0
	calWeekdayH = 56.0
	calCellH    = 84.0
	calMaxCellW = 130.0
	calNavBtn   = 72.0
)

// NewCalendar creates a calendar showing the month of selected (or today).
func NewCalendar(selected time.Time) *Calendar {
	c := &Calendar{Locale: LocaleEN, Today: time.Now(), Selected: selected}
	c.Init(c)
	ref := selected
	if ref.IsZero() {
		ref = c.Today
	}
	c.Year, c.Month = ref.Year(), ref.Month()
	s := &c.Style
	s.BorderColor, s.BorderWidth, s.BorderRadius, s.Background = Ink, BorderThin, Radius+4, Paper
	s.Padding = geom.XY(18, 18)
	return c
}

// monthGrid returns the 42 days (6 weeks) shown for a month.
func monthGrid(year int, month time.Month, first time.Weekday) [42]time.Time {
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := (int(start.Weekday()) - int(first) + 7) % 7
	start = start.AddDate(0, 0, -offset)
	var out [42]time.Time
	for i := range out {
		out[i] = start.AddDate(0, 0, i)
	}
	return out
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Measure implements Widget.
func (c *Calendar) Measure(avail float64) geom.Size {
	return geom.Size{W: geom.Min(avail, 7*calMaxCellW), H: calHeaderH + calWeekdayH + 6*calCellH}
}

func (c *Calendar) navRects() (prev, next geom.Rect) {
	r := c.ContentRect()
	y := r.Y + (calHeaderH-calNavBtn)/2
	return geom.R(r.X, y, calNavBtn, calNavBtn), geom.R(r.Right()-calNavBtn, y, calNavBtn, calNavBtn)
}

// cellRect is the rectangle of day cell i (0..41).
func (c *Calendar) cellRect(i int) geom.Rect {
	r := c.ContentRect()
	w := r.W / 7
	return geom.R(r.X+float64(i%7)*w, r.Y+calHeaderH+calWeekdayH+float64(i/7)*calCellH, w, calCellH)
}

// ShiftMonth moves the shown month by delta months.
func (c *Calendar) ShiftMonth(delta int) {
	t := time.Date(c.Year, c.Month+time.Month(delta), 1, 0, 0, 0, 0, time.UTC)
	c.Year, c.Month = t.Year(), t.Month()
}

// Paint implements Widget.
func (c *Calendar) Paint(cv *gfx.Canvas) {
	r := c.ContentRect()
	prev, next := c.navRects()
	for _, b := range []struct {
		rc   geom.Rect
		icon gfx.IconKind
	}{{prev, gfx.IconChevronLeft}, {next, gfx.IconChevronRight}} {
		cv.StrokeRoundRect(b.rc, 14, BorderThin, Ink)
		cv.Icon(b.icon, b.rc.Inset(geom.All(18)), iconStroke, Ink)
	}
	title := fmt.Sprintf("%s %d", c.Locale.Months[c.Month-1], c.Year)
	cv.Text(title, geom.R(r.X+calNavBtn, r.Y, r.W-2*calNavBtn, calHeaderH), gfx.TextStyle{Size: FontBody + 2, Bold: true, Color: Ink, Align: gfx.AlignCenter})

	w := r.W / 7
	for i := 0; i < 7; i++ {
		d := (int(c.Locale.FirstWeekday) + i) % 7
		cv.Text(c.Locale.Weekdays[d], geom.R(r.X+float64(i)*w, r.Y+calHeaderH, w, calWeekdayH), gfx.TextStyle{Size: FontSmall, Bold: true, Color: Muted, Align: gfx.AlignCenter})
	}
	cv.Line(geom.Point{X: r.X, Y: r.Y + calHeaderH + calWeekdayH}, geom.Point{X: r.Right(), Y: r.Y + calHeaderH + calWeekdayH}, 2, Mid)

	for i, day := range monthGrid(c.Year, c.Month, c.Locale.FirstWeekday) {
		c.paintDay(cv, c.cellRect(i), day)
	}
}

func (c *Calendar) paintDay(cv *gfx.Canvas, cell geom.Rect, day time.Time) {
	color := Ink
	if day.Month() != c.Month {
		color = Disabled
	}
	ctr := cell.Center()
	radius := geom.Min(cell.W, cell.H)/2 - 6
	if !c.Selected.IsZero() && sameDay(day, c.Selected) {
		cv.FillCircle(ctr, radius, Ink)
		color = Paper
	} else if sameDay(day, c.Today) {
		cv.StrokeCircle(ctr, radius, BorderThin, Ink)
	}
	cv.Text(fmt.Sprintf("%d", day.Day()), cell, gfx.TextStyle{Size: FontBody, Bold: sameDay(day, c.Today), Color: color, Align: gfx.AlignCenter})
}

// HandlePointer implements PointerHandler.
func (c *Calendar) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	p := geom.Point{X: e.X, Y: e.Y}
	prev, next := c.navRects()
	switch {
	case prev.Contains(p):
		c.ShiftMonth(-1)
	case next.Contains(p):
		c.ShiftMonth(1)
	default:
		for i, day := range monthGrid(c.Year, c.Month, c.Locale.FirstWeekday) {
			if c.cellRect(i).Contains(p) {
				c.Selected = day
				c.Year, c.Month = day.Year(), day.Month()
				Fire(c.OnChange, c)
				return
			}
		}
	}
}
