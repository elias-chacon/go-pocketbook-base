package application

import (
	"errors"
	"testing"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
)

type fakeBackend struct {
	cals    []domain.Calendar
	events  map[string][]domain.Event
	calsErr error
	evErr   error
}

func (f *fakeBackend) Calendars() ([]domain.Calendar, error) { return f.cals, f.calsErr }
func (f *fakeBackend) Events(id string, _, _ time.Time) ([]domain.Event, error) {
	return append([]domain.Event(nil), f.events[id]...), f.evErr
}

func ev(uid, title string) domain.Event {
	return domain.Event{UID: uid, Title: title, Start: day(20).Add(9 * time.Hour), End: day(20).Add(10 * time.Hour)}
}

func TestSyncBringsTheCalendarsReadOnlyAndTheirEventsAndDropsWhatTheServiceLost(t *testing.T) {
	st := newMem()
	be := &fakeBackend{cals: []domain.Calendar{{ID: "A", Name: "Work"}, {ID: "B", Name: "Home"}},
		events: map[string][]domain.Event{"A": {ev("a1", "Standup"), ev("a2", "Review")}, "B": {ev("b1", "Dentist")}}}
	s := Sync{AccountID: "ana", Store: st, Backend: be, Now: func() time.Time { return now }}
	res, err := s.Run()
	if err != nil || res.Calendars != 2 || res.Events != 3 {
		t.Fatalf("%+v %v", res, err)
	}
	if len(st.cals) != 2 || st.cals[0].ID != "ana:A" || !st.cals[0].ReadOnly || st.cals[0].AccountID != "ana" {
		t.Errorf("calendars = %+v", st.cals)
	}
	for _, e := range st.events {
		if e.CalendarID == "" || e.RemoteID != e.UID || e.Status != domain.Confirmed {
			t.Errorf("event = %+v", e)
		}
	}
	// a2 was deleted online; a1 was renamed
	be.events["A"] = []domain.Event{ev("a1", "Standup (moved)")}
	res, err = s.Run()
	if err != nil || res.Events != 2 {
		t.Fatalf("%+v %v", res, err)
	}
	titles := map[string]bool{}
	for _, e := range st.events {
		titles[e.Title] = true
	}
	if titles["Review"] || !titles["Standup (moved)"] {
		t.Errorf("stale events must go and changed ones be replaced: %v", titles)
	}
}

func TestSyncReportsWhichStepFailed(t *testing.T) {
	be := &fakeBackend{calsErr: errors.New("401")}
	if _, err := (Sync{AccountID: "ana", Store: newMem(), Backend: be}).Run(); err == nil || err.Error() != "calendars: 401" {
		t.Errorf("%v", err)
	}
	be = &fakeBackend{cals: []domain.Calendar{{ID: "A", Name: "Work"}}, evErr: errors.New("boom")}
	if _, err := (Sync{AccountID: "ana", Store: newMem(), Backend: be}).Run(); err == nil || err.Error() != "Work: boom" {
		t.Errorf("%v", err)
	}
}
