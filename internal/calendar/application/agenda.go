// Package application holds the use cases of the Calendar context: what the calendar
// screens ask for (the marks of a month, the events of a day, add, change, delete).
// The screens read only from the store.
package application

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
)

// DefaultCalendarID is the id of the calendar a new install starts with.
const DefaultCalendarID = "local"

// ErrEmptyTitle is returned when saving an event with no title.
var ErrEmptyTitle = errors.New("calendar: an event needs a title")

// Agenda is the use case of the calendar of one account.
type Agenda struct {
	AccountID string
	Store     domain.EventStore
	// Zone is the IANA name of the person's clock ("" is UTC). Events are written in it,
	// so a repeating one keeps its local time across daylight saving changes.
	Zone string
	// Now is the clock (time.Now when nil).
	Now func() time.Time
}

func (a Agenda) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// Loc is the person's clock.
func (a Agenda) Loc() *time.Location {
	if a.Zone == "" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(a.Zone); err == nil {
		return loc
	}
	return time.UTC
}

// Today is the current day on the person's clock, at midnight.
func (a Agenda) Today() time.Time { return a.dayStart(a.now().In(a.Loc())) }

// dayStart is midnight, on the person's clock, of the calendar date t shows in its own
// location: the screens hand over a date, and a date is not shifted by the zone.
func (a Agenda) dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, a.Loc())
}

// EnsureCalendar makes sure there is a calendar to put events in, creating one
// called name when the account has none.
func (a Agenda) EnsureCalendar(name string) (domain.Calendar, error) {
	cals, err := a.Store.Calendars(a.AccountID)
	if err != nil {
		return domain.Calendar{}, err
	}
	for _, c := range cals {
		if !c.ReadOnly {
			return c, nil
		}
	}
	c := domain.Calendar{ID: DefaultCalendarID, AccountID: a.AccountID, Name: name}
	return c, a.Store.SaveCalendars(a.AccountID, append(cals, c))
}

