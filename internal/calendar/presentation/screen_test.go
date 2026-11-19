package presentation

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/calendar/application"
	"go-pocketbook-base/internal/calendar/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type memStore struct {
	cals   []domain.Calendar
	events map[string]domain.Event
	next   int
	err    error
}

func (m *memStore) SaveCalendars(_ string, c []domain.Calendar) error { m.cals = c; return m.err }
func (m *memStore) Calendars(string) ([]domain.Calendar, error)       { return m.cals, m.err }
func (m *memStore) Save(es ...domain.Event) error {
	if m.err != nil {
		return m.err
	}
	for _, e := range es {
		if e.ID == "" {
			m.next++
			e.ID = "e" + strconv.Itoa(m.next)
		}
		m.events[e.ID] = e
	}
	return nil
}
func (m *memStore) Between(time.Time, time.Time, ...string) ([]domain.Event, error) {
	var out []domain.Event
	for _, e := range m.events {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, m.err
}
func (m *memStore) Get(id string) (domain.Event, error) {
	if e, ok := m.events[id]; ok {
		return e, nil
	}
	return domain.Event{}, domain.ErrNotFound
}
func (m *memStore) Delete(id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.events, id)
	return nil
}

var now = time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

func day(d int) time.Time { return time.Date(2026, 5, d, 0, 0, 0, 0, time.UTC) }

func newScreen(t *testing.T, titles ...string) (*Screen, *memStore, application.Agenda) {
	t.Helper()
	st := &memStore{events: map[string]domain.Event{}}
	ag := application.Agenda{AccountID: "local", Store: st, Now: func() time.Time { return now }}
	for i, title := range titles {
		if _, err := ag.Add(application.Entry{Title: title, Day: day(20), Start: (8 + i) * 60, End: (9 + i) * 60}); err != nil {
			t.Fatal(err)
		}
	}
	s := NewScreen(Config{Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Agenda: ag})
	s.Doc.Layout()
	return s, st, ag
}

func byID(t *testing.T, s *Screen, id string) widget.Widget {
	t.Helper()
	var found widget.Widget
	s.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return true
	})
	if found == nil {
		t.Fatalf("no element %q", id)
	}
	return found
}

func tapAt(s *Screen, x, y float64) {
	s.Doc.HandlePointer(widget.PointerDown, x*s.Doc.Scale, y*s.Doc.Scale)
	s.Doc.HandlePointer(widget.PointerUp, x*s.Doc.Scale, y*s.Doc.Scale)
	s.Doc.Layout()
}

func tap(t *testing.T, s *Screen, id string) {
	t.Helper()
	s.Doc.Layout()
	c := byID(t, s, id).Elem().Bounds.Center()
	tapAt(s, c.X, c.Y)
}

func TestEveryTextIsTranslatedInBothLanguages(t *testing.T) {
	tr := NewTranslator("pt-BR")
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks: %v", lang, missing)
		}
	}
}

func TestTheScreenShowsTodayAndItsEvents(t *testing.T) {
	s, _, _ := newScreen(t, "Dentista", "Almoço")
	if !s.Selected().Equal(day(20)) {
		t.Fatalf("selected = %v", s.Selected())
	}
	if got := byID(t, s, "cal-day").(*widget.Label).Text; got != "Qua, 20/05/2026" {
		t.Errorf("heading = %q", got)
	}
	byID(t, s, "event-"+s.items[0].Event.ID)
	if len(s.items) != 2 || !s.marks[20] {
		t.Errorf("items = %d marks = %v", len(s.items), s.marks)
	}
}

