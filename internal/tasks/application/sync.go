package application

import (
	"errors"
	"fmt"

	"go-pocketbook-base/internal/tasks/domain"
)

// Sync keeps the reader's tasks and the lists of a service (Microsoft To Do, Google Tasks)
// the same. The reader stays the place the screens read: Run first sends what the
// person changed offline, then brings what the service has. A task the person changed
// and the service has not seen yet wins over the service's version (it is sent, not
// overwritten). What the service has no field for (steps, repetition, My Day, the e-mail
// a task came from) stays as the reader has it.
type Sync struct {
	AccountID string
	// Store is the store without the change log (sqlstore's Raw): what Sync writes is what
	// the service said, not a change to send back.
	Store   domain.TaskStore
	Log     domain.ChangeLog
	Backend domain.Backend
}

// SyncResult counts what a Run did, and what went wrong with single items (a failed
// item stays to be sent again; it does not stop the others).
type SyncResult struct {
	Sent, Received, Removed int
	Errors                  []error
}

// Run synchronizes once.
func (s Sync) Run() (SyncResult, error) {
	var res SyncResult
	remote, err := s.Backend.Lists()
	if err != nil {
		return res, fmt.Errorf("tasks: the lists of the service: %w", err)
	}
	pending, err := s.Log.Pending()
	if err != nil {
		return res, err
	}
	s.send(pending, &res)
	if err := s.saveLists(remote); err != nil {
		return res, err
	}
	waiting := map[string]bool{}
	left, err := s.Log.Pending()
	if err != nil {
		return res, err
	}
	for _, c := range left {
		waiting[c.TaskID] = true
	}
	local, err := s.Store.Tasks(domain.TaskQuery{AccountID: s.AccountID, IncludeDone: true})
	if err != nil {
		return res, err
	}
	byRemote := map[string]domain.Task{}
	for _, t := range local {
		if t.RemoteID != "" {
			byRemote[t.ListID+"/"+t.RemoteID] = t
		}
	}
	for _, l := range remote {
		s.receive(l, byRemote, waiting, &res)
	}
	return res, nil
}

// send pushes the changes the person made on the reader.
func (s Sync) send(pending []domain.Change, res *SyncResult) {
	for _, c := range pending {
		if err := s.sendOne(c); err != nil {
			res.Errors = append(res.Errors, err)
			continue
		}
		res.Sent++
	}
}

func (s Sync) sendOne(c domain.Change) error {
	list := domain.RemoteID(c.ListID)
	if c.Deleted {
		if c.RemoteID != "" {
			if err := s.Backend.Delete(list, c.RemoteID); err != nil && !errors.Is(err, domain.ErrNotFound) {
				return err
			}
		}
		return s.Log.Resolve(c.TaskID)
	}
	t, err := s.Store.Get(c.TaskID)
	if errors.Is(err, domain.ErrNotFound) {
		return s.Log.Resolve(c.TaskID) // deleted before it was ever sent
	}
	if err != nil {
		return err
	}
	if t.RemoteID == "" {
		id, err := s.Backend.Create(list, t)
		if err != nil {
			return err
		}
		t.RemoteID = id
		if err := s.Store.Save(t); err != nil {
			return err
		}
	} else if err := s.Backend.Update(list, t); err != nil {
		return err
	}
	return s.Log.Resolve(c.TaskID)
}

// saveLists makes the lists of the service the lists of the reader that carry its prefix,
// next to the lists that live only on the reader.
func (s Sync) saveLists(remote []domain.RemoteList) error {
	current, err := s.Store.Lists(s.AccountID)
	if err != nil {
		return err
	}
	var all []domain.List
	for _, l := range current {
		if !domain.IsRemote(l.ID) {
			all = append(all, l)
		}
	}
	for _, r := range remote {
		all = append(all, domain.List{ID: domain.LocalID(r.ID), AccountID: s.AccountID, Name: r.Name})
	}
	return s.Store.SaveLists(s.AccountID, all)
}

// receive brings the tasks of one list of the service.
func (s Sync) receive(l domain.RemoteList, byRemote map[string]domain.Task, waiting map[string]bool, res *SyncResult) {
	localList := domain.LocalID(l.ID)
	tasks, err := s.Backend.Tasks(l.ID)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("tasks: the list %q: %w", l.Name, err))
		return // keep what the reader has: the list could not be read, not emptied
	}
	seen := map[string]bool{}
	for _, r := range tasks {
		seen[r.RemoteID] = true
		t, known := byRemote[localList+"/"+r.RemoteID]
		if known && waiting[t.ID] {
			continue // the person's change goes first
		}
		if !known {
			t = domain.Task{ID: domain.LocalID(r.RemoteID), ListID: localList, RemoteID: r.RemoteID, CreatedAt: r.CreatedAt}
		}
		if err := s.Store.Save(merge(t, r)); err != nil {
			res.Errors = append(res.Errors, err)
			continue
		}
		res.Received++
	}
	s.prune(localList, seen, byRemote, waiting, res)
}

// merge puts the fields the service has onto the reader's task (the others stay).
func merge(t, r domain.Task) domain.Task {
	t.Title, t.Notes, t.Done, t.DoneAt, t.Important, t.Due = r.Title, r.Notes, r.Done, r.DoneAt, r.Important, r.Due
	if !r.Reminder.IsZero() {
		t.Reminder = r.Reminder
	}
	return t
}

// prune removes the tasks of a list that the service no longer has, unless the person
// changed them and the change is still waiting to be sent.
func (s Sync) prune(localList string, seen map[string]bool, byRemote map[string]domain.Task, waiting map[string]bool, res *SyncResult) {
	for key, t := range byRemote {
		if t.ListID != localList || seen[t.RemoteID] || waiting[t.ID] {
			continue
		}
		if err := s.Store.Delete(t.ID); err == nil {
			res.Removed++
		}
		delete(byRemote, key)
	}
}
