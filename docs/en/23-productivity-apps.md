# 23. The productivity apps: Mail, Tasks and Calendar

This chapter is the **quick-reading guide** to the three productivity apps. It says what each one does today, how to install it, what does not work yet and
where the code is. The technical details are in chapters [17](17-productivity-apps-decisions.md) to [22](22-local-storage.md).

> **Current state:** the three apps build for the reader, have automated tests (including on emulated ARMv7) and are already in the Release.
> **None has been tried on the device** and no real mail provider has been tested yet.

## In one table

| App | Folder | Needs an account? | What it does today | What is missing |
|---|---|---|---|---|
| **Mail** | `apps/mail` | Yes (IMAP with an **app password**) | folders, list, reading, reply, forward, new message, delete, search, sync, offline reading | opening attachments, drafts, Bcc in the form, Outlook.com (OAuth) |
| **Tasks** | `apps/tasks` | No (kept on the reader) | My Day, Important, Planned, Overdue and All views; add, tick, edit, delete; repeating tasks | reminders on screen, sync |
| **Calendar** | `apps/calendar` | No (kept on the reader) | month with dots on the days with events, the day's agenda, create, edit and delete events (all-day or timed) | day and week views, ringing the reminders, sync, `.ics` import |

## How to install

1. Install the Launcher and the apps with the [installer](16-installer.md) (it copies everything), or
2. open the **Launcher**, search by name (`Mail`, `Tasks`, `Calendar`) and tap the row under **Available** to download and install.

After installing from the Launcher, the icon shows up on the home screen when the USB cable is unplugged.

## Mail

1. The first time, the app asks for an account: type the **address** and the **app password** (the normal password does not work on Gmail, iCloud and the like).
   - Known provider (Gmail, iCloud…): the servers are filled in for you.
   - Unknown provider: the IMAP and SMTP server fields appear.
2. Tap **Sync** to bring the **50 newest messages** of each folder. The rest stays on the server.
3. The list shows **7 messages a page** (sender, date, subject and preview; unread in bold). The reader's page keys and the arrows turn the page.
4. Tapping a message opens it **page by page** and marks it read. There you have **Reply**, **Reply all**, **Forward**, **Flag**, **Mark unread** and **Delete**.
5. **New** opens the form (To, Cc, Subject, text). A bad address or a refused send keeps the form open with the reason.

Everything is read from the **offline cache**: without a network you keep reading what already came. Delete goes to the server first (if it refuses, the message stays).

## Tasks

- The app opens on **My Day**. The chips at the top switch the view and show how many tasks each has.
- Type a title in the field and confirm to **add**: in My Day the task is for today, in Important it is starred, in Planned it is due today.
- The box on the left **ticks** the task (a repeating task creates the next one by itself). Tapping the text opens the **editor**: title, notes, steps, important,
  My Day and the due date (today, tomorrow, next week, none). Nothing is saved until **Save**.
- **Repeat** (every day, week, month or year) is in the editor.
- The **Lists** menu picks a list (or all), starts a new one and deletes the one shown.
- **8 tasks a page.**

## Calendar

- The app opens on **today's month**. A **dot** marks the days that have events; **Today** goes back to today.
- Tapping a day shows **its agenda** below (all-day first, then by time; **5 events a page**).
- **New** (or tapping an event) opens the form: title, location, the day (with arrows), **all day** and start and end as `HH:MM`, **repeat** (every day, week, month or year) and **remind** (5 minutes to 1 day before).
- An **all-day** event is a date, not an instant: it does not change day because of the time zone.
- The zone comes from the `TZ` variable or `/etc/timezone`; with neither, UTC is used (the zone setting is still to do).

## Where the data lives on the reader

Everything in `applications/.pbgo/`:

| File | Content |
|---|---|
| `accounts.json` | the accounts (no passwords) |
| `secrets.bin` | the vault: the passwords encrypted with **AES-256-GCM** (see [chapter 18](18-accounts-and-secrets.md)) |
| `productivity.db` | the shared SQLite database: messages, tasks and events ([chapter 22](22-local-storage.md)) |

Removing an account also erases everything it left in the database.

## What each part of the code does

```
apps/mail  apps/tasks  apps/calendar     the composition roots (they only wire the parts)
        │
internal/<context>/presentation          the screens, built only from toolkit widgets
internal/<context>/application           the use cases: Mailbox/Sync, Checklist, Agenda
internal/<context>/domain                the rules and the ports (Message, Task, Event…)
internal/<context>/infrastructure        what talks to the world: imapmail, smtpsend, graph,
                                         xoauth2, sqlstore, icalendar, mime
internal/accounts                        accounts, the vault and the shared sign-in screen
internal/storage                         the shared SQLite database (migrations per module)
```

The dependency rule (it only points inwards) is checked by `go test ./internal/architecture` ([chapter 14](14-architecture.md)). Every new screen
component goes to the **shared toolkit**, never inside an app: the toolkit's `Calendar`, for instance, gained `Marked` (the dot on the days) and
`OnMonth` to serve the calendar app and any other.

## What was verified and what was not

| Verified | How |
|---|---|
| Rules, screens, a touch at the center of every element, nothing overflows the screen | automated tests on the PC |
| Everything builds and passes on **ARMv7 with Go 1.17**, without zoneinfo | `.\scripts\arm-test.ps1` (QEMU in user mode) |
| The apps build for the reader | `.\build.ps1 -App <id>` |
| IMAP, SMTP and Graph | against **fake** servers |

| **Not** verified | Note |
|---|---|
| The three apps **on the device** | real touch, speed and memory are still to be checked |
| SQLite on the reader's **FAT** volume | the database uses the `TRUNCATE` journal, never `WAL`, because of this |
| IMAP/SMTP on **real providers** | only test servers so far |
| **Graph** (Outlook.com / Microsoft 365) | and sending with Bcc rests on an assumption not yet checked |
| **OAuth2** sign-in for Outlook.com | needs a Microsoft Entra *client ID* (task T53) |

## Next steps

Attachments, flags and drafts in mail; reminders in tasks; day and week views in the calendar (with a time grid component in the
toolkit); sync (Graph, CalDAV, Exchange); integrations between the apps (mail → task, invitation → event). The full list and the state of each item are in
tasks **T51 to T70** of the [README](../../README.md#tasks).
