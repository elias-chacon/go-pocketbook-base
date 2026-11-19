package sqlstore

import (
	"database/sql"

	"go-pocketbook-base/internal/tasks/domain"
)

// Raw returns the same store without the change log: what the synchronization writes
// (what the service said) is not something to send back.
func (s *Store) Raw() *Store {
	c := *s
	c.untracked = true
	return &c
}

// note records that a task of a service changed (or was deleted).
func note(tx *sql.Tx, account string, t domain.Task, deleted bool) error {
	if !domain.IsRemote(t.ListID) {
		return nil
	}
	_, err := tx.Exec(`INSERT INTO tasks_outbox(task_id, account_id, list_id, remote_id, deleted) VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(task_id) DO UPDATE SET list_id = excluded.list_id, remote_id = excluded.remote_id, deleted = excluded.deleted`,
		t.ID, account, t.ListID, t.RemoteID, flag(deleted))
	return err
}

// Pending implements domain.ChangeLog.
func (s *Store) Pending() ([]domain.Change, error) {
	rows, err := s.db.Query(`SELECT task_id, list_id, remote_id, deleted FROM tasks_outbox ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Change
	for rows.Next() {
		var c domain.Change
		var deleted int
		if err := rows.Scan(&c.TaskID, &c.ListID, &c.RemoteID, &deleted); err != nil {
			return nil, err
		}
		c.Deleted = deleted != 0
		out = append(out, c)
	}
	return out, rows.Err()
}

// Resolve implements domain.ChangeLog.
func (s *Store) Resolve(taskID string) error {
	_, err := s.db.Exec(`DELETE FROM tasks_outbox WHERE task_id = ?`, taskID)
	return err
}