func TestTappingADayShowsItsEventsAndMonthsReloadTheMarks(t *testing.T) {
	s, _, ag := newScreen(t, "hoje")
	ag.Add(application.Entry{Title: "amanhã", Day: day(21), Start: 600, End: 660})
	s.Select(day(20)) // reload with the new event marked
	s.Doc.Layout()
	cal := byID(t, s, "cal-month").(*widget.Calendar)
	s.Select(day(21))
	if len(s.items) != 1 || s.items[0].Event.Title != "amanhã" {
		t.Errorf("items = %+v", s.items)
	}
	s.Select(time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC))
	if cal.Month != time.June || len(s.items) != 0 {
		t.Errorf("month = %v items = %d", cal.Month, len(s.items))
	}
	tap(t, s, "cal-today")
	if !s.Selected().Equal(day(20)) || cal.Month != time.May {
		t.Errorf("today: %v %v", s.Selected(), cal.Month)
	}
}

func TestPagesAndKeys(t *testing.T) {
	titles := make([]string, 7)
	for i := range titles {
		titles[i] = "evento " + strconv.Itoa(i)
	}
	s, _, _ := newScreen(t, titles...)
	if s.pages() != 2 {
		t.Fatalf("pages = %d", s.pages())
	}
	tap(t, s, "cal-next")
	if s.Page() != 1 || len(s.body.Children()) != 2 {
		t.Errorf("page = %d rows = %d", s.Page(), len(s.body.Children()))
	}
	if !s.Key(true) || s.Page() != 0 {
		t.Errorf("previous key: %d", s.Page())
	}
	s.GoTo(9)
	if s.Page() != 1 {
		t.Errorf("clamped = %d", s.Page())
	}
}

func TestNewEventIsSavedOnTheSelectedDay(t *testing.T) {
	s, st, _ := newScreen(t)
	tap(t, s, "cal-new")
	if s.Editing() != "" || s.editor == nil {
		t.Fatal("the form did not open")
	}
	byID(t, s, "edit-title").(*widget.Input).Value = "Reunião"
	byID(t, s, "edit-location").(*widget.Input).Value = "Sala 2"
	byID(t, s, "edit-start").(*widget.Input).Value = "14:00"
	byID(t, s, "edit-end").(*widget.Input).Value = "15:30"
	tap(t, s, "edit-nextday")
	tap(t, s, "edit-save")
	if len(st.events) != 1 {
		t.Fatalf("events = %d", len(st.events))
	}
	for _, e := range st.events {
		if e.Title != "Reunião" || e.Location != "Sala 2" || e.Start.Hour() != 14 || e.Start.Day() != 21 || e.Duration() != 90*time.Minute {
			t.Errorf("event = %+v", e)
		}
	}
	if !s.Selected().Equal(day(21)) || len(s.items) != 1 || s.editor != nil {
		t.Errorf("after save: selected %v items %d", s.Selected(), len(s.items))
	}
}

func TestEditingAnEventChangesItAndAllDayIgnoresTimes(t *testing.T) {
	s, st, _ := newScreen(t, "a")
	id := s.items[0].Event.ID
	tap(t, s, "event-"+id)
	if s.Editing() != id {
		t.Fatalf("editing %q", s.Editing())
	}
	if got := byID(t, s, "edit-start").(*widget.Input).Value; got != "08:00" {
		t.Errorf("start = %q", got)
	}
	byID(t, s, "edit-title").(*widget.Input).Value = "b"
	tap(t, s, "edit-allday")
	byID(t, s, "edit-start").(*widget.Input).Value = "lixo"
	tap(t, s, "edit-prevday")
	tap(t, s, "edit-save")
	got := st.events[id]
	if got.Title != "b" || !got.AllDay || got.Start.Day() != 19 {
		t.Errorf("event = %+v", got)
	}
	// the all-day event is on the 19th now
	s.Select(day(19))
	if len(s.items) != 1 {
		t.Errorf("19th: %d items", len(s.items))
	}
}

