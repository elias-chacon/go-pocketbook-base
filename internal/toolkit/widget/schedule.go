package widget

import (
	"fmt"
	"sort"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// MonthView is a month as a calendar of an agenda app: a grid of days where every cell
// lists what is on that day (a few lines of text and "+N more"), like the month view of
// Outlook. Tapping a day selects it; tapping the selected day again opens it (OnOpen).
// The Calendar widget is the small date picker; this one is for reading a schedule.
type MonthView struct {
	Base
	Locale   Locale
	Year     int
	Month    time.Month
	Selected time.Time // zero when nothing is selected
	Today    time.Time // drawn in bold with a ring; defaults to now
	// Entries gives the short texts of a day (the titles of its events), in the order
	// to show them. Nil lists nothing.
	Entries  func(day time.Time) []string
	OnChange Handler // a day was selected
	OnOpen   Handler // the selected day was tapped again
	OnMonth  Handler // the month changed (ShiftMonth by the caller, or a tap on another month's day)
}

const (
	monthHeadH   = 56.0
	monthCellH   = 188.0
	monthLineH   = 34.0
	monthMaxRows = 3
)

// NewMonthView creates a month view showing the month of selected (or today).
func NewMonthView(selected time.Time) *MonthView {
	m := &MonthView{Locale: LocaleEN, Today: time.Now(), Selected: selected}
	m.Init(m)
	ref := selected
	if ref.IsZero() {
		ref = m.Today
	}
	m.Year, m.Month = ref.Year(), ref.Month()
	m.Style.Background = Paper
	return m
}

// ShiftMonth moves the shown month by delta months.
func (m *MonthView) ShiftMonth(delta int) {
	t := time.Date(m.Year, m.Month+time.Month(delta), 1, 0, 0, 0, 0, time.UTC)
	m.Year, m.Month = t.Year(), t.Month()
}

// Measure implements Widget.
func (m *MonthView) Measure(avail float64) geom.Size {
	return geom.Size{W: avail, H: monthHeadH + 6*monthCellH}
}

func (m *MonthView) cellRect(i int) geom.Rect {
	r := m.ContentRect()
	w := r.W / 7
	return geom.R(r.X+float64(i%7)*w, r.Y+monthHeadH+float64(i/7)*monthCellH, w, monthCellH)
}

// Paint implements Widget.
func (m *MonthView) Paint(c *gfx.Canvas) {
	r := m.ContentRect()
	w := r.W / 7
	for i := 0; i < 7; i++ {
		d := (int(m.Locale.FirstWeekday) + i) % 7
		c.Text(m.Locale.Weekdays[d], geom.R(r.X+float64(i)*w, r.Y, w, monthHeadH), gfx.TextStyle{Size: FontSmall, Bold: true, Color: Muted, Align: gfx.AlignCenter})
	}
	for i, day := range monthGrid(m.Year, m.Month, m.Locale.FirstWeekday) {
		m.paintCell(c, m.cellRect(i), day)
	}
}

func (m *MonthView) paintCell(c *gfx.Canvas, cell geom.Rect, day time.Time) {
	c.StrokeRoundRect(cell, 0, 1, Mid)
	inMonth := day.Month() == m.Month
	color := Ink
	if !inMonth {
		color = Disabled
	}
	num := geom.R(cell.X+8, cell.Y+4, 56, monthLineH)
	switch {
	case !m.Selected.IsZero() && sameDay(day, m.Selected):
		c.FillRoundRect(geom.R(num.X-4, num.Y, 52, monthLineH), 10, Ink)
		color = Paper
	case sameDay(day, m.Today):
		c.StrokeRoundRect(geom.R(num.X-4, num.Y, 52, monthLineH), 10, BorderThin, Ink)
	}
	c.Text(fmt.Sprintf("%d", day.Day()), geom.R(num.X, num.Y, 44, monthLineH), gfx.TextStyle{Size: FontSmall, Bold: sameDay(day, m.Today), Color: color, Align: gfx.AlignCenter})
	if m.Entries == nil || !inMonth {
		return
	}
	entries := m.Entries(day)
	for i, text := range entries {
		y := cell.Y + 4 + monthLineH + float64(i)*monthLineH
		if i == monthMaxRows {
			c.Text(fmt.Sprintf("+%d", len(entries)-monthMaxRows), geom.R(cell.X+8, y, cell.W-16, monthLineH), gfx.TextStyle{Size: FontSmall - 2, Bold: true, Color: Muted})
			break
		}
		c.FillRoundRect(geom.R(cell.X+6, y+2, cell.W-12, monthLineH-4), 6, Soft)
		c.Text(text, geom.R(cell.X+12, y, cell.W-24, monthLineH), gfx.TextStyle{Size: FontSmall - 2, Color: Ink, Ellipsis: true})
	}
}

// HandlePointer implements PointerHandler.
func (m *MonthView) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	p := geom.Point{X: e.X, Y: e.Y}
	for i, day := range monthGrid(m.Year, m.Month, m.Locale.FirstWeekday) {
		if !m.cellRect(i).Contains(p) {
			continue
		}
		again := !m.Selected.IsZero() && sameDay(day, m.Selected)
		m.Selected = day
		if day.Month() != m.Month {
			m.Year, m.Month = day.Year(), day.Month()
			Fire(m.OnMonth, m)
		}
		if again {
			Fire(m.OnOpen, m)
			return
		}
		Fire(m.OnChange, m)
		return
	}
}

