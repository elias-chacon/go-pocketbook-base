# 21. Tasks: domain

[← Calendar: domain and iCalendar](20-calendar-domain-and-icalendar.md) · [Index](README.md)

The `internal/tasks` context is the tasks app (`apps/tasks`, task T63): the **domain**, the **task cache** and the **screens**. Synchronization (T64)
and the e-mail integration (T65) come later. A quick user guide is in [chapter 23](23-productivity-apps.md).

## The domain

| Concept | What it is |
|---|---|
| `Task` | something to do: title, notes, **due date**, **My Day**, reminder, important, **steps**, **repetition** and the e-mail it came from (`MailID`) |
| `List` | a named group of tasks |
| `Step` | a small part of a task, ticked on its own ("2 of 5") |
| `View` | a filter over every list: **My Day**, **Important**, **Planned**, **Overdue** and All |

Dates are **days**, not instants: a task is due on a day, whatever the time zone (`Day` is midnight UTC of the calendar day).

**Completing** (`Complete`) ticks the task. If it **repeats** (an `RRULE`, such as `FREQ=WEEKLY;BYDAY=FR`), it also returns the **next** one: the same
task, open, due on the next date of the rule, without the ids, without My Day and without the reminder, and with the steps unticked. If it was
completed **late**, the next one counts from today (completing a weekly task a month late does not leave a pile of overdue copies); if the rule has
ended (`COUNT`), there is no next one. A repeating task needs a due date to repeat from.

The **views** leave out completed tasks (except "All") and come in the order one expects: by due date in Planned and Overdue; in the others, the
important ones first and then the newest. `Count` gives the numbers next to each view.

## The task cache (SQLite)

The `TaskStore` port is where the screen reads from; `sqlstore` implements it in the shared database (chapter 22). `Tasks` lists newest first, open ones
by default (`IncludeDone` brings the completed ones), and **the views are applied by `Filter` over what it returns**: a person has hundreds of open
tasks, not millions. Completing a repeating task saves the completed one and the next together (`Save(done, *next)`). A list that went away on the
server takes its tasks, and removing the account clears everything.

## The Tasks app

`apps/tasks` is the composition root. It needs **no account**: the tasks live on the reader, in the shared database, under the account `local` (a
list "Tasks" is created on the first run). The `Checklist` use case answers what the screens ask: the **views** with their counts, **Add** (a task added while
looking at My Day is for today, in Important it is starred, in Planned it is due today), **Toggle** (ticks, saving the next occurrence of a repeating
task, or reopens), star, My Day, due date, save and delete. The screen has the five views as chips with counts, a field to add a task, **8 tasks a
page** with a check box (the deepest touch target) and the title and due date (overdue ones say so); tapping the text opens the task to edit: title,
notes, **steps** (add, tick, remove), important, My Day and the due date (today, tomorrow, next week, none). Nothing is saved until **Save**. **Not yet tried on the device.** **Lists:** the **Lists** menu next to the add field shows one list or all of them (new tasks go to the one shown), starts a new list (the add field then asks for its name) and deletes the one shown with its tasks (the last list cannot be deleted). **Repeat** (every day, week, month or year; choosing one on a task with no due date makes it due today, and clearing the date clears the repetition). Still to
do (T63): reminders in the screen.

## The code

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/tasks/domain` | `Task`, `List`, `Step`, `View`, `Complete`, `Filter`, `Count`, the `TaskStore` port |
| application | `internal/tasks/application` | `Checklist`: views, add, toggle, star, My Day, due, save, delete |
| presentation | `internal/tasks/presentation` | the Tasks screens: the list with views and the editor |
| infrastructure | `internal/tasks/infrastructure/sqlstore` | the `TaskStore` on SQLite |

The domain uses `rrule-go` for repetition and depends only on the shared kernel; the architecture test (chapter 14) checks it.