func TestABadTimeOrAFailureKeepsTheFormOpen(t *testing.T) {
	s, st, _ := newScreen(t, "a")
	id := s.items[0].Event.ID
	s.Edit(id)
	byID(t, s, "edit-start").(*widget.Input).Value = "25:99"
	tap(t, s, "edit-save")
	if s.editor == nil || !strings.Contains(s.editor.note.Text, "HH:MM") {
		t.Errorf("note = %q", s.editor.note.Text)
	}
	byID(t, s, "edit-start").(*widget.Input).Value = "9:05"
	byID(t, s, "edit-title").(*widget.Input).Value = " "
	tap(t, s, "edit-save")
	if s.editor == nil || !strings.Contains(s.editor.note.Text, "title") {
		t.Errorf("an empty title was saved: %q", s.editor.note.Text)
	}
	byID(t, s, "edit-title").(*widget.Input).Value = "ok"
	st.err = errors.New("disco")
	tap(t, s, "edit-save")
	if s.editor == nil || !strings.Contains(s.editor.note.Text, "disco") {
		t.Errorf("note = %q", s.editor.note.Text)
	}
	tap(t, s, "edit-delete")
	if len(st.events) != 1 || !strings.Contains(s.editor.note.Text, "disco") {
		t.Errorf("a failed delete removed it: %d", len(st.events))
	}
}

func TestDeleteRemovesTheEventAndNothingBreaksWithoutAForm(t *testing.T) {
	s, st, _ := newScreen(t, "a")
	tap(t, s, "event-"+s.items[0].Event.ID)
	tap(t, s, "edit-delete")
	if len(st.events) != 0 || s.editor != nil || len(s.items) != 0 {
		t.Errorf("events = %d items = %d", len(st.events), len(s.items))
	}
	s.Edit("missing")
	if !strings.Contains(s.notice, "Não foi possível") {
		t.Errorf("notice = %q", s.notice)
	}
	var none Screen
	none.SaveEdit()
	none.DeleteEdit()
	if s.Key(true) != true {
		t.Error("the page keys must work on the month")
	}
	s.NewEvent()
	if s.Key(false) {
		t.Error("the page keys must be left to the form")
	}
}

func TestParseClock(t *testing.T) {
	for in, want := range map[string]int{"09:30": 570, " 9:05 ": 545, "00:00": 0, "23:59": 1439} {
		if got, ok := parseClock(in); !ok || got != want {
			t.Errorf("%q = %d %v", in, got, ok)
		}
	}
	for _, bad := range []string{"", "24:00", "10:60", "1030", "a:b", "10:5:5", "-1:00"} {
		if _, ok := parseClock(bad); ok {
			t.Errorf("%q accepted", bad)
		}
	}
}

func checkElement(t *testing.T, s *Screen, where string, w widget.Widget, screenH float64) {
	t.Helper()
	e := w.Elem()
	if !e.Visible() {
		return
	}
	if e.Bounds.Y+e.Bounds.H > screenH+0.5 && e.ID != "" {
		t.Errorf("%s: %s overflows the screen (%v)", where, e.ID, e.Bounds)
	}
	if _, ok := w.(widget.PointerHandler); !ok || e.Disabled || e.Bounds.W <= 0 {
		return
	}
	if got := s.Doc.TargetAt(e.Bounds.Center()); got != w {
		t.Errorf("%s: %s (%T) does not get its own touch", where, e.ID, w)
	}
	if e.Bounds.H < 56 {
		t.Errorf("%s: %s is only %.0f px high", where, e.ID, e.Bounds.H)
	}
}

func TestEveryInteractiveElementReceivesItsOwnTouchAndNothingOverflows(t *testing.T) {
	s, _, _ := newScreen(t, "a", "b", "c", "d", "e")
	check := func(where string) {
		t.Helper()
		s.Doc.Layout()
		h := s.Doc.Root.Elem().Bounds.H
		s.Doc.Walk(func(w widget.Widget) bool {
			checkElement(t, s, where, w, h)
			return true
		})
	}
	check("month")
	s.Edit(s.items[0].Event.ID)
	check("editor")
}
