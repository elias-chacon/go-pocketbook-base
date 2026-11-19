# 20. Calendar: domain and iCalendar

[← Mail: domain and MIME](19-mail-domain-and-mime.md) · [Index](README.md)

The `internal/calendar` context is the base of the future calendar app (`apps/calendar`, tasks T66 to T70). For now it has the **domain**
(events, recurrence, reminders) and **reading and writing `.ics` files**, with import and export. Synchronization (CalDAV, Google, Graph, EWS),
storage (T54) and the screen come later.

## The domain

| Concept | What it is |
|---|---|
| `Event` | something that happens at a time: title, description, location, start and end (the end is exclusive), time zone, recurrence, exceptions, reminders, organizer, attendees, status and `Sequence` |
| `Occurrence` | one appearance of an event (a repeating event has many) |
| `Calendar` | a named collection of events; it can be **read-only** (imported or subscribed) |
| `Reminder` | how long **before** the start the person wants to be told |

An **all-day** event has its start and end at midnight UTC (the end is the day after the last). An event that **repeats** keeps the local time of
its zone across daylight saving: a "9 am in New York every Monday" stays at 9 am after the change, and becomes 13:00 UTC instead of 14:00 (there is
a test for that).

**`Occurrences(from, to)`** gives the appearances that overlap a range: a single event gives itself, a repeating one is expanded by its `RRULE`
without the exceptions (`EXDATE`), and an occurrence that began before the range but is still going (a night shift) is listed. A boundless rule
("every minute, forever") is capped at 2,000 occurrences, so it cannot fill the reader's memory. **`Expand`** merges several events, sorts by start
and applies **changes to a single occurrence** (`RecurrenceID`): the postponed version replaces the original, and a cancelled event shows nowhere.
The expansion uses `rrule-go`, because the rules of RFC 5545 are long and exact.

## `.ics` files (iCalendar)

`icalendar.Parse` reads the events of a file and also returns a list of **warnings**; `icalendar.Write` writes the file. The syntax (folded lines,
escapes) is `go-ical`'s; the meaning is here.

- **Time zones**: Outlook writes Windows names (`E. South America Standard Time`): a table turns them into IANA names (`America/Sao_Paulo`). IANA
  names with a path prefix (`/mozilla.org/20050126_1/Europe/Paris`) are understood too. An unknown zone becomes UTC **with a warning**: the event is
  not lost, only possibly shifted. The time zone database is carried in the program (`time/tzdata`), since the reader may have no `zoneinfo`.
- **Dates and durations**: UTC (`...Z`), with `TZID`, "floating" and all-day dates; the end comes from `DTEND`, from `DURATION`, from the whole day or
  from the start itself.
- **Recurrence**: `RRULE`, `EXDATE` (one or several values per line) and `RECURRENCE-ID` for changed occurrences.
- **Reminders** (`VALARM`): before the start (`-PT15M`) or at a fixed instant. **People**: organizer and attendees with name, email and answer.
- **Problems**: an event with no start, with an end before its start or with an invalid rule is **skipped with a warning** and the others are kept; an
  event with no `UID` gets one (`imported-N`); an event with no title shows as "(sem título)". `VTODO` and other components are ignored (tasks will have
  their own reader).
- **Writing**: times with a zone keep their `TZID`; without one they go out in UTC; all-day events go out as dates. The file carries **no
  `VTIMEZONE`**: the IANA name is enough for the calendars that matter, and one that does not know it shows the event in UTC. With no events there is
  no file (`ErrNothingToWrite`), because an empty iCalendar is not valid.

## The event cache (SQLite)

The `EventStore` port is where the screen reads from; `sqlstore` implements it in the shared database (chapter 22). **`Between(from, to)`** returns the
events that **can** have an occurrence in the range, ready for `Expand`: the single ones that overlap it, the repeating ones that started before its
end, and the **changes to a single occurrence** (including ones that moved out of the range to another day, and ones that came in from outside,
because the original has to be replaced). Saving again replaces (same calendar, `UID` and `RecurrenceID`); deleting a repeating event takes its changes
with it; a calendar that went away on the server takes its events; and removing the account clears everything.

## The code

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/calendar/domain` | `Event`, `Occurrence`, `Calendar`, `Occurrences`, `Expand`, the validation rules |
| infrastructure | `internal/calendar/infrastructure/icalendar` | `Parse`, `Write`, the zone table |
| | `internal/calendar/infrastructure/sqlstore` | the `EventStore` on SQLite: `Between`, calendars |

The domain uses `rrule-go` (a small library with no dependencies, like the standard library) and knows neither the platform nor the toolkit; the architecture test (chapter 14) checks the rest.
