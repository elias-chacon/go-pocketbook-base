// Package application holds the use cases of the Tasks context: what the to-do
// screens ask for (the views, add, tick, star, plan, delete). The screens read only
// from the store.
package application

import (
	"errors"
	"strings"
	"time"

	"go-pocketbook-base/internal/tasks/domain"
)

// DefaultListID is the id of the list a new install starts with.
const DefaultListID = "tasks"

// ErrEmptyTitle is returned when adding a task with no title.
var ErrEmptyTitle = errors.New("tasks: a task needs a title")

// Checklist is the use case of the to-do list of one account.
type Checklist struct {
	AccountID string
	Store     domain.TaskStore
	// Now is the clock (time.Now when nil).
	Now func() time.Time
}

func (u Checklist) now() time.Time {
	if u.Now != nil {
		return u.Now()
	}
	return time.Now()
}

// Today is the day the views are about.
func (u Checklist) Today() time.Time { return domain.Day(u.now()) }

// EnsureList makes sure there is a list to put tasks in, creating one called name
// when the account has none.
func (u Checklist) EnsureList(name string) (domain.List, error) {
	lists, err := u.Store.Lists(u.AccountID)
	if err != nil {
		return domain.List{}, err
	}
	if len(lists) > 0 {
		return lists[0], nil
	}
	l := domain.List{ID: DefaultListID, AccountID: u.AccountID, Name: name}
	return l, u.Store.SaveLists(u.AccountID, []domain.List{l})
}

// All returns every task of the account, open and done.
func (u Checklist) All() ([]domain.Task, error) {
	return u.Store.Tasks(domain.TaskQuery{AccountID: u.AccountID, IncludeDone: true})
}

// View returns the tasks of a view and the counts of all of them.
func (u Checklist) View(v domain.View) ([]domain.Task, domain.Counts, error) {
	all, err := u.All()
	if err != nil {
		return nil, domain.Counts{}, err
	}
	today := u.Today()
	return domain.Filter(all, v, today), domain.Count(all, today), nil
}

// Add creates a task in the account's first list. A task added while looking at a
// view belongs to it: in My Day it is for today, in Important it is starred, in
// Planned it is due today.
func (u Checklist) Add(title string, v domain.View) (domain.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Task{}, ErrEmptyTitle
	}
	list, err := u.EnsureList("")
	if err != nil {
		return domain.Task{}, err
	}
	t := domain.Task{ListID: list.ID, Title: title, CreatedAt: u.now()}
	switch v {
	case domain.ViewMyDay:
		t.MyDay = u.Today()
	case domain.ViewImportant:
		t.Important = true
	case domain.ViewPlanned, domain.ViewOverdue:
		t.Due = u.Today()
	}
	if err := t.Validate(); err != nil {
		return domain.Task{}, err
	}
	if err := u.Store.Save(t); err != nil {
		return domain.Task{}, err
	}
	return t, nil
}

// Toggle ticks an open task (a repeating one gives its next occurrence) or reopens a done one.
func (u Checklist) Toggle(id string) error {
	t, err := u.Store.Get(id)
	if err != nil {
		return err
	}
	if t.Done {
		t.Done, t.DoneAt = false, time.Time{}
		return u.Store.Save(t)
	}
	done, next, err := t.Complete(u.now())
	if err != nil {
		return err
	}
	if next != nil {
		return u.Store.Save(done, *next)
	}
	return u.Store.Save(done)
}

// Change applies a change to a task and keeps it.
func (u Checklist) Change(id string, change func(*domain.Task)) error {
	t, err := u.Store.Get(id)
	if err != nil {
		return err
	}
	change(&t)
	if err := t.Validate(); err != nil {
		return err
	}
	return u.Store.Save(t)
}

// SetImportant stars or unstars a task.
func (u Checklist) SetImportant(id string, on bool) error {
	return u.Change(id, func(t *domain.Task) { t.Important = on })
}

// SetMyDay puts a task on today's list or takes it off.
func (u Checklist) SetMyDay(id string, on bool) error {
	return u.Change(id, func(t *domain.Task) {
		t.MyDay = time.Time{}
		if on {
			t.MyDay = u.Today()
		}
	})
}

// SetDue gives a task a due day (the zero time clears it).
func (u Checklist) SetDue(id string, day time.Time) error {
	return u.Change(id, func(t *domain.Task) {
		t.Due = time.Time{}
		if !day.IsZero() {
			t.Due = domain.Day(day)
		}
	})
}

// Get returns one task.
func (u Checklist) Get(id string) (domain.Task, error) { return u.Store.Get(id) }

// Save keeps an edited task.
func (u Checklist) Save(t domain.Task) error {
	if err := t.Validate(); err != nil {
		return err
	}
	return u.Store.Save(t)
}

// Delete removes a task.
func (u Checklist) Delete(id string) error { return u.Store.Delete(id) }