// GridEvent is one block of a TimeGrid. Start and End are in the same time zone as the
// grid's Days, whose wall clock is what is drawn.
type GridEvent struct {
	ID, Title, Detail string
	Start, End        time.Time
	AllDay            bool
}

// TimeGrid is the day, work week and week views of an agenda app: one column per day
// with the hours down the side and the events as blocks placed by time (events that
// overlap share the width), and a strip for all-day events on top.
type TimeGrid struct {
	Base
	Locale Locale
	// Days are the columns, as dates (their time of day is ignored).
	Days []time.Time
	// FirstHour and LastHour bound the hours drawn (7 and 21 when zero): an event
	// outside them is clipped to them.
	FirstHour, LastHour int
	Events              []GridEvent
	Today               time.Time
	OnEvent             func(id string)     // an event block was tapped
	OnSlot              func(at time.Time)  // an empty time was tapped (rounded down to 30 minutes)
	OnDay               func(day time.Time) // a day header was tapped
	Now                 func() time.Time    // marks the current time with a line (nil: no line)
}

const (
	gridGutter   = 96.0
	gridHeadH    = 84.0
	gridHourH    = 84.0
	gridAllDayH  = 40.0
	gridMaxAllDy = 3
	gridSlot     = 30
)

// NewTimeGrid creates a grid for some days.
func NewTimeGrid(days ...time.Time) *TimeGrid {
	g := &TimeGrid{Locale: LocaleEN, Days: days, Today: time.Now()}
	g.Init(g)
	g.Style.Background = Paper
	return g
}

func (g *TimeGrid) hours() (int, int) {
	first, last := g.FirstHour, g.LastHour
	if first == 0 && last == 0 {
		first, last = 7, 21
	}
	if last <= first {
		last = first + 1
	}
	return first, last
}

func (g *TimeGrid) allDayRows() int {
	most := 0
	for _, d := range g.Days {
		n := 0
		for _, e := range g.Events {
			if e.AllDay && coversDay(e, d) {
				n++
			}
		}
		if n > most {
			most = n
		}
	}
	if most > gridMaxAllDy {
		most = gridMaxAllDy
	}
	return most
}

// coversDay reports whether an all-day event is on a date (its End is exclusive).
func coversDay(e GridEvent, day time.Time) bool {
	y, m, d := day.Date()
	date := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	sy, sm, sd := e.Start.Date()
	start := time.Date(sy, sm, sd, 0, 0, 0, 0, time.UTC)
	ey, em, ed := e.End.Date()
	end := time.Date(ey, em, ed, 0, 0, 0, 0, time.UTC)
	if !end.After(start) {
		end = start.AddDate(0, 0, 1)
	}
	return !date.Before(start) && date.Before(end)
}

func (g *TimeGrid) topOfGrid() float64 {
	return gridHeadH + float64(g.allDayRows())*gridAllDayH
}

// Measure implements Widget.
func (g *TimeGrid) Measure(avail float64) geom.Size {
	first, last := g.hours()
	return geom.Size{W: avail, H: g.topOfGrid() + float64(last-first)*gridHourH}
}

