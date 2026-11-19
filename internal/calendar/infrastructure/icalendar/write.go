package icalendar

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/emersion/go-ical"

	"go-pocketbook-base/internal/calendar/domain"
)

// ErrNothingToWrite is returned when there are no events: an iCalendar file with none is not valid.
var ErrNothingToWrite = errors.New("icalendar: there are no events to write")

// ProdID identifies this software in the files it writes.
const ProdID = "-//go-pocketbook-base//Calendar//EN"

// Write writes events as an iCalendar file: what to export, to send by e-mail or to
// put on a server. now is the DTSTAMP of the events (the moment of writing).
//
// Times with a TZID keep their zone (a repeating event must keep its local time
// across daylight saving). The file carries no VTIMEZONE: the IANA name is enough
// for the calendars that matter, and one that does not know it shows the event in
// UTC.
func Write(w io.Writer, events []domain.Event, now time.Time) error {
	if len(events) == 0 {
		return ErrNothingToWrite
	}
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, ProdID)
	cal.Props.SetText(ical.PropCalendarScale, "GREGORIAN")
	for _, e := range events {
		if err := e.Validate(); err != nil {
			return err
		}
		cal.Children = append(cal.Children, componentOf(e, now))
	}
	return ical.NewEncoder(w).Encode(cal)
}

func componentOf(e domain.Event, now time.Time) *ical.Component {
	c := ical.NewComponent(ical.CompEvent)
	c.Props.SetText(ical.PropUID, e.UID)
	c.Props.SetDateTime(ical.PropDateTimeStamp, now.UTC())
	setTime(c, ical.PropDateTimeStart, e.Start, e)
	if !e.End.Equal(e.Start) || e.AllDay {
		setTime(c, ical.PropDateTimeEnd, e.End, e)
	}
	c.Props.SetText(ical.PropSummary, e.Title)
	if e.Description != "" {
		c.Props.SetText(ical.PropDescription, e.Description)
	}
	if e.Location != "" {
		c.Props.SetText(ical.PropLocation, e.Location)
	}
	if e.Recurring() {
		rule := ical.NewProp(ical.PropRecurrenceRule)
		rule.Value = e.Recurrence
		c.Props.Set(rule)
	}
	for _, x := range e.Exceptions {
		p := ical.NewProp(ical.PropExceptionDates)
		stamp(p, x, e)
		c.Props.Add(p)
	}
	if !e.RecurrenceID.IsZero() {
		p := ical.NewProp(ical.PropRecurrenceID)
		stamp(p, e.RecurrenceID, e)
		c.Props.Set(p)
	}
	if e.Status != "" && e.Status != domain.Confirmed {
		c.Props.SetText(ical.PropStatus, map[domain.Status]string{domain.Tentative: "TENTATIVE", domain.Cancelled: "CANCELLED"}[e.Status])
	}
	if e.Sequence > 0 {
		seq := ical.NewProp(ical.PropSequence)
		seq.Value = fmt.Sprint(e.Sequence)
		c.Props.Set(seq)
	}
	if e.Organizer.Email != "" {
		c.Props.Set(personProp(ical.PropOrganizer, e.Organizer, false))
	}
	for _, a := range e.Attendees {
		c.Props.Add(personProp(ical.PropAttendee, a, true))
	}
	for _, before := range e.Reminders {
		c.Children = append(c.Children, alarm(e.Title, before))
	}
	return c
}

func setTime(c *ical.Component, name string, t time.Time, e domain.Event) {
	p := ical.NewProp(name)
	stamp(p, t, e)
	c.Props.Set(p)
}

// stamp puts a time in a property: a date for an all-day event, a time in the
// event's zone when it has one, UTC otherwise.
func stamp(p *ical.Prop, t time.Time, e domain.Event) {
	switch {
	case e.AllDay:
		p.SetDate(t)
	case e.TimeZone != "":
		if loc, err := time.LoadLocation(e.TimeZone); err == nil {
			t = t.In(loc)
		}
		p.SetDateTime(t)
	default:
		p.SetDateTime(t.UTC())
	}
}

func personProp(name string, a domain.Attendee, withResponse bool) *ical.Prop {
	p := ical.NewProp(name)
	p.Value = "mailto:" + a.Email
	if a.Name != "" {
		p.Params.Set("CN", a.Name)
	}
	if withResponse {
		parts := map[domain.Response]string{
			domain.Accepted: "ACCEPTED", domain.Declined: "DECLINED", domain.Maybe: "TENTATIVE", domain.NeedsAction: "NEEDS-ACTION",
		}
		if v := parts[a.Response]; v != "" {
			p.Params.Set("PARTSTAT", v)
		}
	}
	return p
}

func alarm(title string, before time.Duration) *ical.Component {
	a := ical.NewComponent(ical.CompAlarm)
	a.Props.SetText(ical.PropAction, "DISPLAY")
	a.Props.SetText(ical.PropDescription, title)
	trigger := ical.NewProp(ical.PropTrigger)
	trigger.SetDuration(-before)
	a.Props.Set(trigger)
	return a
}
