// Package icalendar reads and writes iCalendar files (RFC 5545, ".ics"): the
// format calendars are exchanged in (Outlook, Google, Apple, e-mail invitations).
// Parse turns the events of a file into the domain's Events; Write goes the other
// way. It uses github.com/emersion/go-ical for the syntax (lines, folding,
// escaping) and handles the meaning here: time zones (including the Windows names
// Outlook writes), all-day events, recurrence, reminders and attendees.
package icalendar

import (
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-ical"

	"go-pocketbook-base/internal/calendar/domain"
)

// Parse reads the events of an iCalendar file. A problem with one event (a date
// that cannot be read, an unknown time zone) does not lose the others: it becomes
// a warning, and the event is skipped only when it cannot be understood at all.
func Parse(r io.Reader) ([]domain.Event, []string, error) {
	cal, err := ical.NewDecoder(r).Decode()
	if err != nil {
		return nil, nil, fmt.Errorf("icalendar: %w", err)
	}
	var events []domain.Event
	var warnings []string
	for i, comp := range cal.Children {
		if comp.Name != ical.CompEvent {
			continue
		}
		e, warns, err := eventOf(comp, i+1)
		warnings = append(warnings, warns...)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("event %d skipped: %v", i+1, err))
			continue
		}
		events = append(events, e)
	}
	return events, warnings, nil
}

// zonedTime is a date or date-time property read in the zone it names.
type zonedTime struct {
	t      time.Time
	zone   string // the IANA name, "" for UTC and dates
	allDay bool
}

// readTime reads a DTSTART-like property: UTC ("...Z"), with a TZID, floating or a date.
func readTime(prop *ical.Prop) (zonedTime, string, error) {
	loc, known := resolveZone(prop.Params.Get(ical.PropTimezoneID))
	warning := ""
	if !known {
		warning = fmt.Sprintf("the time zone %q is not known: UTC was used", prop.Params.Get(ical.PropTimezoneID))
	}
	// go-ical would load the TZID itself and fail on a Windows name: give it the resolved location
	clone := *prop
	clone.Params = ical.Params{}
	for k, v := range prop.Params {
		if k != ical.PropTimezoneID {
			clone.Params[k] = v
		}
	}
	t, err := clone.DateTime(loc)
	if err != nil {
		return zonedTime{}, warning, err
	}
	allDay := clone.ValueType() == ical.ValueDate || (clone.ValueType() == ical.ValueDefault && len(clone.Value) == 8)
	zt := zonedTime{t: t, allDay: allDay}
	if allDay {
		zt.t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) // a date is a day, not an instant
	} else if loc != time.UTC && !strings.HasSuffix(prop.Value, "Z") {
		zt.zone = loc.String()
	}
	return zt, warning, nil
}

func eventOf(comp *ical.Component, n int) (domain.Event, []string, error) {
	var warnings []string
	var e domain.Event
	e.UID = textOf(comp, ical.PropUID)
	if e.UID == "" {
		e.UID = fmt.Sprintf("imported-%d", n)
		warnings = append(warnings, fmt.Sprintf("event %d has no UID: %q was used", n, e.UID))
	}
	e.Title, e.Description, e.Location = textOf(comp, ical.PropSummary), textOf(comp, ical.PropDescription), textOf(comp, ical.PropLocation)

	startProp := comp.Props.Get(ical.PropDateTimeStart)
	if startProp == nil {
		return e, warnings, fmt.Errorf("it has no start")
	}
	start, warn, err := readTime(startProp)
	if warn != "" {
		warnings = append(warnings, warn)
	}
	if err != nil {
		return e, warnings, fmt.Errorf("the start cannot be read: %w", err)
	}
	e.Start, e.AllDay, e.TimeZone = start.t, start.allDay, start.zone

	e.End = endOf(comp, start, &warnings)
	if e.Title == "" {
		e.Title = "(sem título)"
	}
	e.Recurrence = recurrenceOf(comp)
	e.Exceptions = exceptionsOf(comp, &warnings)
	if p := comp.Props.Get(ical.PropRecurrenceID); p != nil {
		if rid, _, err := readTime(p); err == nil {
			e.RecurrenceID = rid.t
		}
	}
	e.Status = statusOf(textOf(comp, ical.PropStatus))
	if p := comp.Props.Get(ical.PropSequence); p != nil {
		e.Sequence, _ = p.Int()
	}
	e.Organizer = personOf(comp.Props.Get(ical.PropOrganizer))
	for _, p := range comp.Props.Values(ical.PropAttendee) {
		p := p
		e.Attendees = append(e.Attendees, personOf(&p))
	}
	e.Reminders = remindersOf(comp, e.Start)
	if err := e.Validate(); err != nil {
		return e, warnings, err
	}
	return e, warnings, nil
}

