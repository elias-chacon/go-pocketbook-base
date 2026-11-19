package icalendar

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
)

func crlf(s string) string { return strings.ReplaceAll(strings.TrimLeft(s, "\n"), "\n", "\r\n") }

func utc(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.UTC)
}

// outlook is a file the way Outlook writes one: Windows time zone names, a
// weekly meeting with an exception, a reminder, attendees, folded lines.
const outlook = `
BEGIN:VCALENDAR
PRODID:-//Microsoft Corporation//Outlook 16.0 MIMEDIR//EN
VERSION:2.0
BEGIN:VEVENT
UID:040000008200E00074C5B7101A82E00800000000
DTSTART;TZID=E. South America Standard Time:20260105T090000
DTEND;TZID=E. South America Standard Time:20260105T093000
RRULE:FREQ=WEEKLY;BYDAY=MO;COUNT=6
EXDATE;TZID=E. South America Standard Time:20260112T090000
SUMMARY:Daily do time
DESCRIPTION:Pauta:\n- bloqueios\n- entregas\, prazos e riscos que precisam
  de atenção
LOCATION:Sala 3
STATUS:TENTATIVE
SEQUENCE:2
ORGANIZER;CN="Ana Souza":mailto:ana@example.org
ATTENDEE;CN=Eu;PARTSTAT=ACCEPTED:mailto:eu@example.org
ATTENDEE;CN=Bia;PARTSTAT=NEEDS-ACTION:mailto:bia@example.org
BEGIN:VALARM
TRIGGER:-PT15M
ACTION:DISPLAY
DESCRIPTION:Reminder
END:VALARM
END:VEVENT
END:VCALENDAR
`

func TestAnOutlookMeetingIsReadWithItsZoneRecurrenceAndPeople(t *testing.T) {
	events, warnings, err := Parse(strings.NewReader(crlf(outlook)))
	if err != nil || len(events) != 1 || len(warnings) != 0 {
		t.Fatalf("%d events, warnings=%v, %v", len(events), warnings, err)
	}
	e := events[0]
	if e.Title != "Daily do time" || e.Location != "Sala 3" || e.TimeZone != "America/Sao_Paulo" {
		t.Errorf("%+v", e)
	}
	if !strings.Contains(e.Description, "- bloqueios") || !strings.Contains(e.Description, "entregas, prazos e riscos que precisam de atenção") {
		t.Errorf("escapes and folded lines: %q", e.Description)
	}
	// 9:00 in São Paulo (UTC-3) is 12:00 UTC
	if !e.Start.Equal(utc(2026, 1, 5, 12, 0)) || e.Duration() != 30*time.Minute || e.AllDay {
		t.Errorf("%v %v", e.Start, e.Duration())
	}
	if e.Recurrence != "FREQ=WEEKLY;BYDAY=MO;COUNT=6" || len(e.Exceptions) != 1 || !e.Exceptions[0].Equal(utc(2026, 1, 12, 12, 0)) {
		t.Errorf("recurrence: %q %v", e.Recurrence, e.Exceptions)
	}
	if e.Status != domain.Tentative || e.Sequence != 2 || len(e.Reminders) != 1 || e.Reminders[0] != 15*time.Minute {
		t.Errorf("%v %d %v", e.Status, e.Sequence, e.Reminders)
	}
	checkPeople(t, e)

	occ, err := e.Occurrences(utc(2026, 1, 1, 0, 0), utc(2026, 3, 1, 0, 0))
	if err != nil || len(occ) != 5 {
		t.Errorf("6 weekly meetings minus the exception: %d %v", len(occ), err)
	}
}

const googleStyle = `
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Google Inc//Google Calendar 70.9054//EN
BEGIN:VEVENT
DTSTART;VALUE=DATE:20260320
DTEND;VALUE=DATE:20260323
UID:ferias@google.com
SUMMARY:Férias
END:VEVENT
BEGIN:VEVENT
DTSTART:20260310T130000Z
DURATION:PT1H30M
UID:call@google.com
SUMMARY:Chamada
END:VEVENT
BEGIN:VEVENT
DTSTART;TZID=Europe/Paris:20260315T100000
UID:call2@google.com
SUMMARY:Marco
RECURRENCE-ID;TZID=Europe/Paris:20260308T100000
END:VEVENT
BEGIN:VEVENT
DTSTART:20260401T100000Z
UID:sem-titulo@google.com
END:VEVENT
END:VCALENDAR
`

