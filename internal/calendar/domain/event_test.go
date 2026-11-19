package domain

import (
	"strings"
	"testing"
	"time"
)

func utc(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.UTC)
}

func meeting() Event {
	return Event{UID: "u1", Title: "Reunião", Start: utc(2026, 1, 5, 10, 0), End: utc(2026, 1, 5, 11, 0)}
}

func TestASingleEventAppearsOnlyWhenItOverlapsTheRange(t *testing.T) {
	e := meeting()
	cases := []struct {
		name     string
		from, to time.Time
		want     int
	}{
		{"the day", utc(2026, 1, 5, 0, 0), utc(2026, 1, 6, 0, 0), 1},
		{"another day", utc(2026, 1, 6, 0, 0), utc(2026, 1, 7, 0, 0), 0},
		{"ends exactly when the range starts", utc(2026, 1, 5, 11, 0), utc(2026, 1, 5, 12, 0), 0},
		{"starts exactly when the range ends", utc(2026, 1, 5, 9, 0), utc(2026, 1, 5, 10, 0), 0},
		{"starts before and still runs", utc(2026, 1, 5, 10, 30), utc(2026, 1, 5, 12, 0), 1},
	}
	for _, c := range cases {
		got, err := e.Occurrences(c.from, c.to)
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %d occurrences, %v", c.name, len(got), err)
		}
	}
	instant := Event{Title: "Aviso", Start: utc(2026, 1, 5, 10, 0), End: utc(2026, 1, 5, 10, 0)}
	if got, _ := instant.Occurrences(utc(2026, 1, 5, 10, 0), utc(2026, 1, 5, 10, 1)); len(got) != 1 {
		t.Error("an event with no length counts at its instant")
	}
}

