// Package domain is the heart of the Calendar context: events, their recurrence
// and reminders, and the rules for finding the occurrences of a day, a week or a
// month. It knows nothing of iCalendar files, servers or the screen.
//
// Ubiquitous language:
//
//   - Event: something that happens at a time, once or repeating.
//   - Occurrence: one appearance of an event (a repeating event has many).
//   - Calendar: a collection of events with a name (a person can have several).
//   - Reminder: how long before the start the person wants to be told.
//
// Recurrence uses github.com/teambition/rrule-go, a small library without
// dependencies, because the rules of RRULE (RFC 5545) are long and exact.
package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// Status is how sure the event is.
type Status string

// The statuses of an event.
const (
	Confirmed Status = "confirmed"
	Tentative Status = "tentative"
	Cancelled Status = "cancelled"
)

// Response is what an attendee answered to an invitation.
type Response string

// The answers to an invitation.
const (
	NeedsAction Response = "needs-action"
	Accepted    Response = "accepted"
	Declined    Response = "declined"
	Maybe       Response = "tentative"
)

// Attendee is someone invited to an event.
type Attendee struct {
	Name     string
	Email    string
	Response Response
}

// Calendar is a named collection of events.
type Calendar struct {
	ID        string
	AccountID string
	Name      string
	ReadOnly  bool // an imported or subscribed calendar cannot be changed
}

// Event is something that happens at a time.
type Event struct {
	ID         string // local id
	CalendarID string
	RemoteID   string

	UID         string // identifies the event across systems (iCalendar UID)
	Title       string
	Description string
	Location    string

	// Start and End are instants; End is exclusive. An all-day event has its Start
	// and End at midnight UTC of the first day and of the day after the last.
	Start, End time.Time
	AllDay     bool
	// TimeZone is the IANA name the event was written in ("" for UTC and all-day).
	// A repeating event keeps its local time across daylight saving changes.
	TimeZone string

	// Recurrence is an RRULE (RFC 5545) without the "RRULE:" prefix, e.g.
	// "FREQ=WEEKLY;BYDAY=MO,WE". Exceptions are the starts of the occurrences
	// that were deleted (EXDATE).
	Recurrence string
	Exceptions []time.Time
	// RecurrenceID, when not zero, makes this event the changed version of one
	// occurrence of the repeating event with the same UID (the occurrence that
	// started at RecurrenceID).
	RecurrenceID time.Time

	Reminders []time.Duration // before the start
	Organizer Attendee
	Attendees []Attendee
	Status    Status
	Sequence  int // grows with every change, so a server can tell the newer
}

// Recurring reports whether the event repeats.
func (e Event) Recurring() bool { return e.Recurrence != "" }

// Duration is how long the event lasts.
func (e Event) Duration() time.Duration { return e.End.Sub(e.Start) }

// Location of the event's clock: its time zone, or UTC.
func (e Event) location() (*time.Location, error) {
	if e.TimeZone == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(e.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("calendar: unknown time zone %q", e.TimeZone)
	}
	return loc, nil
}

// Validate checks the rules of an event. It returns every problem at once.
func (e Event) Validate() error {
	var problems []string
	if strings.TrimSpace(e.Title) == "" {
		problems = append(problems, "title is required")
	}
	if e.Start.IsZero() {
		problems = append(problems, "start is required")
	}
	if e.End.Before(e.Start) {
		problems = append(problems, "the end is before the start")
	}
	if _, err := e.location(); err != nil {
		problems = append(problems, err.Error())
	}
	if e.Recurring() {
		if _, err := e.rule(); err != nil {
			problems = append(problems, fmt.Sprintf("recurrence %q: %v", e.Recurrence, err))
		}
	}
	for _, r := range e.Reminders {
		if r < 0 {
			problems = append(problems, "a reminder is a time before the start, not after")
		}
	}
	if len(problems) > 0 {
		return errors.New("calendar: " + strings.Join(problems, "; "))
	}
	return nil
}

// rule is the event's recurrence as a rule starting at the event's start.
func (e Event) rule() (*rrule.RRule, error) {
	loc, err := e.location()
	if err != nil {
		return nil, err
	}
	opt, err := rrule.StrToROptionInLocation(e.Recurrence, loc)
	if err != nil {
		return nil, err
	}
	opt.Dtstart = e.Start.In(loc)
	return rrule.NewRRule(*opt)
}

// Occurrence is one appearance of an event.
type Occurrence struct {
	Event      Event
	Start, End time.Time
	// Changed is true for an occurrence that was edited on its own (it came from an
	// override with a RecurrenceID).
	Changed bool
}

// MaxOccurrences bounds the expansion of a repeating event, so a rule such as
// "every minute, forever" cannot fill the reader's memory.
const MaxOccurrences = 2000

// Occurrences are the appearances of one event that overlap [from, to), by start.
// A single event gives itself (when it overlaps); a repeating one gives each
// occurrence, without the exceptions.
func (e Event) Occurrences(from, to time.Time) ([]Occurrence, error) {
	if !e.Recurring() {
		if e.overlaps(e.Start, e.End, from, to) {
			return []Occurrence{{Event: e, Start: e.Start, End: e.End}}, nil
		}
		return nil, nil
	}
	r, err := e.rule()
	if err != nil {
		return nil, err
	}
	skipped := map[int64]bool{}
	for _, x := range e.Exceptions {
		skipped[x.Unix()] = true
	}
	// an occurrence that started before from may still be running: look back one duration
	var out []Occurrence
	for _, start := range r.Between(from.Add(-e.Duration()), to, true) {
		if skipped[start.Unix()] {
			continue
		}
		end := start.Add(e.Duration())
		if !e.overlaps(start, end, from, to) {
			continue
		}
		out = append(out, Occurrence{Event: e, Start: start, End: end})
		if len(out) >= MaxOccurrences {
			break
		}
	}
	return out, nil
}

// overlaps reports whether [start, end) touches [from, to); an event with no
// length counts at its instant.
func (e Event) overlaps(start, end, from, to time.Time) bool {
	if !end.After(start) {
		return !start.Before(from) && start.Before(to)
	}
	return start.Before(to) && end.After(from)
}

// Expand gives every occurrence of the events that overlaps [from, to), sorted by
// start. An event with a RecurrenceID replaces that occurrence of the repeating
// event with the same UID; a cancelled event shows nowhere.
func Expand(events []Event, from, to time.Time) ([]Occurrence, error) {
	replaced := map[string]map[int64]bool{} // UID -> starts replaced by an override
	for _, e := range events {
		if !e.RecurrenceID.IsZero() {
			if replaced[e.UID] == nil {
				replaced[e.UID] = map[int64]bool{}
			}
			replaced[e.UID][e.RecurrenceID.Unix()] = true
		}
	}

	var out []Occurrence
	for _, e := range events {
		if e.Status == Cancelled {
			continue
		}
		occ, err := e.Occurrences(from, to)
		if err != nil {
			return nil, err
		}
		for _, o := range occ {
			if e.RecurrenceID.IsZero() && replaced[e.UID][o.Start.Unix()] {
				continue // this one was changed: its override is shown instead
			}
			o.Changed = !e.RecurrenceID.IsZero()
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out, nil
}

// ReminderTimes are the instants the person is told about an occurrence.
func (o Occurrence) ReminderTimes() []time.Time {
	out := make([]time.Time, 0, len(o.Event.Reminders))
	for _, before := range o.Event.Reminders {
		out = append(out, o.Start.Add(-before))
	}
	return out
}
