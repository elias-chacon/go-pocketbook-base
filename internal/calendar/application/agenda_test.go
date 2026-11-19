package application

import (
	"errors"
	"sort"
	"strconv"
	"testing"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
)

type memStore struct {
	cals   []domain.Calendar
	events map[string]domain.Event
	next   int
	err    error
}

func newMem() *memStore { return &memStore{events: map[string]domain.Event{}} }

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

func agenda(zone string) (Agenda, *memStore) {
	st := newMem()
	return Agenda{AccountID: "local", Store: st, Zone: zone, Now: func() time.Time { return now }}, st
}

func day(d int) time.Time { return time.Date(2026, 5, d, 0, 0, 0, 0, time.UTC) }

func TestAddCreatesTheCalendarAndTheEvent(t *testing.T) {
	a, st := agenda("")
	e, err := a.Add(Entry{Title: "  Dentista ", Location: "Centro", Day: day(20), Start: 9 * 60, End: 10*60 + 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.cals) != 1 || st.cals[0].ID != DefaultCalendarID {
		t.Errorf("calendars = %+v", st.cals)
	}
	if e.Title != "Dentista" || e.UID == "" || e.Duration() != 90*time.Minute {
		t.Errorf("event = %+v", e)
	}
	if _, err := a.Add(Entry{Title: " ", Day: day(20)}); !errors.Is(err, ErrEmptyTitle) {
		t.Errorf("err = %v", err)
	}
	// an end before the start is made an instant, never invalid
	e2, err := a.Add(Entry{Title: "x", Day: day(21), Start: 600, End: 500})
	if err != nil || !e2.End.Equal(e2.Start) {
		t.Errorf("e2 = %+v %v", e2, err)
	}
}

func TestDayListsAllDayFirstThenByStart(t *testing.T) {
	a, _ := agenda("")
	a.Add(Entry{Title: "tarde", Day: day(20), Start: 15 * 60, End: 16 * 60})
	a.Add(Entry{Title: "manhã", Day: day(20), Start: 8 * 60, End: 9 * 60})
	a.Add(Entry{Title: "feriado", Day: day(20), AllDay: true})
	a.Add(Entry{Title: "outro dia", Day: day(21), Start: 600, End: 660})
	got, err := a.Day(day(20))
	if err != nil || len(got) != 3 {
		t.Fatalf("day = %d %v", len(got), err)
	}
	if got[0].Event.Title != "feriado" || got[1].Event.Title != "manhã" || got[2].Event.Title != "tarde" {
		t.Errorf("order = %s, %s, %s", got[0].Event.Title, got[1].Event.Title, got[2].Event.Title)
	}
}

func TestAnAllDayEventIsADateWhateverTheZone(t *testing.T) {
	a, _ := agenda("America/Sao_Paulo")
	a.Add(Entry{Title: "feriado", Day: day(20), AllDay: true})
	for d, want := range map[int]int{19: 0, 20: 1, 21: 0} {
		got, _ := a.Day(day(d))
		if len(got) != want {
			t.Errorf("day %d: %d events, want %d", d, len(got), want)
		}
	}
}

func TestMarksShowTheDaysWithEvents(t *testing.T) {
	a, _ := agenda("America/Sao_Paulo")
	a.Add(Entry{Title: "a", Day: day(3), Start: 600, End: 660})
	a.Add(Entry{Title: "b", Day: day(31), Start: 23 * 60, End: 23*60 + 30})
	marks, err := a.Marks(2026, time.May)
	if err != nil || len(marks) != 2 || !marks[3] || !marks[31] {
		t.Errorf("marks = %v %v", marks, err)
	}
}

func TestARepeatingEventAppearsOnItsDays(t *testing.T) {
	a, st := agenda("America/Sao_Paulo")
	e, _ := a.Add(Entry{Title: "yoga", Day: day(4), Start: 7 * 60, End: 8 * 60})
	e.Recurrence = "FREQ=WEEKLY"
	st.events[e.ID] = e
	marks, _ := a.Marks(2026, time.May)
	if !marks[4] || !marks[11] || !marks[18] || !marks[25] || marks[5] {
		t.Errorf("marks = %v", marks)
	}
}

func TestChangeAndDelete(t *testing.T) {
	a, st := agenda("")
	e, _ := a.Add(Entry{Title: "a", Day: day(5), Start: 600, End: 660})
	if err := a.Change(e.ID, Entry{Title: "b", Day: day(6), AllDay: true}); err != nil {
		t.Fatal(err)
	}
	got, _ := a.Get(e.ID)
	if got.Title != "b" || !got.AllDay || got.Sequence != 1 {
		t.Errorf("event = %+v", got)
	}
	if err := a.Change("nope", Entry{Title: "x", Day: day(1)}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v", err)
	}
	if err := a.Change(e.ID, Entry{Title: "", Day: day(1)}); !errors.Is(err, ErrEmptyTitle) {
		t.Errorf("err = %v", err)
	}
	if err := a.Delete(e.ID); err != nil || len(st.events) != 0 {
		t.Errorf("delete: %v %d", err, len(st.events))
	}
}

func TestZoneAndStoreErrors(t *testing.T) {
	a, st := agenda("Nowhere/Land")
	if a.Loc() != time.UTC {
		t.Error("an unknown zone must fall back to UTC")
	}
	if got := a.Today(); !got.Equal(day(20)) {
		t.Errorf("today = %v", got)
	}
	st.err = errors.New("disco")
	if _, err := a.Add(Entry{Title: "x", Day: day(1)}); err == nil {
		t.Error("add ignored the store error")
	}
	if _, err := a.Day(day(1)); err == nil {
		t.Error("day ignored the store error")
	}
	if _, err := a.Marks(2026, time.May); err == nil {
		t.Error("marks ignored the store error")
	}
	if (Agenda{}).now().IsZero() {
		t.Error("the default clock gave zero")
	}
}
