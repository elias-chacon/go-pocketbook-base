// Package sqlstore keeps the calendars and events in the shared SQLite database
// (internal/storage): the domain.EventStore of the reader.
package sqlstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
	"go-pocketbook-base/internal/storage"
)

// Module is the name of this schema in storage.Migrate.
const Module = "calendar"

var migrations = []storage.Migration{
	{Version: 1, SQL: `
CREATE TABLE calendar_calendars (
	id TEXT NOT NULL, account_id TEXT NOT NULL, name TEXT NOT NULL, read_only INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (account_id, id)
);
CREATE TABLE calendar_events (
	id TEXT PRIMARY KEY, account_id TEXT NOT NULL, calendar_id TEXT NOT NULL, remote_id TEXT NOT NULL DEFAULT '',
	uid TEXT NOT NULL, title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', location TEXT NOT NULL DEFAULT '',
	start INTEGER NOT NULL, end INTEGER NOT NULL, all_day INTEGER NOT NULL DEFAULT 0, tz TEXT NOT NULL DEFAULT '',
	recurrence TEXT NOT NULL DEFAULT '', exceptions TEXT NOT NULL DEFAULT '[]', recurrence_id INTEGER NOT NULL DEFAULT 0,
	reminders TEXT NOT NULL DEFAULT '[]', organizer TEXT NOT NULL DEFAULT '{}', attendees TEXT NOT NULL DEFAULT '[]',
	status TEXT NOT NULL DEFAULT 'confirmed', sequence INTEGER NOT NULL DEFAULT 0,
	UNIQUE (calendar_id, uid, recurrence_id)
);
CREATE INDEX calendar_events_start ON calendar_events (calendar_id, start);
CREATE INDEX calendar_events_uid ON calendar_events (calendar_id, uid);
`},
}

// Store implements domain.EventStore. An event belongs to a calendar, and a calendar
// to an account: the account id of an event is the one of its calendar.
type Store struct{ db *sql.DB }

// New prepares the calendar tables in the database and returns the store.
func New(s *storage.Store) (*Store, error) {
	if err := s.Migrate(Module, migrations); err != nil {
		return nil, err
	}
	for _, table := range []string{"calendar_calendars", "calendar_events"} {
		if err := s.RegisterAccountTable(table); err != nil {
			return nil, err
		}
	}
	return &Store{db: s.DB()}, nil
}

// SaveCalendars implements domain.EventStore.
func (s *Store) SaveCalendars(accountID string, calendars []domain.Calendar) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	args := []interface{}{accountID}
	for _, c := range calendars {
		ro := 0
		if c.ReadOnly {
			ro = 1
		}
		if _, err := tx.Exec(`INSERT INTO calendar_calendars(id, account_id, name, read_only) VALUES(?, ?, ?, ?)
			ON CONFLICT(account_id, id) DO UPDATE SET name = excluded.name, read_only = excluded.read_only`,
			c.ID, accountID, c.Name, ro); err != nil {
			tx.Rollback()
			return err
		}
		args = append(args, c.ID)
	}
	where := "account_id = ?"
	if len(calendars) > 0 {
		where += " AND calendar_id NOT IN (" + strings.TrimSuffix(strings.Repeat("?,", len(calendars)), ",") + ")"
	}
	if _, err := tx.Exec("DELETE FROM calendar_events WHERE "+where, args...); err != nil {
		tx.Rollback()
		return err
	}
	where = strings.Replace(where, "calendar_id", "id", 1)
	if _, err := tx.Exec("DELETE FROM calendar_calendars WHERE "+where, args...); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Calendars implements domain.EventStore.
func (s *Store) Calendars(accountID string) ([]domain.Calendar, error) {
	rows, err := s.db.Query(`SELECT id, name, read_only FROM calendar_calendars WHERE account_id = ? ORDER BY name`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Calendar
	for rows.Next() {
		c := domain.Calendar{AccountID: accountID}
		var ro int
		if err := rows.Scan(&c.ID, &c.Name, &ro); err != nil {
			return nil, err
		}
		c.ReadOnly = ro != 0
		out = append(out, c)
	}
	return out, rows.Err()
}

// accountOf finds the account of a calendar.
func accountOf(tx *sql.Tx, calendarID string) (string, error) {
	var account string
	err := tx.QueryRow(`SELECT account_id FROM calendar_calendars WHERE id = ? LIMIT 1`, calendarID).Scan(&account)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("calendar: the calendar %q is not in the store", calendarID)
	}
	return account, err
}

