package domain

import "time"

// Backend is the port to an online calendar service (Outlook.com, Google Calendar): what the
// synchronization reads. The events come as the occurrences of a window, one by one (the service
// expands the repeating ones), so the reader keeps them as single events.
type Backend interface {
	// Calendars lists the calendars of the account; each Calendar.ID is the service's own.
	Calendars() ([]Calendar, error)
	// Events lists the events of a calendar (by its service id) that overlap [from, to). Each one
	// has its UID (the service's id) and no CalendarID.
	Events(calendarID string, from, to time.Time) ([]Event, error)
}
