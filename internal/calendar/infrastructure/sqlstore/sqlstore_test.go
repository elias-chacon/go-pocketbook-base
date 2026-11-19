package sqlstore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
	"go-pocketbook-base/internal/storage"
)

func utc(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, time.UTC) }

func newStore(t *testing.T) (*Store, *storage.Store) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveCalendars("ana", []domain.Calendar{{ID: "work", Name: "Trabalho"}, {ID: "home", Name: "Casa"}}); err != nil {
		t.Fatal(err)
	}
	return s, db
}

func event(calendar, uid, title string, start time.Time, hours int) domain.Event {
	return domain.Event{CalendarID: calendar, UID: uid, Title: title, Start: start, End: start.Add(time.Duration(hours) * time.Hour)}
}

func titles(events []domain.Event) string {
	var s []string
	for _, e := range events {
		s = append(s, e.Title)
	}
	return strings.Join(s, "|")
}

func TestAnEventRoundTripsWithEveryField(t *testing.T) {
	s, _ := newStore(t)
	in := domain.Event{
		CalendarID: "work", RemoteID: "r1", UID: "u1", Title: "Reunião", Description: "Pauta", Location: "Sala 3",
		Start: utc(2026, 3, 2, 12), End: utc(2026, 3, 2, 13), TimeZone: "America/Sao_Paulo",
		Recurrence: "FREQ=WEEKLY;COUNT=4", Exceptions: []time.Time{utc(2026, 3, 9, 12)},
		Reminders: []time.Duration{15 * time.Minute, 24 * time.Hour},
		Organizer: domain.Attendee{Name: "Ana", Email: "ana@example.org"},
		Attendees: []domain.Attendee{{Name: "Eu", Email: "eu@example.org", Response: domain.Accepted}},
		Status:    domain.Tentative, Sequence: 3,
	}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("work:u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != in.Title || got.Description != in.Description || got.Location != in.Location || got.RemoteID != "r1" || got.TimeZone != in.TimeZone {
		t.Errorf("%+v", got)
	}
	if !got.Start.Equal(in.Start) || !got.End.Equal(in.End) || got.AllDay || got.Recurrence != in.Recurrence || got.Sequence != 3 || got.Status != domain.Tentative {
		t.Errorf("%+v", got)
	}
	if len(got.Exceptions) != 1 || !got.Exceptions[0].Equal(in.Exceptions[0]) || len(got.Reminders) != 2 || got.Reminders[1] != 24*time.Hour {
		t.Errorf("exceptions and reminders: %+v %+v", got.Exceptions, got.Reminders)
	}
	if got.Organizer.Email != "ana@example.org" || len(got.Attendees) != 1 || got.Attendees[0].Response != domain.Accepted || !got.RecurrenceID.IsZero() {
		t.Errorf("people: %+v %+v", got.Organizer, got.Attendees)
	}
	if _, err := s.Get("ghost"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}

func TestSavingAgainReplacesAndAnUnknownCalendarIsRefused(t *testing.T) {
	s, _ := newStore(t)
	s.Save(event("work", "u1", "Antes", utc(2026, 3, 2, 9), 1))
	s.Save(event("work", "u1", "Depois", utc(2026, 3, 2, 10), 1))
	got, _ := s.Between(utc(2026, 3, 1, 0), utc(2026, 3, 3, 0))
	if len(got) != 1 || got[0].Title != "Depois" {
		t.Errorf("%s", titles(got))
	}
	if err := s.Save(event("nowhere", "u2", "X", utc(2026, 3, 2, 9), 1)); err == nil {
		t.Error("an event needs a calendar the store knows")
	}
}

func TestBetweenFindsWhatCanAppearInTheRange(t *testing.T) {
	s, _ := newStore(t)
	allDay := domain.Event{CalendarID: "home", UID: "trip", Title: "Viagem", AllDay: true, Start: utc(2026, 3, 5, 0), End: utc(2026, 3, 8, 0)}
	instant := domain.Event{CalendarID: "work", UID: "ping", Title: "Aviso", Start: utc(2026, 3, 10, 9), End: utc(2026, 3, 10, 9)}
	weekly := event("work", "weekly", "Semanal", utc(2026, 1, 5, 9), 1)
	weekly.Recurrence = "FREQ=WEEKLY"
	s.Save(
		event("work", "before", "Antes", utc(2026, 3, 1, 9), 1),
		event("work", "inside", "Dentro", utc(2026, 3, 10, 14), 2),
		event("home", "across", "Atravessa", utc(2026, 3, 9, 22), 12), // until 10:00 on the 10th
		event("work", "after", "Depois", utc(2026, 3, 12, 0), 1),
		allDay, instant, weekly,
		event("work", "touching", "Termina quando começa", utc(2026, 3, 10, 7), 1), // ends at 8:00, the range starts at 8:00
	)
	from, to := utc(2026, 3, 10, 8), utc(2026, 3, 11, 0)

	got, err := s.Between(from, to)
	if err != nil {
		t.Fatal(err)
	}
	// the single ones that overlap, the instant inside the range, and the repeating one that began earlier
	if titles(got) != "Semanal|Atravessa|Aviso|Dentro" {
		t.Errorf("%s", titles(got))
	}
	if got, _ := s.Between(from, to, "home"); titles(got) != "Atravessa" {
		t.Errorf("only the home calendar: %s", titles(got))
	}
	if got, _ := s.Between(from, to, "home", "work"); len(got) != 4 {
		t.Errorf("both calendars: %s", titles(got))
	}
	if got, _ := s.Between(utc(2026, 3, 6, 0), utc(2026, 3, 7, 0)); !strings.Contains(titles(got), "Viagem") {
		t.Errorf("an all-day event inside the range: %s", titles(got))
	}
}

func TestTheStoreFeedsExpandWithExceptionsAndChangedOccurrences(t *testing.T) {
	s, _ := newStore(t)
	weekly := event("work", "u1", "Reunião", utc(2026, 3, 2, 10), 1) // a Monday
	weekly.Recurrence = "FREQ=WEEKLY;COUNT=5"
	weekly.Exceptions = []time.Time{utc(2026, 3, 16, 10)}
	moved := event("work", "u1", "Reunião (adiada)", utc(2026, 3, 9, 15), 1)
	moved.RecurrenceID = utc(2026, 3, 9, 10)
	farMoved := event("work", "u1", "Reunião (para abril)", utc(2026, 4, 20, 10), 1) // moved out of the range, from inside it
	farMoved.RecurrenceID = utc(2026, 3, 23, 10)
	s.Save(weekly, moved, farMoved)

	events, err := s.Between(utc(2026, 3, 1, 0), utc(2026, 3, 31, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("the master and both changed occurrences: %s", titles(events))
	}
	occ, err := domain.Expand(events, utc(2026, 3, 1, 0), utc(2026, 3, 31, 0))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, o := range occ {
		got = append(got, o.Start.Format("02 15h")+" "+o.Event.Title)
	}
	// 2nd normal, 9th moved to the afternoon, 16th deleted, 23rd moved to April, 30th normal
	if strings.Join(got, "|") != "02 10h Reunião|09 15h Reunião (adiada)|30 10h Reunião" {
		t.Errorf("%s", strings.Join(got, "|"))
	}
}

func TestDeletingARepeatingEventDeletesItsChangedOccurrences(t *testing.T) {
	s, _ := newStore(t)
	weekly := event("work", "u1", "Reunião", utc(2026, 3, 2, 10), 1)
	weekly.Recurrence = "FREQ=WEEKLY"
	moved := event("work", "u1", "Adiada", utc(2026, 3, 9, 15), 1)
	moved.RecurrenceID = utc(2026, 3, 9, 10)
	other := event("work", "u2", "Outra", utc(2026, 3, 3, 10), 1)
	s.Save(weekly, moved, other)

	if err := s.Delete(fmt.Sprintf("work:u1:%d", moved.RecurrenceID.Unix())); err != nil { // just the changed occurrence
		t.Fatal(err)
	}
	if got, _ := s.Between(utc(2026, 3, 1, 0), utc(2026, 4, 1, 0)); titles(got) != "Reunião|Outra" {
		t.Errorf("one changed occurrence deleted: %s", titles(got))
	}
	if err := s.Delete("work:u1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Between(utc(2026, 3, 1, 0), utc(2026, 4, 1, 0)); titles(got) != "Outra" {
		t.Errorf("the series is gone: %s", titles(got))
	}
	if err := s.Delete("work:u1"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}

func TestCalendarsFollowTheServerAndTakeTheirEventsWithThem(t *testing.T) {
	s, db := newStore(t)
	s.Save(event("work", "a", "Do trabalho", utc(2026, 3, 2, 9), 1), event("home", "b", "De casa", utc(2026, 3, 2, 9), 1))

	cals, _ := s.Calendars("ana")
	if len(cals) != 2 || cals[0].Name != "Casa" || cals[1].Name != "Trabalho" {
		t.Errorf("by name: %+v", cals)
	}
	if err := s.SaveCalendars("ana", []domain.Calendar{{ID: "work", Name: "Trabalho 2", ReadOnly: true}}); err != nil {
		t.Fatal(err)
	}
	cals, _ = s.Calendars("ana")
	if len(cals) != 1 || cals[0].Name != "Trabalho 2" || !cals[0].ReadOnly {
		t.Errorf("%+v", cals)
	}
	if got, _ := s.Between(utc(2026, 3, 1, 0), utc(2026, 3, 3, 0)); titles(got) != "Do trabalho" {
		t.Errorf("the home calendar took its events: %s", titles(got))
	}

	if err := db.Erase("ana"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Between(utc(2026, 3, 1, 0), utc(2026, 3, 3, 0)); len(got) != 0 {
		t.Error("erasing the account clears its events")
	}
	if cals, _ := s.Calendars("ana"); len(cals) != 0 {
		t.Error("and its calendars")
	}
	if err := s.SaveCalendars("ana", nil); err != nil {
		t.Errorf("no calendars at all is fine: %v", err)
	}
}

func TestADamagedRowIsReportedAndAClosedDatabaseFailsCleanly(t *testing.T) {
	s, db := newStore(t)
	s.Save(event("work", "a", "A", utc(2026, 3, 2, 9), 1))
	db.DB().Exec(`UPDATE calendar_events SET attendees = 'not json'`)
	if _, err := s.Get("work:a"); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Errorf("%v", err)
	}
	if _, err := s.Between(utc(2026, 3, 1, 0), utc(2026, 3, 3, 0)); err == nil {
		t.Error("a list with a damaged row is an error")
	}

	db.Close()
	if err := s.Save(event("work", "b", "B", utc(2026, 3, 2, 9), 1)); err == nil {
		t.Error("Save")
	}
	if err := s.SaveCalendars("ana", nil); err == nil {
		t.Error("SaveCalendars")
	}
	if _, err := s.Calendars("ana"); err == nil {
		t.Error("Calendars")
	}
	if _, err := s.Between(utc(2026, 3, 1, 0), utc(2026, 3, 3, 0)); err == nil {
		t.Error("Between")
	}
	if err := s.Delete("work:a"); err == nil || err == domain.ErrNotFound {
		t.Errorf("Delete: %v", err)
	}
}
