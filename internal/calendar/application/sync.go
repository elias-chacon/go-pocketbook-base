package application

import (
	"fmt"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
)

// The window of the synchronization: what the reader keeps of the online calendars.
const (
	SyncPast   = 30 * 24 * time.Hour
	SyncFuture = 365 * 24 * time.Hour
)

// SyncResult is what a synchronization brought.
type SyncResult struct{ Calendars, Events int }

// Sync is the use case "bring the events of the online calendars to the reader": the calendars of
// the account (read only: what changes there is read again, what is changed here is not sent) and
// the events of a window around today. What the service no longer has is removed. The screens read
// from the store, so the calendar works offline with what was brought.
type Sync struct {
	AccountID string // the account in the store: its calendars are replaced by the service's
	Store     domain.EventStore
	Backend   domain.Backend
	Now       func() time.Time // time.Now when nil
}

// Run synchronizes once.
func (s Sync) Run() (SyncResult, error) {
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	from, to := now.Add(-SyncPast), now.Add(SyncFuture)
	remote, err := s.Backend.Calendars()
	if err != nil {
		return SyncResult{}, fmt.Errorf("calendars: %w", err)
	}
	cals := make([]domain.Calendar, len(remote))
	for i, c := range remote {
		cals[i] = domain.Calendar{ID: s.AccountID + ":" + c.ID, AccountID: s.AccountID, Name: c.Name, ReadOnly: true}
	}
	if err := s.Store.SaveCalendars(s.AccountID, cals); err != nil {
		return SyncResult{}, err
	}
	res := SyncResult{Calendars: len(cals)}
	for i, c := range remote {
		n, err := s.syncCalendar(cals[i].ID, c.ID, from, to)
		if err != nil {
			return res, fmt.Errorf("%s: %w", c.Name, err)
		}
		res.Events += n
	}
	return res, nil
}

// syncCalendar replaces the events of the window of one calendar and returns how many it has now.
func (s Sync) syncCalendar(localID, remoteID string, from, to time.Time) (int, error) {
	events, err := s.Backend.Events(remoteID, from, to)
	if err != nil {
		return 0, err
	}
	fresh := map[string]bool{}
	for i := range events {
		events[i].CalendarID = localID
		events[i].RemoteID = events[i].UID
		if events[i].Status == "" {
			events[i].Status = domain.Confirmed
		}
		fresh[events[i].UID] = true
	}
	old, err := s.Store.Between(from, to, localID)
	if err != nil {
		return 0, err
	}
	for _, e := range old {
		if !fresh[e.UID] {
			if err := s.Store.Delete(e.ID); err != nil {
				return 0, err
			}
		}
	}
	if len(events) == 0 {
		return 0, nil
	}
	return len(events), s.Store.Save(events...)
}
