# 21. Tasks: domain

[← Calendar: domain and iCalendar](20-calendar-domain-and-icalendar.md) · [Index](README.md)

The `internal/tasks` context is the base of the future tasks app (`apps/tasks`, task T63). For now it has only the **domain**; the screen,
storage (T54), synchronization (T64) and the e-mail integration (T65) come later.

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

## The code

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/tasks/domain` | `Task`, `List`, `Step`, `View`, `Complete`, `Filter`, `Count` |

The domain uses `rrule-go` for repetition and depends only on the shared kernel; the architecture test (chapter 14) checks it.
