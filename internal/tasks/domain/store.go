package domain

import "errors"

// ErrNotFound is returned for a task or list that is not in the store.
var ErrNotFound = errors.New("tasks: not found")

// TaskQuery asks for tasks. A zero ListID means every list of the account.
type TaskQuery struct {
	AccountID   string
	ListID      string
	IncludeDone bool
}

// TaskStore is the port to the tasks kept on the reader. The screens read only
// from it (offline-first); the synchronization writes to it.
type TaskStore interface {
	// SaveLists replaces the lists of an account with the ones the server has; the
	// tasks of a list that no longer exists go with it.
	SaveLists(accountID string, lists []List) error
	Lists(accountID string) ([]List, error)

	// Save keeps tasks, replacing the ones with the same ID. A task with no ID gets
	// one. The list must be one the store knows.
	Save(tasks ...Task) error
	// Tasks lists tasks, newest first. The views (My Day, Planned...) are applied by
	// Filter over what this returns: a person has hundreds of open tasks, not millions.
	Tasks(q TaskQuery) ([]Task, error)
	Get(id string) (Task, error)
	Delete(id string) error
}
