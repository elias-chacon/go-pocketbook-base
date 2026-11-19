// Package domain is the heart of the Tasks context: tasks, their lists, steps,
// due dates and repetition, and the smart views of a to-do app ("My Day",
// "Important", "Planned"). It knows nothing of servers, files or the screen.
//
// Ubiquitous language:
//
//   - Task: something to do, with an optional due date, steps and repetition.
//   - List: a named group of tasks.
//   - Step: a small part of a task, ticked on its own.
//   - View: a filter over every list ("My Day", "Important", "Planned").
//
// Dates are days, not instants: a task is due on a day, whatever the time zone.
package domain

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// Step is a small part of a task.
type Step struct {
	Title string
	Done  bool
}

// Task is something to do.
type Task struct {
	ID        string
	ListID    string
	RemoteID  string
	Title     string
	Notes     string
	Done      bool
	DoneAt    time.Time // when it was ticked (zero while open)
	Important bool
	// Due is the day the task is due, at midnight UTC (zero when it has none).
	Due time.Time
	// MyDay is the day the person chose to do it (zero when it is not on a day's list).
	MyDay time.Time
	// Reminder is when to tell the person (zero for none).
	Reminder time.Time
	// Recurrence is an RRULE without the "RRULE:" prefix ("FREQ=WEEKLY;BYDAY=FR").
	Recurrence string
	Steps      []Step
	// MailID is the e-mail the task came from, to open it back (Tasks <-> Mail).
	MailID    string
	CreatedAt time.Time
}

// List is a named group of tasks.
type List struct {
	ID        string
	AccountID string
	Name      string
}

// Day returns midnight UTC of the calendar day of t, as the domain stores dates.
func Day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Validate checks the rules of a task.
func (t Task) Validate() error {
	var problems []string
	if strings.TrimSpace(t.Title) == "" {
		problems = append(problems, "title is required")
	}
	if t.Recurrence != "" {
		if t.Due.IsZero() {
			problems = append(problems, "a repeating task needs a due date to repeat from")
		}
		if _, err := rrule.StrToROption(t.Recurrence); err != nil {
			problems = append(problems, "recurrence: "+err.Error())
		}
	}
	if len(problems) > 0 {
		return errors.New("tasks: " + strings.Join(problems, "; "))
	}
	return nil
}

// Overdue reports whether the task is open and was due before today.
func (t Task) Overdue(today time.Time) bool {
	return !t.Done && !t.Due.IsZero() && Day(t.Due).Before(Day(today))
}

// StepsDone and StepsTotal count the steps, for "2 of 5".
func (t Task) StepsDone() int {
	n := 0
	for _, s := range t.Steps {
		if s.Done {
			n++
		}
	}
	return n
}

// StepsTotal is how many steps the task has.
func (t Task) StepsTotal() int { return len(t.Steps) }

// Complete ticks a task. A repeating task also gives the next one: the same task,
// open, due on the next date of its rule after the one just done (or after today
// when it was done late, so finishing a weekly task a month late does not leave a
// pile of overdue copies). next is nil when the task does not repeat or the rule
// has ended.
func (t Task) Complete(now time.Time) (done Task, next *Task, err error) {
	if t.Done {
		return t, nil, nil
	}
	t.Done, t.DoneAt = true, now
	if t.Recurrence == "" {
		return t, nil, nil
	}
	if err := t.Validate(); err != nil {
		return t, nil, err
	}
	opt, _ := rrule.StrToROption(t.Recurrence) // Validate checked it
	opt.Dtstart = Day(t.Due)
	rule, err := rrule.NewRRule(*opt)
	if err != nil {
		return t, nil, err
	}
	after := Day(t.Due)
	if today := Day(now); today.After(after) {
		after = today
	}
	following := rule.After(after, false)
	if following.IsZero() {
		return t, nil, nil // the rule has ended
	}
	n := t
	n.ID, n.RemoteID = "", ""
	n.Done, n.DoneAt = false, time.Time{}
	n.Due, n.MyDay = Day(following), time.Time{}
	n.Reminder = time.Time{}
	n.Steps = make([]Step, len(t.Steps))
	for i, s := range t.Steps {
		n.Steps[i] = Step{Title: s.Title} // the steps start again
	}
	return t, &n, nil
}

// View is a filter over the tasks of every list.
type View string

// The smart views.
const (
	ViewMyDay     View = "my-day"
	ViewImportant View = "important"
	ViewPlanned   View = "planned"
	ViewOverdue   View = "overdue"
	ViewAll       View = "all"
)

// In reports whether the task belongs to a view today. Done tasks are left out of
// every view but "all".
func (t Task) In(v View, today time.Time) bool {
	if v == ViewAll {
		return true
	}
	if t.Done {
		return false
	}
	switch v {
	case ViewMyDay:
		return !t.MyDay.IsZero() && Day(t.MyDay).Equal(Day(today))
	case ViewImportant:
		return t.Important
	case ViewPlanned:
		return !t.Due.IsZero()
	case ViewOverdue:
		return t.Overdue(today)
	}
	return false
}

// Filter returns the tasks of a view, in the order the person expects: by due
// date for "Planned" and "Overdue", otherwise important first, then newest first.
func Filter(tasks []Task, v View, today time.Time) []Task {
	var out []Task
	for _, t := range tasks {
		if t.In(v, today) {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if v == ViewPlanned || v == ViewOverdue {
			return a.Due.Before(b.Due)
		}
		if a.Important != b.Important {
			return a.Important
		}
		return a.CreatedAt.After(b.CreatedAt)
	})
	return out
}

// Counts are the numbers shown next to the views.
type Counts struct {
	MyDay, Important, Planned, Overdue int
}

// Count counts the open tasks of each view.
func Count(tasks []Task, today time.Time) Counts {
	var c Counts
	for _, t := range tasks {
		if t.In(ViewMyDay, today) {
			c.MyDay++
		}
		if t.In(ViewImportant, today) {
			c.Important++
		}
		if t.In(ViewPlanned, today) {
			c.Planned++
		}
		if t.In(ViewOverdue, today) {
			c.Overdue++
		}
	}
	return c
}