func (g *TimeGrid) colRect(i int) geom.Rect {
	r := g.ContentRect()
	n := len(g.Days)
	if n == 0 {
		n = 1
	}
	w := (r.W - gridGutter) / float64(n)
	return geom.R(r.X+gridGutter+float64(i)*w, r.Y, w, r.H)
}

// minuteY is the y of a minute of the day.
func (g *TimeGrid) minuteY(minute float64) float64 {
	first, _ := g.hours()
	return g.ContentRect().Y + g.topOfGrid() + (minute/60-float64(first))*gridHourH
}

// timedOn lists the timed events that touch a day, with their clock minutes clipped to
// the hours shown.
type placed struct {
	ev          GridEvent
	from, to    float64
	lane, lanes int
}

func (g *TimeGrid) placedOn(day time.Time) []placed {
	first, last := g.hours()
	lo, hi := float64(first*60), float64(last*60)
	var out []placed
	for _, e := range g.Events {
		if e.AllDay || !sameDay(e.Start, day) {
			continue
		}
		from := float64(e.Start.Hour()*60 + e.Start.Minute())
		to := from + e.End.Sub(e.Start).Minutes()
		if to < from+gridSlot {
			to = from + gridSlot // an instant still needs a block to tap
		}
		if to <= lo || from >= hi {
			continue
		}
		out = append(out, placed{ev: e, from: geom.Max(from, lo), to: geom.Min(to, hi)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].from < out[j].from })
	assignLanes(out)
	return out
}

// assignLanes puts overlapping events side by side: each cluster of events that
// overlap in time shares the width, one lane each.
func assignLanes(list []placed) {
	for start := 0; start < len(list); {
		stop := clusterEnd(list, start)
		assignCluster(list[start:stop])
		start = stop
	}
}

// clusterEnd is the index after the last event of the cluster that starts at start.
func clusterEnd(list []placed, start int) int {
	end := list[start].to
	stop := start + 1
	for stop < len(list) && list[stop].from < end {
		if list[stop].to > end {
			end = list[stop].to
		}
		stop++
	}
	return stop
}

// assignCluster gives each event of a cluster the first lane that is free at its start.
func assignCluster(cluster []placed) {
	var laneEnds []float64
	for i := range cluster {
		lane := freeLane(laneEnds, cluster[i].from)
		if lane == len(laneEnds) {
			laneEnds = append(laneEnds, 0)
		}
		laneEnds[lane] = cluster[i].to
		cluster[i].lane = lane
	}
	for i := range cluster {
		cluster[i].lanes = len(laneEnds)
	}
}

// freeLane is the first lane that ended by from (len(laneEnds) when none did).
func freeLane(laneEnds []float64, from float64) int {
	for l, e := range laneEnds {
		if e <= from {
			return l
		}
	}
	return len(laneEnds)
}

func (p placed) rect(g *TimeGrid, col geom.Rect) geom.Rect {
	w := (col.W - 6) / float64(p.lanes)
	return geom.R(col.X+3+float64(p.lane)*w, g.minuteY(p.from), w-2, g.minuteY(p.to)-g.minuteY(p.from))
}

// Paint implements Widget.
func (g *TimeGrid) Paint(c *gfx.Canvas) {
	r := g.ContentRect()
	first, last := g.hours()
	top := r.Y + g.topOfGrid()
	for h := first; h <= last; h++ {
		y := top + float64(h-first)*gridHourH
		c.Line(geom.Point{X: r.X + gridGutter - 8, Y: y}, geom.Point{X: r.Right(), Y: y}, 1, Mid)
		if h < last {
			c.Text(fmt.Sprintf("%02d:00", h), geom.R(r.X, y+2, gridGutter-14, 30), gfx.TextStyle{Size: FontSmall - 4, Color: Muted, Align: gfx.AlignEnd})
		}
	}
	for i, day := range g.Days {
		col := g.colRect(i)
		c.Line(geom.Point{X: col.X, Y: r.Y}, geom.Point{X: col.X, Y: r.Bottom()}, 1, Mid)
		g.paintHead(c, col, day)
		g.paintAllDay(c, col, day)
		for _, p := range g.placedOn(day) {
			g.paintBlock(c, p, col)
		}
		if g.Now != nil && sameDay(day, g.Now()) {
			now := g.Now()
			m := float64(now.Hour()*60 + now.Minute())
			if m >= float64(first*60) && m <= float64(last*60) {
				y := g.minuteY(m)
				c.Line(geom.Point{X: col.X, Y: y}, geom.Point{X: col.X + col.W, Y: y}, 3, Ink)
			}
		}
	}
}

