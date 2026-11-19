package domain

import "strings"

// RemotePrefix marks the lists and tasks that belong to a service (Microsoft To Do,
// Google Tasks): their ids are the service's own after it. A list without it lives only on
// the reader and is never sent anywhere.
const RemotePrefix = "r:"

// IsRemote reports whether an id (of a list or a task) belongs to a service.
func IsRemote(id string) bool { return strings.HasPrefix(id, RemotePrefix) }

// RemoteID is the service's own id of a prefixed one ("" when it has no prefix).
func RemoteID(id string) string { return strings.TrimPrefix(id, RemotePrefix) }

// LocalID is the id the reader gives to what the service calls remoteID.
func LocalID(remoteID string) string { return RemotePrefix + remoteID }

// RemoteList is a list of the service.
type RemoteList struct {
	ID   string
	Name string
}

// Backend is the port to a to-do service: Microsoft To Do (Graph) or Google Tasks.
// A Task it returns has the service's id in RemoteID and no ListID: the caller knows it.
type Backend interface {
	Lists() ([]RemoteList, error)
	Tasks(listID string) ([]Task, error)
	// Create sends a new task and returns the id the service gave it.
	Create(listID string, t Task) (remoteID string, err error)
	Update(listID string, t Task) error
	Delete(listID, remoteID string) error
}

// Change is something the person did on the reader that the service does not know yet.
type Change struct {
	TaskID   string
	ListID   string // the local, prefixed id of the list
	RemoteID string // "" for a task the service has never seen
	Deleted  bool
}

// ChangeLog keeps the Changes until the synchronization sends them.
type ChangeLog interface {
	Pending() ([]Change, error)
	// Resolve forgets the change of a task: the service has it now.
	Resolve(taskID string) error
}
