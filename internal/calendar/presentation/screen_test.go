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

func firstID(st *memStore) string {
	id := ""
	for k := range st.events {
		id = k
	}
	return id
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

func tap(t *testing.T, s *Screen, id string) {
	t.Helper()
	s.Doc.Layout()
	c := byID(t, s, id).Elem().Bounds.Center()
	s.Doc.HandlePointer(widget.PointerDown, c.X*s.Doc.Scale, c.Y*s.Doc.Scale)
	s.Doc.HandlePointer(widget.PointerUp, c.X*s.Doc.Scale, c.Y*s.Doc.Scale)
	s.Doc.Layout()
}

func title(t *testing.T, s *Screen) string {
	t.Helper()
	return byID(t, s, "cal-title").(*widget.Label).Text
}

func TestEveryTextIsTranslatedInBothLanguages(t *testing.T) {
	tr := NewTranslator("pt-BR")
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks: %v", lang, missing)
		}
	}
}

func TestItOpensOnTheMonthLikeOutlookNotOnADatePicker(t *testing.T) {
	s, _, _ := newScreen(t, "Dentista")
	if s.CurrentView() != MonthView || title(t, s) != "Maio 2026" {
		t.Fatalf("view %v title %q", s.CurrentView(), title(t, s))
	}
	mv := byID(t, s, "cal-month").(*widget.MonthView)
	if got := mv.Entries(day(20)); len(got) != 1 || got[0] != "08:00 Dentista" {
		t.Errorf("the day lists its events: %v", got)
	}
	if got := mv.Entries(day(21)); len(got) != 0 {
		t.Errorf("an empty day lists nothing: %v", got)
	}
	if mv.Entries(day(1).AddDate(0, -3, 0)) != nil {
		t.Error("a day outside the grid lists nothing")
	}
}

func TestTappingADayTwiceOpensItInTheDayView(t *testing.T) {
	s, _, _ := newScreen(t, "Dentista")
	mv := byID(t, s, "cal-month").(*widget.MonthView)
	mv.Selected = day(21)
	mv.OnChange(nil)
	if !s.Selected().Equal(day(21)) || !strings.Contains(title(t, s), "Maio") {
		t.Errorf("selected %v title %q", s.Selected(), title(t, s))
	}
	mv.OnOpen(nil)
	if s.CurrentView() != DayView || title(t, s) != "Qui, 21/05/2026" {
		t.Errorf("view %v title %q", s.CurrentView(), title(t, s))
	}
}

func TestTheViewMenuAndNavigationMoveThePeriod(t *testing.T) {
	s, _, _ := newScreen(t)
	byID(t, s, "cal-views").(*widget.Dropdown).Items[1].OnSelect() // work week
	if s.CurrentView() != WorkWeekView || title(t, s) != "18/05 – 22/05/2026" {
		t.Fatalf("work week: %v %q", s.CurrentView(), title(t, s))
	}
	tap(t, s, "cal-next")
	if title(t, s) != "25/05 – 29/05/2026" {
		t.Errorf("next week: %q", title(t, s))
	}
	byID(t, s, "cal-views").(*widget.Dropdown).Items[2].OnSelect() // week (Sunday first)
	if title(t, s) != "24/05 – 30/05/2026" {
		t.Errorf("week: %q", title(t, s))
	}
	tap(t, s, "cal-today")
	byID(t, s, "cal-views").(*widget.Dropdown).Items[0].OnSelect() // day
	tap(t, s, "cal-prev")
	if title(t, s) != "Ter, 19/05/2026" {
		t.Errorf("day: %q", title(t, s))
	}
	byID(t, s, "cal-views").(*widget.Dropdown).Items[3].OnSelect() // month
	s.Select(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC))
	if !s.Key(false) || title(t, s) != "Fevereiro 2026" || s.Selected().Day() != 28 {
		t.Errorf("the 31st must clamp to the end of the next month: %v %q", s.Selected(), title(t, s))
	}
	if !s.Key(true) || title(t, s) != "Janeiro 2026" {
		t.Errorf("previous: %q", title(t, s))
	}
}

