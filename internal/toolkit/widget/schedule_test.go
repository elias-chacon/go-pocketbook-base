package widget

import (
	"bytes"
	"testing"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestMonthViewSelectsOpensAndTellsWhenTheMonthChanges(t *testing.T) {
	mv := NewMonthView(day(2026, time.November, 13))
	changes, opens, months := &counter{}, &counter{}, &counter{}
	mv.OnChange, mv.OnOpen, mv.OnMonth = changes.handler(), opens.handler(), months.handler()
	d := testDoc(mv)

	tapAt(d, mv.cellRect(10).Center()) // 11 November
	if mv.Selected.Day() != 11 || changes.n != 1 || opens.n != 0 {
		t.Errorf("select: %v changes=%d opens=%d", mv.Selected, changes.n, opens.n)
	}
	tapAt(d, mv.cellRect(10).Center()) // the same day again opens it
	if opens.n != 1 || changes.n != 1 {
		t.Errorf("open: changes=%d opens=%d", changes.n, opens.n)
	}
	tapAt(d, mv.cellRect(40).Center()) // a day of December: the month follows
	if mv.Month != time.December || months.n != 1 || mv.Selected.Month() != time.December {
		t.Errorf("month = %v months=%d selected=%v", mv.Month, months.n, mv.Selected)
	}
	mv.ShiftMonth(-1)
	if mv.Month != time.November {
		t.Errorf("ShiftMonth: %v", mv.Month)
	}
}

func TestMonthViewListsTheEntriesOfEachDayAndCountsTheRest(t *testing.T) {
	mv := NewMonthView(day(2026, time.November, 13))
	asked := map[int]int{}
	mv.Entries = func(d time.Time) []string {
		asked[d.Day()]++
		if d.Day() == 20 && d.Month() == time.November {
			return []string{"a", "b", "c", "d", "e"}
		}
		return nil
	}
	d := testDoc(mv)
	with := append([]byte(nil), d.Render().Pix...)
	if asked[20] == 0 {
		t.Error("Entries was not asked about the visible days")
	}
	mv.Entries = nil
	without := append([]byte(nil), d.Render().Pix...)
	if bytes.Equal(with, without) {
		t.Error("the entries are not drawn")
	}
	if h := mv.Measure(1404).H; h != monthHeadH+6*monthCellH {
		t.Errorf("height = %v", h)
	}
}

func gridDays() []time.Time {
	return []time.Time{day(2026, time.May, 18), day(2026, time.May, 19), day(2026, time.May, 20)}
}

func at(d int, h, m int) time.Time { return time.Date(2026, time.May, d, h, m, 0, 0, time.UTC) }

func TestTimeGridPlacesEventsByTimeAndSharesTheWidthWhenTheyOverlap(t *testing.T) {
	g := NewTimeGrid(gridDays()...)
	g.Events = []GridEvent{
		{ID: "a", Title: "A", Start: at(19, 9, 0), End: at(19, 10, 0)},
		{ID: "b", Title: "B", Start: at(19, 9, 30), End: at(19, 11, 0)},
		{ID: "c", Title: "C", Start: at(19, 12, 0), End: at(19, 13, 0)},
	}
	testDoc(g)
	col := g.colRect(1)
	list := g.placedOn(g.Days[1])
	if len(list) != 3 {
		t.Fatalf("placed = %d", len(list))
	}
	if list[0].lanes != 2 || list[1].lanes != 2 || list[0].lane == list[1].lane {
		t.Errorf("overlapping events must share the width: %+v %+v", list[0], list[1])
	}
	if list[2].lanes != 1 {
		t.Errorf("a lone event takes the whole column: %+v", list[2])
	}
	a, c := list[0].rect(g, col), list[2].rect(g, col)
	if c.Y <= a.Y || c.W <= a.W {
		t.Errorf("c must be lower and wider than a: %v %v", a, c)
	}
	if h := a.H; h < gridHourH-1 || h > gridHourH+1 {
		t.Errorf("one hour is %v px high", h)
	}
}

func TestTimeGridTouchReachesEventsSlotsAndDays(t *testing.T) {
	g := NewTimeGrid(gridDays()...)
	g.Events = []GridEvent{
		{ID: "dentist", Title: "Dentista", Start: at(19, 9, 0), End: at(19, 10, 0)},
		{ID: "holiday", Title: "Feriado", Start: day(2026, time.May, 20), End: day(2026, time.May, 21), AllDay: true},
	}
	var event string
	var slot, headDay time.Time
	g.OnEvent = func(id string) { event = id }
	g.OnSlot = func(s time.Time) { slot = s }
	g.OnDay = func(d time.Time) { headDay = d }
	d := testDoc(g)

	list := g.placedOn(g.Days[1])
	tapAt(d, list[0].rect(g, g.colRect(1)).Center())
	if event != "dentist" {
		t.Errorf("event = %q", event)
	}
	// an empty time: 14:20 on the 18th is the 14:00 slot
	col := g.colRect(0)
	tapAt(d, geom.Point{X: col.X + 40, Y: g.minuteY(14*60 + 20)})
	if !slot.Equal(at(18, 14, 0)) {
		t.Errorf("slot = %v", slot)
	}
	tapAt(d, geom.Point{X: g.colRect(2).X + 40, Y: g.ContentRect().Y + 20})
	if !sameDay(headDay, g.Days[2]) {
		t.Errorf("head day = %v", headDay)
	}
	event = ""
	tapAt(d, geom.Point{X: g.colRect(2).X + 40, Y: g.ContentRect().Y + gridHeadH + 10})
	if event != "holiday" {
		t.Errorf("all-day event = %q", event)
	}
}

func TestTimeGridClipsToTheHoursAndGivesInstantsABlock(t *testing.T) {
	g := NewTimeGrid(gridDays()[:1]...)
	g.FirstHour, g.LastHour = 8, 12
	g.Events = []GridEvent{
		{ID: "early", Start: at(18, 6, 0), End: at(18, 7, 0)},  // before the hours: not drawn
		{ID: "late", Start: at(18, 11, 0), End: at(18, 15, 0)}, // clipped to 12:00
		{ID: "instant", Start: at(18, 9, 0), End: at(18, 9, 0)},
		{ID: "other-day", Start: at(19, 9, 0), End: at(19, 10, 0)},
	}
	list := g.placedOn(g.Days[0])
	if len(list) != 2 {
		t.Fatalf("placed = %d", len(list))
	}
	if list[1].ev.ID != "late" || list[1].to != 12*60 {
		t.Errorf("late = %+v", list[1])
	}
	if list[0].to-list[0].from < gridSlot {
		t.Errorf("an instant needs a block to tap: %+v", list[0])
	}
	if h := g.Measure(1404).H; h != gridHeadH+4*gridHourH {
		t.Errorf("height = %v", h)
	}
	g.FirstHour, g.LastHour = 0, 0
	if f, l := g.hours(); f != 7 || l != 21 {
		t.Errorf("default hours = %d..%d", f, l)
	}
	g.FirstHour, g.LastHour = 10, 10
	if f, l := g.hours(); l <= f {
		t.Errorf("hours = %d..%d", f, l)
	}
}

func TestTimeGridDrawsAndMarksNow(t *testing.T) {
	g := NewTimeGrid(gridDays()...)
	g.Today = day(2026, time.May, 19)
	g.Events = []GridEvent{
		{ID: "a", Title: "Reunião", Detail: "09:00", Start: at(19, 9, 0), End: at(19, 11, 0)},
		{ID: "h", Title: "Feriado", Start: day(2026, time.May, 19), End: day(2026, time.May, 20), AllDay: true},
	}
	d := testDoc(g)
	plain := append([]byte(nil), d.Render().Pix...)
	g.Now = func() time.Time { return at(19, 10, 0) }
	d.Invalidate()
	withNow := append([]byte(nil), d.Render().Pix...)
	if bytes.Equal(plain, withNow) {
		t.Error("the current time is not marked")
	}
	if rows := g.allDayRows(); rows != 1 {
		t.Errorf("all-day rows = %d", rows)
	}
}