func textOf(comp *ical.Component, name string) string {
	s, _ := comp.Props.Text(name)
	return strings.TrimSpace(s)
}

// endOf is DTEND, or the start plus DURATION, or the whole day for an all-day
// event, or the start itself.
func endOf(comp *ical.Component, start zonedTime, warnings *[]string) time.Time {
	if p := comp.Props.Get(ical.PropDateTimeEnd); p != nil {
		if end, _, err := readTime(p); err == nil {
			return end.t
		}
		*warnings = append(*warnings, "the end cannot be read: the event ends where it starts")
		return start.t
	}
	if p := comp.Props.Get(ical.PropDuration); p != nil {
		if d, err := p.Duration(); err == nil {
			return start.t.Add(d)
		}
	}
	if start.allDay {
		return start.t.AddDate(0, 0, 1)
	}
	return start.t
}

func recurrenceOf(comp *ical.Component) string {
	if p := comp.Props.Get(ical.PropRecurrenceRule); p != nil {
		return strings.TrimSpace(p.Value)
	}
	return ""
}

// exceptionsOf reads every EXDATE (each may list several values).
func exceptionsOf(comp *ical.Component, warnings *[]string) []time.Time {
	var out []time.Time
	for _, p := range comp.Props.Values(ical.PropExceptionDates) {
		for _, v := range strings.Split(p.Value, ",") {
			one := p
			one.Value = strings.TrimSpace(v)
			x, _, err := readTime(&one)
			if err != nil {
				*warnings = append(*warnings, fmt.Sprintf("an exception date %q cannot be read", v))
				continue
			}
			out = append(out, x.t)
		}
	}
	return out
}

func statusOf(s string) domain.Status {
	switch strings.ToUpper(s) {
	case "TENTATIVE":
		return domain.Tentative
	case "CANCELLED":
		return domain.Cancelled
	}
	return domain.Confirmed
}

// personOf reads ORGANIZER and ATTENDEE: a "mailto:" address with CN and PARTSTAT.
func personOf(p *ical.Prop) domain.Attendee {
	if p == nil {
		return domain.Attendee{}
	}
	email := strings.TrimSpace(p.Value)
	if i := strings.Index(strings.ToLower(email), "mailto:"); i == 0 {
		email = email[len("mailto:"):]
	}
	if a, err := mail.ParseAddress(email); err == nil {
		email = a.Address
	}
	return domain.Attendee{Name: strings.Trim(p.Params.Get("CN"), `"`), Email: email, Response: responseOf(p.Params.Get("PARTSTAT"))}
}

func responseOf(s string) domain.Response {
	switch strings.ToUpper(s) {
	case "ACCEPTED":
		return domain.Accepted
	case "DECLINED":
		return domain.Declined
	case "TENTATIVE":
		return domain.Maybe
	}
	return domain.NeedsAction
}

// remindersOf reads the VALARMs: a trigger before the start ("-PT15M") or at a given instant.
func remindersOf(comp *ical.Component, start time.Time) []time.Duration {
	var out []time.Duration
	for _, child := range comp.Children {
		if child.Name != ical.CompAlarm {
			continue
		}
		p := child.Props.Get(ical.PropTrigger)
		if p == nil {
			continue
		}
		var before time.Duration
		if p.ValueType() == ical.ValueDateTime {
			at, _, err := readTime(p)
			if err != nil {
				continue
			}
			before = start.Sub(at.t)
		} else {
			d, err := p.Duration()
			if err != nil {
				continue
			}
			before = -d
		}
		if before >= 0 {
			out = append(out, before)
		}
	}
	return out
}