func TestTheGridShowsEventsAndOpensThemOrNewOnesOnTouch(t *testing.T) {
	s, st, _ := newScreen(t, "Dentista", "Almoço")
	s.SetView(WorkWeekView)
	g := byID(t, s, "cal-grid").(*widget.TimeGrid)
	if len(g.Events) != 2 || g.Events[0].Title != "Dentista" || g.Events[0].Detail != "08:00–09:00" {
		t.Fatalf("events = %+v", g.Events)
	}
	g.OnEvent(g.Events[0].ID)
	if s.Editing() == "" || byID(t, s, "edit-title").(*widget.Input).Value != "Dentista" {
		t.Fatal("the event did not open")
	}
	tap(t, s, "edit-back")
	g = byID(t, s, "cal-grid").(*widget.TimeGrid)
	g.OnSlot(time.Date(2026, 5, 21, 14, 30, 0, 0, time.UTC))
	if byID(t, s, "edit-start").(*widget.Input).Value != "14:30" || byID(t, s, "edit-end").(*widget.Input).Value != "15:30" {
		t.Error("a tap on an empty slot must start a one-hour event there")
	}
	byID(t, s, "edit-title").(*widget.Input).Value = "Reunião"
	tap(t, s, "edit-save")
	if len(st.events) != 3 || !s.Selected().Equal(day(21)) {
		t.Errorf("events %d selected %v", len(st.events), s.Selected())
	}
	g = byID(t, s, "cal-grid").(*widget.TimeGrid)
	g.OnDay(day(19))
	if s.CurrentView() != DayView || !s.Selected().Equal(day(19)) {
		t.Errorf("a day header opens the Day view: %v %v", s.CurrentView(), s.Selected())
	}
}

func TestTheGridWidensItsHoursToShowEarlyAndLateEvents(t *testing.T) {
	s, _, ag := newScreen(t)
	ag.Add(application.Entry{Title: "voo", Day: day(20), Start: 5 * 60, End: 6 * 60})
	ag.Add(application.Entry{Title: "festa", Day: day(20), Start: 22 * 60, End: 23*60 + 30})
	ag.Add(application.Entry{Title: "feriado", Day: day(20), AllDay: true})
	s.Select(day(20))
	s.SetView(DayView)
	g := byID(t, s, "cal-grid").(*widget.TimeGrid)
	if g.FirstHour != 5 || g.LastHour != 24 || len(g.Events) != 3 {
		t.Errorf("hours %d..%d events %d", g.FirstHour, g.LastHour, len(g.Events))
	}
}

func TestEditingChangesAnEventAndAllDayIgnoresTimes(t *testing.T) {
	s, st, _ := newScreen(t, "a")
	id := firstID(st)
	s.Edit(id)
	if s.Editing() != id || byID(t, s, "edit-start").(*widget.Input).Value != "08:00" {
		t.Fatalf("editing %q", s.Editing())
	}
	byID(t, s, "edit-title").(*widget.Input).Value = "b"
	byID(t, s, "edit-description").(*widget.TextArea).Value = "levar exames"
	tap(t, s, "edit-allday")
	byID(t, s, "edit-start").(*widget.Input).Value = "lixo"
	tap(t, s, "edit-prevday")
	tap(t, s, "edit-save")
	got := st.events[id]
	if got.Title != "b" || got.Description != "levar exames" || !got.AllDay || got.Start.Day() != 19 {
		t.Errorf("event = %+v", got)
	}
}