// Save implements domain.EventStore.
func (s *Store) Save(events ...domain.Event) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, e := range events {
		account, err := accountOf(tx, e.CalendarID)
		if err != nil {
			tx.Rollback()
			return err
		}
		if e.ID == "" {
			e.ID = fmt.Sprintf("%s:%s:%d", e.CalendarID, e.UID, e.RecurrenceID.Unix())
			if e.RecurrenceID.IsZero() {
				e.ID = e.CalendarID + ":" + e.UID
			}
		}
		if err := insert(tx, account, e); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func insert(tx *sql.Tx, account string, e domain.Event) error {
	exceptions := make([]int64, len(e.Exceptions))
	for i, x := range e.Exceptions {
		exceptions[i] = x.Unix()
	}
	reminders := make([]int64, len(e.Reminders))
	for i, r := range e.Reminders {
		reminders[i] = int64(r / time.Second)
	}
	ex, _ := json.Marshal(exceptions)
	rem, _ := json.Marshal(reminders)
	org, _ := json.Marshal(e.Organizer)
	att, _ := json.Marshal(e.Attendees)
	var recID int64
	if !e.RecurrenceID.IsZero() {
		recID = e.RecurrenceID.Unix()
	}
	allDay := 0
	if e.AllDay {
		allDay = 1
	}
	status := string(e.Status)
	if status == "" {
		status = string(domain.Confirmed)
	}
	_, err := tx.Exec(`
INSERT INTO calendar_events(id, account_id, calendar_id, remote_id, uid, title, description, location, start, end, all_day, tz,
	recurrence, exceptions, recurrence_id, reminders, organizer, attendees, status, sequence)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(calendar_id, uid, recurrence_id) DO UPDATE SET
	remote_id = excluded.remote_id, title = excluded.title, description = excluded.description, location = excluded.location,
	start = excluded.start, end = excluded.end, all_day = excluded.all_day, tz = excluded.tz, recurrence = excluded.recurrence,
	exceptions = excluded.exceptions, reminders = excluded.reminders, organizer = excluded.organizer,
	attendees = excluded.attendees, status = excluded.status, sequence = excluded.sequence`,
		e.ID, account, e.CalendarID, e.RemoteID, e.UID, e.Title, e.Description, e.Location, e.Start.Unix(), e.End.Unix(), allDay, e.TimeZone,
		e.Recurrence, string(ex), recID, string(rem), string(org), string(att), status, e.Sequence)
	return err
}

const eventColumns = `id, calendar_id, remote_id, uid, title, description, location, start, end, all_day, tz, recurrence,
	exceptions, recurrence_id, reminders, organizer, attendees, status, sequence`

type scanner interface {
	Scan(dest ...interface{}) error
}

func scan(r scanner) (domain.Event, error) {
	var e domain.Event
	var start, end, recID int64
	var allDay int
	var status, exceptions, reminders, organizer, attendees string
	if err := r.Scan(&e.ID, &e.CalendarID, &e.RemoteID, &e.UID, &e.Title, &e.Description, &e.Location, &start, &end, &allDay, &e.TimeZone,
		&e.Recurrence, &exceptions, &recID, &reminders, &organizer, &attendees, &status, &e.Sequence); err != nil {
		return domain.Event{}, err
	}
	e.Start, e.End, e.AllDay, e.Status = time.Unix(start, 0).UTC(), time.Unix(end, 0).UTC(), allDay != 0, domain.Status(status)
	if recID != 0 {
		e.RecurrenceID = time.Unix(recID, 0).UTC()
	}
	var ex, rem []int64
	for target, raw := range map[interface{}]string{&ex: exceptions, &rem: reminders, &e.Organizer: organizer, &e.Attendees: attendees} {
		if err := json.Unmarshal([]byte(raw), target); err != nil {
			return domain.Event{}, fmt.Errorf("sqlstore: a damaged row: %w", err)
		}
	}
	for _, x := range ex {
		e.Exceptions = append(e.Exceptions, time.Unix(x, 0).UTC())
	}
	for _, r := range rem {
		e.Reminders = append(e.Reminders, time.Duration(r)*time.Second)
	}
	return e, nil
}

// Get implements domain.EventStore.
func (s *Store) Get(id string) (domain.Event, error) {
	e, err := scan(s.db.QueryRow("SELECT "+eventColumns+" FROM calendar_events WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return domain.Event{}, domain.ErrNotFound
	}
	return e, err
}

// Between implements domain.EventStore.
func (s *Store) Between(from, to time.Time, calendarIDs ...string) ([]domain.Event, error) {
	where, args := "(", []interface{}{}
	// single events overlapping the range (an event with no length counts at its instant)
	where += "(recurrence = '' AND recurrence_id = 0 AND start < ? AND (end > ? OR (end = start AND start >= ?)))"
	args = append(args, to.Unix(), from.Unix(), from.Unix())
	// repeating events that start before the range ends
	where += " OR (recurrence <> '' AND start < ?)"
	args = append(args, to.Unix())
	// changed occurrences: overlapping the range, or of a repeating event that is returned
	where += " OR (recurrence_id <> 0 AND ((start < ? AND end >= ?) OR (recurrence_id >= ? AND recurrence_id < ?)"
	args = append(args, to.Unix(), from.Unix(), from.Unix(), to.Unix())
	where += " OR uid IN (SELECT uid FROM calendar_events m WHERE m.calendar_id = calendar_events.calendar_id AND m.recurrence <> '' AND m.start < ?)))"
	args = append(args, to.Unix())
	where += ")"
	if len(calendarIDs) > 0 {
		where += " AND calendar_id IN (" + strings.TrimSuffix(strings.Repeat("?,", len(calendarIDs)), ",") + ")"
		for _, id := range calendarIDs {
			args = append(args, id)
		}
	}
	rows, err := s.db.Query("SELECT "+eventColumns+" FROM calendar_events WHERE "+where+" ORDER BY start, id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Event
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Delete implements domain.EventStore.
func (s *Store) Delete(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	var calendar, uid, recurrence string
	err = tx.QueryRow(`SELECT calendar_id, uid, recurrence FROM calendar_events WHERE id = ?`, id).Scan(&calendar, &uid, &recurrence)
	if err == sql.ErrNoRows {
		tx.Rollback()
		return domain.ErrNotFound
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	query, args := "DELETE FROM calendar_events WHERE id = ?", []interface{}{id}
	if recurrence != "" {
		query, args = "DELETE FROM calendar_events WHERE calendar_id = ? AND uid = ?", []interface{}{calendar, uid}
	}
	if _, err := tx.Exec(query, args...); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