func TestAWeeklyEventRepeatsAndSkipsItsExceptions(t *testing.T) {
	e := meeting()
	e.Recurrence = "FREQ=WEEKLY;COUNT=5"
	e.Exceptions = []time.Time{utc(2026, 1, 12, 10, 0)}

	got, err := e.Occurrences(utc(2026, 1, 1, 0, 0), utc(2026, 3, 1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{utc(2026, 1, 5, 10, 0), utc(2026, 1, 19, 10, 0), utc(2026, 1, 26, 10, 0), utc(2026, 2, 2, 10, 0)}
	if len(got) != len(want) {
		t.Fatalf("%d occurrences, want %d", len(got), len(want))
	}
	for i, w := range want {
		if !got[i].Start.Equal(w) || !got[i].End.Equal(w.Add(time.Hour)) {
			t.Errorf("occurrence %d: %v-%v", i, got[i].Start, got[i].End)
		}
	}
}

func TestARepeatingEventKeepsItsLocalTimeAcrossDaylightSaving(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone database on this machine")
	}
	e := Event{
		Title: "Stand-up", TimeZone: "America/New_York", Recurrence: "FREQ=WEEKLY",
		Start: time.Date(2026, 3, 2, 9, 0, 0, 0, ny), End: time.Date(2026, 3, 2, 9, 30, 0, 0, ny),
	}
	got, err := e.Occurrences(utc(2026, 3, 1, 0, 0), utc(2026, 3, 17, 0, 0))
	if err != nil || len(got) != 3 {
		t.Fatalf("%d %v", len(got), err)
	}
	// 9:00 in New York is 14:00 UTC before the change (March 8) and 13:00 after it
	hours := []int{got[0].Start.UTC().Hour(), got[1].Start.UTC().Hour(), got[2].Start.UTC().Hour()}
	if hours[0] != 14 || hours[1] != 13 || hours[2] != 13 {
		t.Errorf("the local 9:00 is kept: %v", hours)
	}
}

func TestAnOccurrenceThatStartedBeforeTheRangeButStillRunsIsListed(t *testing.T) {
	e := Event{Title: "Plantão", Recurrence: "FREQ=DAILY", Start: utc(2026, 1, 5, 22, 0), End: utc(2026, 1, 6, 6, 0)}
	got, _ := e.Occurrences(utc(2026, 1, 7, 0, 0), utc(2026, 1, 7, 12, 0))
	if len(got) != 1 || !got[0].Start.Equal(utc(2026, 1, 6, 22, 0)) {
		t.Errorf("the night that began on the 6th is still on at midnight of the 7th: %+v", got)
	}
}

func TestABoundlessRuleIsCapped(t *testing.T) {
	e := Event{Title: "Ping", Recurrence: "FREQ=MINUTELY", Start: utc(2026, 1, 1, 0, 0), End: utc(2026, 1, 1, 0, 1)}
	got, err := e.Occurrences(utc(2026, 1, 1, 0, 0), utc(2026, 12, 31, 0, 0))
	if err != nil || len(got) != MaxOccurrences {
		t.Errorf("%d occurrences, %v", len(got), err)
	}
}

func TestExpandMergesEventsSortedAndReplacesAChangedOccurrence(t *testing.T) {
	weekly := meeting()
	weekly.Recurrence = "FREQ=WEEKLY;COUNT=3"
	moved := Event{ // the second Monday, moved to the afternoon
		UID: "u1", Title: "Reunião (adiada)", RecurrenceID: utc(2026, 1, 12, 10, 0),
		Start: utc(2026, 1, 12, 15, 0), End: utc(2026, 1, 12, 16, 0),
	}
	lunch := Event{UID: "u2", Title: "Almoço", Start: utc(2026, 1, 12, 12, 0), End: utc(2026, 1, 12, 13, 0)}
	cancelled := Event{UID: "u3", Title: "Cancelada", Status: Cancelled, Start: utc(2026, 1, 12, 8, 0), End: utc(2026, 1, 12, 9, 0)}

	got, err := Expand([]Event{moved, weekly, lunch, cancelled}, utc(2026, 1, 1, 0, 0), utc(2026, 2, 1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, o := range got {
		titles = append(titles, o.Start.Format("02 15h")+" "+o.Event.Title)
	}
	want := "05 10h Reunião|12 12h Almoço|12 15h Reunião (adiada)|19 10h Reunião"
	if strings.Join(titles, "|") != want {
		t.Errorf("%s\nwant %s", strings.Join(titles, "|"), want)
	}
	if !got[2].Changed || got[0].Changed {
		t.Error("only the override is marked as changed")
	}
}

func TestValidateReportsEveryProblem(t *testing.T) {
	if err := meeting().Validate(); err != nil {
		t.Fatal(err)
	}
	bad := Event{Title: " ", End: utc(2026, 1, 1, 0, 0), TimeZone: "Mars/Olympus", Recurrence: "FREQ=SOMETIMES", Reminders: []time.Duration{-time.Minute}}
	err := bad.Validate()
	if err == nil {
		t.Fatal("must be invalid")
	}
	for _, want := range []string{"title is required", "start is required", "unknown time zone", "reminder"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("lacks %q: %v", want, err)
		}
	}
	backwards := meeting()
	backwards.End = backwards.Start.Add(-time.Hour)
	if backwards.Validate() == nil {
		t.Error("an event cannot end before it starts")
	}
	badRule := meeting()
	badRule.Recurrence = "FREQ=SOMETIMES"
	if err := badRule.Validate(); err == nil || !strings.Contains(err.Error(), "recurrence") {
		t.Errorf("a bad rule: %v", err)
	}
	if _, err := badRule.Occurrences(utc(2026, 1, 1, 0, 0), utc(2026, 2, 1, 0, 0)); err == nil {
		t.Error("expanding a bad rule is an error")
	}
	if _, err := Expand([]Event{badRule}, utc(2026, 1, 1, 0, 0), utc(2026, 2, 1, 0, 0)); err == nil {
		t.Error("so is Expand")
	}
}

func TestRemindersAreTimesBeforeTheStart(t *testing.T) {
	e := meeting()
	e.Reminders = []time.Duration{15 * time.Minute, 24 * time.Hour}
	occ, _ := e.Occurrences(utc(2026, 1, 5, 0, 0), utc(2026, 1, 6, 0, 0))
	got := occ[0].ReminderTimes()
	if len(got) != 2 || !got[0].Equal(utc(2026, 1, 5, 9, 45)) || !got[1].Equal(utc(2026, 1, 4, 10, 0)) {
		t.Errorf("%v", got)
	}
	if e.Duration() != time.Hour || e.Recurring() {
		t.Error("Duration and Recurring")
	}
}