func (g *TimeGrid) paintHead(c *gfx.Canvas, col geom.Rect, day time.Time) {
	label := fmt.Sprintf("%s %d", g.Locale.Weekdays[day.Weekday()], day.Day())
	today := sameDay(day, g.Today)
	if today {
		c.FillRoundRect(geom.R(col.X+6, col.Y+10, col.W-12, gridHeadH-20), 12, Ink)
	}
	color := Ink
	if today {
		color = Paper
	}
	c.Text(label, geom.R(col.X, col.Y, col.W, gridHeadH), gfx.TextStyle{Size: FontSmall, Bold: true, Color: color, Align: gfx.AlignCenter, Ellipsis: true})
}

func (g *TimeGrid) paintAllDay(c *gfx.Canvas, col geom.Rect, day time.Time) {
	i := 0
	for _, e := range g.Events {
		if !e.AllDay || !coversDay(e, day) {
			continue
		}
		if i >= gridMaxAllDy {
			break
		}
		y := col.Y + gridHeadH + float64(i)*gridAllDayH
		c.FillRoundRect(geom.R(col.X+3, y+2, col.W-6, gridAllDayH-4), 8, Ink)
		c.Text(e.Title, geom.R(col.X+10, y, col.W-20, gridAllDayH), gfx.TextStyle{Size: FontSmall - 4, Color: Paper, Ellipsis: true})
		i++
	}
}

func (g *TimeGrid) paintBlock(c *gfx.Canvas, p placed, col geom.Rect) {
	rc := p.rect(g, col)
	c.FillRoundRect(rc, 8, Soft)
	c.StrokeRoundRect(rc, 8, 2, Ink)
	c.Text(p.ev.Title, geom.R(rc.X+8, rc.Y+4, rc.W-16, 30), gfx.TextStyle{Size: FontSmall - 4, Bold: true, Color: Ink, Ellipsis: true})
	if p.ev.Detail != "" && rc.H >= 64 {
		c.Text(p.ev.Detail, geom.R(rc.X+8, rc.Y+34, rc.W-16, 28), gfx.TextStyle{Size: FontSmall - 6, Color: Muted, Ellipsis: true})
	}
}

// HandlePointer implements PointerHandler.
func (g *TimeGrid) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	pt := geom.Point{X: e.X, Y: e.Y}
	r := g.ContentRect()
	for i, day := range g.Days {
		col := g.colRect(i)
		if pt.X < col.X || pt.X >= col.X+col.W {
			continue
		}
		g.tapColumn(pt, col, day, r)
		return
	}
}

func (g *TimeGrid) tapColumn(pt geom.Point, col geom.Rect, day time.Time, r geom.Rect) {
	switch {
	case pt.Y < r.Y+gridHeadH:
		if g.OnDay != nil {
			g.OnDay(day)
		}
	case pt.Y < r.Y+g.topOfGrid():
		g.tapAllDay(int((pt.Y-r.Y-gridHeadH)/gridAllDayH), day)
	default:
		g.tapHours(pt, col, day, r)
	}
}

// tapAllDay opens the all-day event shown in a row of the strip.
func (g *TimeGrid) tapAllDay(row int, day time.Time) {
	i := 0
	for _, ev := range g.Events {
		if !ev.AllDay || !coversDay(ev, day) {
			continue
		}
		if i == row && g.OnEvent != nil {
			g.OnEvent(ev.ID)
			return
		}
		i++
	}
}

// tapHours opens the event block under the touch, or else asks for a new event at that time.
func (g *TimeGrid) tapHours(pt geom.Point, col geom.Rect, day time.Time, r geom.Rect) {
	list := g.placedOn(day)
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].rect(g, col).Contains(pt) {
			if g.OnEvent != nil {
				g.OnEvent(list[i].ev.ID)
			}
			return
		}
	}
	first, _ := g.hours()
	minutes := float64(first*60) + (pt.Y-r.Y-g.topOfGrid())/gridHourH*60
	slot := int(minutes) / gridSlot * gridSlot
	if g.OnSlot != nil {
		y, mo, d := day.Date()
		g.OnSlot(time.Date(y, mo, d, slot/60, slot%60, 0, 0, day.Location()))
	}
}
