// Package sqlstore keeps the task lists and tasks in the shared SQLite database
// (internal/storage): the domain.TaskStore of the reader.
package sqlstore

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-pocketbook-base/internal/storage"
	"go-pocketbook-base/internal/tasks/domain"
)

// Module is the name of this schema in storage.Migrate.
const Module = "tasks"

var migrations = []storage.Migration{
	{Version: 1, SQL: `
CREATE TABLE tasks_lists (
	id TEXT NOT NULL, account_id TEXT NOT NULL, name TEXT NOT NULL,
	PRIMARY KEY (account_id, id)
);
CREATE TABLE tasks_tasks (
	id TEXT PRIMARY KEY, account_id TEXT NOT NULL, list_id TEXT NOT NULL, remote_id TEXT NOT NULL DEFAULT '',
	title TEXT NOT NULL, notes TEXT NOT NULL DEFAULT '', done INTEGER NOT NULL DEFAULT 0, done_at INTEGER NOT NULL DEFAULT 0,
	important INTEGER NOT NULL DEFAULT 0, due INTEGER NOT NULL DEFAULT 0, my_day INTEGER NOT NULL DEFAULT 0,
	reminder INTEGER NOT NULL DEFAULT 0, recurrence TEXT NOT NULL DEFAULT '', steps TEXT NOT NULL DEFAULT '[]',
	mail_id TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX tasks_tasks_list ON tasks_tasks (account_id, list_id, done);
`},
}

// Store implements domain.TaskStore.
type Store struct{ db *sql.DB }

// New prepares the tasks tables in the database and returns the store.
func New(s *storage.Store) (*Store, error) {
	if err := s.Migrate(Module, migrations); err != nil {
		return nil, err
	}
	for _, table := range []string{"tasks_lists", "tasks_tasks"} {
		if err := s.RegisterAccountTable(table); err != nil {
			return nil, err
		}
	}
	return &Store{db: s.DB()}, nil
}

// SaveLists implements domain.TaskStore.
func (s *Store) SaveLists(accountID string, lists []domain.List) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	args := []interface{}{accountID}
	for _, l := range lists {
		if _, err := tx.Exec(`INSERT INTO tasks_lists(id, account_id, name) VALUES(?, ?, ?)
			ON CONFLICT(account_id, id) DO UPDATE SET name = excluded.name`, l.ID, accountID, l.Name); err != nil {
			tx.Rollback()
			return err
		}
		args = append(args, l.ID)
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(lists)), ",")
	for _, q := range []struct{ table, column string }{{"tasks_tasks", "list_id"}, {"tasks_lists", "id"}} {
		where := "account_id = ?"
		if len(lists) > 0 {
			where += " AND " + q.column + " NOT IN (" + marks + ")"
		}
		if _, err := tx.Exec("DELETE FROM "+q.table+" WHERE "+where, args...); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// Lists implements domain.TaskStore.
func (s *Store) Lists(accountID string) ([]domain.List, error) {
	rows, err := s.db.Query(`SELECT id, name FROM tasks_lists WHERE account_id = ? ORDER BY name`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.List
	for rows.Next() {
		l := domain.List{AccountID: accountID}
		if err := rows.Scan(&l.ID, &l.Name); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// newID makes an id for a task that has none.
func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system has no randomness: nothing safe to do
	}
	return hex.EncodeToString(b)
}

// Save implements domain.TaskStore.
func (s *Store) Save(tasks ...domain.Task) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, t := range tasks {
		var account string
		err := tx.QueryRow(`SELECT account_id FROM tasks_lists WHERE id = ? LIMIT 1`, t.ListID).Scan(&account)
		if err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return fmt.Errorf("tasks: the list %q is not in the store", t.ListID)
			}
			return err
		}
		if t.ID == "" {
			t.ID = newID()
		}
		if err := insert(tx, account, t); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func unix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func fromUnix(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

func flag(b bool) int {
	if b {
		return 1
	}
	return 0
}

func insert(tx *sql.Tx, account string, t domain.Task) error {
	steps, _ := json.Marshal(t.Steps)
	_, err := tx.Exec(`
INSERT INTO tasks_tasks(id, account_id, list_id, remote_id, title, notes, done, done_at, important, due, my_day, reminder,
	recurrence, steps, mail_id, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	list_id = excluded.list_id, remote_id = excluded.remote_id, title = excluded.title, notes = excluded.notes,
	done = excluded.done, done_at = excluded.done_at, important = excluded.important, due = excluded.due,
	my_day = excluded.my_day, reminder = excluded.reminder, recurrence = excluded.recurrence, steps = excluded.steps,
	mail_id = excluded.mail_id, created_at = excluded.created_at`,
		t.ID, account, t.ListID, t.RemoteID, t.Title, t.Notes, flag(t.Done), unix(t.DoneAt), flag(t.Important), unix(t.Due),
		unix(t.MyDay), unix(t.Reminder), t.Recurrence, string(steps), t.MailID, unix(t.CreatedAt))
	return err
}

const taskColumns = `id, list_id, remote_id, title, notes, done, done_at, important, due, my_day, reminder, recurrence, steps, mail_id, created_at`

type scanner interface {
	Scan(dest ...interface{}) error
}

func scan(r scanner) (domain.Task, error) {
	var t domain.Task
	var done, important int
	var doneAt, due, myDay, reminder, created int64
	var steps string
	if err := r.Scan(&t.ID, &t.ListID, &t.RemoteID, &t.Title, &t.Notes, &done, &doneAt, &important, &due, &myDay, &reminder,
		&t.Recurrence, &steps, &t.MailID, &created); err != nil {
		return domain.Task{}, err
	}
	if err := json.Unmarshal([]byte(steps), &t.Steps); err != nil {
		return domain.Task{}, fmt.Errorf("sqlstore: a damaged row: %w", err)
	}
	t.Done, t.Important = done != 0, important != 0
	t.DoneAt, t.Due, t.MyDay, t.Reminder, t.CreatedAt = fromUnix(doneAt), fromUnix(due), fromUnix(myDay), fromUnix(reminder), fromUnix(created)
	return t, nil
}

// Tasks implements domain.TaskStore.
func (s *Store) Tasks(q domain.TaskQuery) ([]domain.Task, error) {
	where, args := "account_id = ?", []interface{}{q.AccountID}
	if q.ListID != "" {
		where += " AND list_id = ?"
		args = append(args, q.ListID)
	}
	if !q.IncludeDone {
		where += " AND done = 0"
	}
	rows, err := s.db.Query("SELECT "+taskColumns+" FROM tasks_tasks WHERE "+where+" ORDER BY created_at DESC, id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Task
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Get implements domain.TaskStore.
func (s *Store) Get(id string) (domain.Task, error) {
	t, err := scan(s.db.QueryRow("SELECT "+taskColumns+" FROM tasks_tasks WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return domain.Task{}, domain.ErrNotFound
	}
	return t, err
}

// Delete implements domain.TaskStore.
func (s *Store) Delete(id string) error {
	res, err := s.db.Exec("DELETE FROM tasks_tasks WHERE id = ?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
