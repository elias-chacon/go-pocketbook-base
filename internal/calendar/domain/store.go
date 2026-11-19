package domain

import (
	"errors"
	"time"
)

// ErrNotFound is returned for an event or calendar that is not in the store.
var ErrNotFound = errors.New("calendar: not found")

// EventStore is the port to the events kept on the reader. The screens read only
// from it (offline-first); the synchronization and the importer write to it.
type EventStore interface {
	// SaveCalendars replaces the calendars of an account with the ones the server
	// has; the events of a calendar that no longer exists go with it.
	SaveCalendars(accountID string, calendars []Calendar) error
	Calendars(accountID string) ([]Calendar, error)

	// Save keeps events, replacing the ones with the same calendar, UID and
	// RecurrenceID (the changed version of one occurrence is its own event). An
	// event with no ID gets one.
	Save(events ...Event) error
	// Between returns the events that can have an occurrence in [from, to): the
	// single ones that overlap it, the repeating ones that start before its end,
	// and the changed occurrences of those (so Expand can apply them). Give calendar
	// ids to look only in those; none looks in every calendar.
	Between(from, to time.Time, calendarIDs ...string) ([]Event, error)
	// Get returns one event.
	Get(id string) (Event, error)
	// Delete removes an event, and with it the changed occurrences of its UID when
	// it is the repeating one.
	Delete(id string) error
}