func TestAGoogleStyleFileGivesAllDayEventsDurationsAndOverrides(t *testing.T) {
	events, warnings, err := Parse(strings.NewReader(crlf(googleStyle)))
	if err != nil || len(events) != 4 || len(warnings) != 0 {
		t.Fatalf("%d %v %v", len(events), warnings, err)
	}
	trip := events[0]
	if !trip.AllDay || !trip.Start.Equal(utc(2026, 3, 20, 0, 0)) || !trip.End.Equal(utc(2026, 3, 23, 0, 0)) || trip.TimeZone != "" {
		t.Errorf("an all-day event: %+v", trip)
	}
	if call := events[1]; call.Duration() != 90*time.Minute || call.TimeZone != "" || !call.Start.Equal(utc(2026, 3, 10, 13, 0)) {
		t.Errorf("DURATION: %+v", call)
	}
	over := events[2]
	if over.RecurrenceID.IsZero() || over.TimeZone != "Europe/Paris" || over.Duration() != 0 {
		t.Errorf("an override with no end: %+v", over)
	}
	if events[3].Title != "(sem título)" {
		t.Errorf("an event without a title still shows: %q", events[3].Title)
	}
}

func TestProblemsBecomeWarningsAndDoNotLoseTheOtherEvents(t *testing.T) {
	raw := `
BEGIN:VCALENDAR
VERSION:2.0
PRODID:x
BEGIN:VEVENT
UID:a
DTSTART;TZID=Planeta Marte:20260105T090000
SUMMARY:Zona desconhecida
END:VEVENT
BEGIN:VEVENT
UID:b
SUMMARY:Sem início
END:VEVENT
BEGIN:VEVENT
SUMMARY:Sem UID
DTSTART:20260105T090000Z
DTEND:lixo
END:VEVENT
BEGIN:VEVENT
UID:c
DTSTART:20260105T090000Z
DTEND:20260105T080000Z
SUMMARY:Termina antes
END:VEVENT
BEGIN:VEVENT
UID:d
DTSTART:20260105T090000Z
RRULE:FREQ=SOMETIMES
SUMMARY:Regra ruim
END:VEVENT
BEGIN:VTODO
UID:t
SUMMARY:Uma tarefa não é evento
END:VTODO
END:VCALENDAR
`
	events, warnings, err := Parse(strings.NewReader(crlf(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Title != "Zona desconhecida" || events[1].Title != "Sem UID" {
		t.Fatalf("only the two that can be understood: %+v", events)
	}
	if !events[0].Start.Equal(utc(2026, 1, 5, 9, 0)) || events[1].UID != "imported-3" {
		t.Errorf("UTC fallback and a made-up UID: %v %q", events[0].Start, events[1].UID)
	}
	all := strings.Join(warnings, "\n")
	for _, want := range []string{"Planeta Marte", "no start", "no UID", "end cannot be read", "end is before", "recurrence"} {
		if !strings.Contains(all, want) {
			t.Errorf("a warning about %q is missing:\n%s", want, all)
		}
	}
	if _, _, err := Parse(strings.NewReader("this is not a calendar")); err == nil {
		t.Error("garbage is an error")
	}
}

func TestZoneNamesOfEveryOrigin(t *testing.T) {
	cases := map[string]bool{
		"America/Sao_Paulo":                                               true,
		"E. South America Standard Time":                                  true,
		"/mozilla.org/20050126_1/Europe/Paris":                            true,
		"/freeassociation.sourceforge.net/America/Argentina/Buenos_Aires": true,
		`"Pacific Standard Time"`:                                         true,
		"":                                                                true,
		"Nowhere Standard Time":                                           false,
	}
	for name, want := range cases {
		if _, ok := resolveZone(name); ok != want {
			t.Errorf("%q: %v", name, ok)
		}
	}
	if loc, _ := resolveZone("Eastern Standard Time"); loc.String() != "America/New_York" {
		t.Error(loc)
	}
}

func sample() []domain.Event {
	ny, _ := time.LoadLocation("America/New_York")
	return []domain.Event{
		{
			UID: "weekly@x", Title: "Stand-up; com vírgula, e quebra\nde linha", Description: "Pauta:\n- um\n- dois", Location: "Sala, 3",
			TimeZone: "America/New_York", Start: time.Date(2026, 3, 2, 9, 0, 0, 0, ny), End: time.Date(2026, 3, 2, 9, 30, 0, 0, ny),
			Recurrence: "FREQ=WEEKLY;COUNT=4", Exceptions: []time.Time{time.Date(2026, 3, 9, 9, 0, 0, 0, ny)},
			Reminders: []time.Duration{10 * time.Minute, 24 * time.Hour}, Status: domain.Tentative, Sequence: 3,
			Organizer: domain.Attendee{Name: "Ana", Email: "ana@example.org"},
			Attendees: []domain.Attendee{{Name: "Eu", Email: "eu@example.org", Response: domain.Accepted}, {Email: "bia@example.org", Response: domain.Declined}},
		},
		{UID: "trip@x", Title: "Viagem", AllDay: true, Start: utc(2026, 5, 1, 0, 0), End: utc(2026, 5, 4, 0, 0)},
		{UID: "once@x", Title: "Dentista", Start: utc(2026, 4, 1, 17, 0), End: utc(2026, 4, 1, 18, 0)},
		{UID: "weekly@x", Title: "Stand-up (adiado)", RecurrenceID: utc(2026, 3, 16, 14, 0), TimeZone: "", Start: utc(2026, 3, 16, 18, 0), End: utc(2026, 3, 16, 18, 30)},
		{UID: "instant@x", Title: "Aviso", Start: utc(2026, 4, 2, 8, 0), End: utc(2026, 4, 2, 8, 0)},
	}
}

func TestEventsSurviveWriteAndParse(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sample(), utc(2026, 1, 1, 0, 0)); err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	for _, want := range []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:" + ProdID, "DTSTART;TZID=America/New_York:20260302T090000", "DTSTART;VALUE=DATE:20260501", "DTSTAMP:20260101T000000Z", "RRULE:FREQ=WEEKLY;COUNT=4", "TRIGGER", "MAILTO:", "mailto:ana@example.org"} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
			t.Errorf("the file lacks %q", want)
		}
	}

	got, warnings, err := Parse(strings.NewReader(text))
	if err != nil || len(warnings) != 0 || len(got) != len(sample()) {
		t.Fatalf("%d events, warnings=%v, %v\n%s", len(got), warnings, err, text)
	}
	want := sample()
	for i := range want {
		g, w := got[i], want[i]
		if g.UID != w.UID || g.Title != w.Title || g.Description != w.Description || g.Location != w.Location ||
			!g.Start.Equal(w.Start) || !g.End.Equal(w.End) || g.AllDay != w.AllDay || g.TimeZone != w.TimeZone ||
			g.Recurrence != w.Recurrence || g.Sequence != w.Sequence || !g.RecurrenceID.Equal(w.RecurrenceID) {
			t.Errorf("event %d changed:\n got %+v\nwant %+v", i, g, w)
		}
	}
	first := got[0]
	if len(first.Exceptions) != 1 || !first.Exceptions[0].Equal(want[0].Exceptions[0]) {
		t.Errorf("exceptions: %v", first.Exceptions)
	}
	if len(first.Reminders) != 2 || first.Reminders[0] != 10*time.Minute || first.Reminders[1] != 24*time.Hour {
		t.Errorf("reminders: %v", first.Reminders)
	}
	if first.Status != domain.Tentative || first.Organizer.Email != "ana@example.org" || len(first.Attendees) != 2 ||
		first.Attendees[0].Response != domain.Accepted || first.Attendees[1].Response != domain.Declined {
		t.Errorf("%v %+v %+v", first.Status, first.Organizer, first.Attendees)
	}
}

func TestWriteRefusesAnInvalidEvent(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, []domain.Event{{Title: ""}}, time.Now()); err == nil {
		t.Error("an event with no title or start cannot be exported")
	}
	if err := Write(&buf, nil, time.Now()); err != ErrNothingToWrite {
		t.Errorf("an iCalendar file with no events is not valid: %v", err)
	}
}

// checkPeople: the organizer and the two attendees of the Outlook sample, with their answers.
func checkPeople(t *testing.T, e domain.Event) {
	t.Helper()
	if e.Organizer.Name != "Ana Souza" || e.Organizer.Email != "ana@example.org" {
		t.Errorf("organizer: %+v", e.Organizer)
	}
	if len(e.Attendees) != 2 || e.Attendees[0].Response != domain.Accepted || e.Attendees[1].Response != domain.NeedsAction {
		t.Errorf("attendees: %+v", e.Attendees)
	}
}