func TestABadTimeOrAFailureKeepsTheFormOpen(t *testing.T) {
	s, st, _ := newScreen(t, "a")
	s.Edit(firstID(st))
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
	s.Edit(firstID(st))
	tap(t, s, "edit-delete")
	if len(st.events) != 0 || s.editor != nil {
		t.Errorf("events = %d", len(st.events))
	}
	s.Edit("missing")
	if !strings.Contains(s.notice, "Não foi possível") {
		t.Errorf("notice = %q", s.notice)
	}
	var none Screen
	none.SaveEdit()
	none.DeleteEdit()
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

func TestTheRepetitionIsChosenFromTheMenuAndKeptWhenCustom(t *testing.T) {
	s, st, _ := newScreen(t, "yoga")
	id := firstID(st)
	s.Edit(id)
	rep := byID(t, s, "edit-repeat").(*widget.Dropdown)
	if !strings.Contains(rep.Text, "Não repete") {
		t.Errorf("label = %q", rep.Text)
	}
	rep.Items[2].OnSelect()
	if !strings.Contains(rep.Text, "Toda semana") {
		t.Errorf("label = %q", rep.Text)
	}
	tap(t, s, "edit-save")
	if st.events[id].Recurrence != "FREQ=WEEKLY" {
		t.Fatalf("recurrence = %q", st.events[id].Recurrence)
	}
	e := st.events[id]
	e.Recurrence = "FREQ=WEEKLY;BYDAY=MO,WE"
	st.events[id] = e
	s.Edit(id)
	if !strings.Contains(byID(t, s, "edit-repeat").(*widget.Dropdown).Text, "Personalizada") {
		t.Error("custom rule not shown as custom")
	}
	tap(t, s, "edit-save")
	if st.events[id].Recurrence != "FREQ=WEEKLY;BYDAY=MO,WE" {
		t.Errorf("custom rule lost: %q", st.events[id].Recurrence)
	}
}

func TestTheReminderIsChosenAndSaved(t *testing.T) {
	s, st, _ := newScreen(t, "dentista")
	id := firstID(st)
	s.Edit(id)
	rem := byID(t, s, "edit-reminder").(*widget.Dropdown)
	rem.Items[2].OnSelect()
	if !strings.Contains(rem.Text, "15 minutos antes") {
		t.Errorf("label = %q", rem.Text)
	}
	tap(t, s, "edit-save")
	if r := st.events[id].Reminders; len(r) != 1 || r[0] != 15*time.Minute {
		t.Fatalf("reminders = %v", r)
	}
	e := st.events[id]
	e.Reminders = []time.Duration{45 * time.Minute}
	st.events[id] = e
	s.Edit(id)
	if !strings.Contains(byID(t, s, "edit-reminder").(*widget.Dropdown).Text, "45 minutos") {
		t.Error("an unusual reminder is not shown")
	}
}

func TestAStoreFailureIsShownInsteadOfBreakingTheView(t *testing.T) {
	s, st, _ := newScreen(t, "a")
	st.err = errors.New("disco")
	s.SetView(DayView)
	if !strings.Contains(s.notice, "disco") {
		t.Errorf("notice = %q", s.notice)
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

func TestEveryViewFitsTheScreenAndEveryElementGetsItsTouch(t *testing.T) {
	s, st, _ := newScreen(t, "a", "b", "c", "d", "e")
	check := func(where string) {
		t.Helper()
		s.Doc.Layout()
		h := s.Doc.Root.Elem().Bounds.H
		s.Doc.Walk(func(w widget.Widget) bool {
			checkElement(t, s, where, w, h)
			return true
		})
	}
	for _, v := range []View{MonthView, DayView, WorkWeekView, WeekView} {
		s.SetView(v)
		check("view " + strconv.Itoa(int(v)))
	}
	s.Edit(firstID(st))
	check("editor")
}

type fakeImporter struct {
	files    []string
	count    int
	warnings []string
	err      error
	got      string
}

func (f *fakeImporter) Files() []string { return f.files }
func (f *fakeImporter) Import(src string) (int, []string, error) {
	f.got = src
	return f.count, f.warnings, f.err
}

func TestImportScreenImportsFromAnAddressAndFiles(t *testing.T) {
	imp := &fakeImporter{files: []string{"/a/one.ics"}, count: 3}
	s := NewScreen(Config{Width: 1404, Height: 1872, Tr: NewTranslator("en"), Agenda: application.Agenda{AccountID: "local", Store: &memStore{events: map[string]domain.Event{}}, Now: func() time.Time { return now }}, Importer: imp})
	s.Doc.Layout()
	tap(t, s, "cal-import")
	status := func() string { return byID(t, s, "import-status").(*widget.Label).Text }
	tap(t, s, "import-fetch")
	if !strings.Contains(status(), "Type an address") {
		t.Fatalf("status = %q", status())
	}
	byID(t, s, "import-url").(*widget.Input).Value = " webcal://x/y.ics "
	tap(t, s, "import-fetch")
	if imp.got != "webcal://x/y.ics" || !strings.Contains(status(), "Imported 3") {
		t.Fatalf("got %q status %q", imp.got, status())
	}
	imp.warnings = []string{"bad date"}
	tap(t, s, "import-file-0")
	if imp.got != "/a/one.ics" || !strings.Contains(status(), "bad date") {
		t.Fatalf("got %q status %q", imp.got, status())
	}
	imp.err = errors.New("boom")
	tap(t, s, "import-file-0")
	if !strings.Contains(status(), "boom") {
		t.Fatalf("status = %q", status())
	}
	tap(t, s, "import-back")
	byID(t, s, "cal-import")
}

func TestImportScreenSaysWhenNoFileWasFound(t *testing.T) {
	s := NewScreen(Config{Width: 1404, Height: 1872, Tr: NewTranslator("en"), Agenda: application.Agenda{AccountID: "local", Store: &memStore{events: map[string]domain.Event{}}, Now: func() time.Time { return now }}, Importer: &fakeImporter{}})
	s.ShowImport()
	s.Doc.Layout()
	byID(t, s, "import-fetch")
}