// occurrences expands the events around a range, on the person's clock.
func (a Agenda) occurrences(from, to time.Time) ([]domain.Occurrence, error) {
	// a day of margin on each side: an all-day event is a date, not an instant
	events, err := a.Store.Between(from.AddDate(0, 0, -1), to.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	return domain.Expand(events, from.AddDate(0, 0, -1), to.AddDate(0, 0, 1))
}

// onDay reports whether an occurrence happens on a day (a midnight on the person's
// clock). An all-day event is a date: it is on that date whatever the zone.
func (a Agenda) onDay(o domain.Occurrence, day time.Time) bool {
	if o.Event.AllDay {
		y, m, d := day.Date()
		date := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		return !o.Start.After(date) && o.End.After(date)
	}
	return (o.Start.Before(day.AddDate(0, 0, 1)) && o.End.After(day)) || o.Start.Equal(day)
}

// Day returns what happens on a day, all-day events first and then by start.
func (a Agenda) Day(day time.Time) ([]domain.Occurrence, error) {
	day = a.dayStart(day)
	occ, err := a.occurrences(day, day.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	var out []domain.Occurrence
	for _, o := range occ {
		if a.onDay(o, day) {
			out = append(out, o)
		}
	}
	sortDay(out)
	return out, nil
}

// Marks tells which days of a month (by day number) have something on them.
func (a Agenda) Marks(year int, month time.Month) (map[int]bool, error) {
	first := time.Date(year, month, 1, 0, 0, 0, 0, a.Loc())
	last := first.AddDate(0, 1, 0)
	occ, err := a.occurrences(first, last)
	if err != nil {
		return nil, err
	}
	marks := map[int]bool{}
	for day := first; day.Before(last); day = day.AddDate(0, 0, 1) {
		for _, o := range occ {
			if a.onDay(o, day) {
				marks[day.Day()] = true
				break
			}
		}
	}
	return marks, nil
}

// Entry is what a person types to make or change an event.
type Entry struct {
	Title, Location, Description string
	// Day is the day of the event (a date; the time of day is ignored).
	Day time.Time
	// Start and End are minutes after midnight; ignored for an all-day event.
	Start, End int
	AllDay     bool
	// Recurrence is an RRULE without the prefix ("FREQ=WEEKLY"); empty for an event that happens once.
	Recurrence string
	// Reminder is how long before the start to remind (0: no reminder).
	Reminder time.Duration
}

// build turns an entry into an event on the person's clock. An end that is not after
// the start makes the event end the same minute (an instant), never before it.
func (a Agenda) build(e domain.Event, in Entry) (domain.Event, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return e, ErrEmptyTitle
	}
	e.Title, e.Location, e.Description = in.Title, strings.TrimSpace(in.Location), in.Description
	e.Recurrence = strings.TrimSpace(in.Recurrence)
	e.Reminders = nil
	if in.Reminder > 0 {
		e.Reminders = []time.Duration{in.Reminder}
	}
	y, m, d := in.Day.Date()
	if in.AllDay {
		e.AllDay, e.TimeZone = true, ""
		e.Start = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		e.End = e.Start.AddDate(0, 0, 1)
		return e, e.Validate()
	}
	if in.End < in.Start {
		in.End = in.Start
	}
	e.AllDay, e.TimeZone = false, a.Zone
	loc := a.Loc()
	e.Start = time.Date(y, m, d, 0, in.Start, 0, 0, loc)
	e.End = time.Date(y, m, d, 0, in.End, 0, 0, loc)
	return e, e.Validate()
}

// Add creates an event in the account's calendar.
func (a Agenda) Add(in Entry) (domain.Event, error) {
	cal, err := a.EnsureCalendar("")
	if err != nil {
		return domain.Event{}, err
	}
	e, err := a.build(domain.Event{CalendarID: cal.ID, Status: domain.Confirmed}, in)
	if err != nil {
		return domain.Event{}, err
	}
	e.UID = newUID(a.now(), e.Title)
	e.ID = e.UID // the caller needs the id to open the event it just made
	return e, a.Store.Save(e)
}

// Change rewrites an event with a new entry.
func (a Agenda) Change(id string, in Entry) error {
	e, err := a.Store.Get(id)
	if err != nil {
		return err
	}
	e, err = a.build(e, in)
	if err != nil {
		return err
	}
	e.Sequence++
	return a.Store.Save(e)
}

// Import keeps events read from an .ics file or address in the account's calendar. An
// event that is already there (same UID) is replaced, so importing a file again does
// not duplicate it. It returns how many events were kept.
func (a Agenda) Import(events []domain.Event) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	cal, err := a.EnsureCalendar("")
	if err != nil {
		return 0, err
	}
	for i := range events {
		events[i].CalendarID = cal.ID
		if events[i].Status == "" {
			events[i].Status = domain.Confirmed
		}
	}
	return len(events), a.Store.Save(events...)
}

// Get returns one event.
func (a Agenda) Get(id string) (domain.Event, error) { return a.Store.Get(id) }

// Delete removes an event.
func (a Agenda) Delete(id string) error { return a.Store.Delete(id) }

// newUID makes an id that is unique across events and across systems: the moment, the
// start of the title and some random bytes.
func newUID(now time.Time, title string) string {
	var salt [4]byte
	_, _ = rand.Read(salt[:])
	return now.UTC().Format("20060102T150405") + "-" + hex.EncodeToString(salt[:]) + "-" + slug(title) + "@pbgo"
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if b.Len() > 12 {
		return b.String()[:12]
	}
	return b.String()
}

// Span returns what happens on each of n consecutive days from first: one list per day,
// all-day events first and then by start. It reads the store once, so a month or a
// week costs one query.
func (a Agenda) Span(first time.Time, n int) ([][]domain.Occurrence, error) {
	first = a.dayStart(first)
	occ, err := a.occurrences(first, first.AddDate(0, 0, n))
	if err != nil {
		return nil, err
	}
	out := make([][]domain.Occurrence, n)
	for i := range out {
		day := first.AddDate(0, 0, i)
		for _, o := range occ {
			if a.onDay(o, day) {
				out[i] = append(out[i], o)
			}
		}
		sortDay(out[i])
	}
	return out, nil
}

func sortDay(list []domain.Occurrence) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Event.AllDay != list[j].Event.AllDay {
			return list[i].Event.AllDay
		}
		return list[i].Start.Before(list[j].Start)
	})
}
