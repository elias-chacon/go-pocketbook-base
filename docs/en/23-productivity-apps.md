# 23. The productivity apps: Mail, Tasks and Calendar

This chapter is the **quick-reading guide** to the three productivity apps. It says what each one does today, how to install it, what does not work yet and
where the code is. The technical details are in chapters [17](17-productivity-apps-decisions.md) to [22](22-local-storage.md).

> **Current state:** the three apps build for the reader, have automated tests (including on emulated ARMv7) and are already in the Release.
> **None has been tried on the device** and no real mail provider has been tested yet.

## In one table

| App | Folder | Needs an account? | What it does today | What is missing |
|---|---|---|---|---|
| **Mail** | `apps/mail` | Yes (IMAP with an **app password**) | folders, list, reading, reply, forward, new message, delete, search, sync, offline reading | opening attachments, drafts on the server, testing Outlook.com (needs the client id) |
| **Tasks** | `apps/tasks` | No (kept on the reader) | My Day, Important, Planned, Overdue and All views; add, tick, edit, delete; repeating tasks | reminders on screen, sync |
| **Calendar** | `apps/calendar` | No (kept on the reader) | Outlook-style Month, Week, Work week and Day views, create, edit and delete events (all-day, timed, repeating, with reminder and description) | ringing the reminders, sync, `.ics` import |

## How to install

1. Install the Launcher and the apps with the [installer](16-installer.md) (it copies everything), or
2. open the **Launcher**, search by name (`Mail`, `Tasks`, `Calendar`) and tap the row under **Available** to download and install.

After installing from the Launcher, the icon shows up on the home screen when the USB cable is unplugged.

## Mail

1. The first time, the app asks for the **provider** and then for an account:
   - **Gmail, iCloud:** the **address** and an **app password** (the normal password does not work on them); the servers are filled in for you.
   - **Another server (IMAP):** the address, the password and the IMAP and SMTP servers.
   - **Outlook.com / Microsoft 365:** a code is shown to type on another device (OAuth2, no password here). It needs the app registered at Microsoft and its client id in `applications/.pbgo/oauth.json` (see [chapter 18](18-accounts-and-secrets.md)); without it the button says so.
2. Tap **Sync** to bring the **50 newest messages** of each folder. The rest stays on the server.
3. The list shows **7 messages a page** (sender, date, subject and preview; unread in bold). The reader's page keys and the arrows turn the page.
4. Tapping a message opens it **page by page** and marks it read. There you have **Reply**, **Reply all**, **Forward**, **Flag**, **Mark unread** and **Delete**.
5. **New** opens the form (To, Cc, Bcc, Subject, text). A bad address or a refused send keeps the form open with the reason. **Save draft** keeps it on the reader; **Drafts** (next to the search) lists the saved ones.

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

- It looks like the **Outlook calendar**: a toolbar (**Today**, previous and next, the period, the **view** menu: Day, Work week, Week or Month, and **New**).
- **Month:** each day lists its events. Tap a day to select it; tap it again to open it in the Day view.
- **Day, Work week, Week:** the hours down the side and the events as blocks. Tap an event to edit it, or an empty time to start an event there.
- The form has title, location, the day (with arrows), **all day**, start and end as `HH:MM`, **repeat**, **remind** and a **description**.
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
| Everything builds and passes on **ARMv7 with Go 1.17**, without zoneinfo | script ARM, já removido (QEMU in user mode) |
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

Opening attachments and OAuth sign-in for Outlook.com in mail; ringing the reminders of tasks and of the calendar; sync (Graph, CalDAV, Exchange); integrations between the apps (mail → task, invitation → event). The full list and the state of each item are in
tasks **T51 to T70** of the [README](../../README.md#tasks).

## Importing calendars (.ics and Google/Outlook links)

The calendar has an **Import** button (bottom right). Two sources:

- **An address**: any iCalendar link, including `webcal://`. For **Google Calendar** use *Settings of the calendar → Integrate calendar → Secret address in iCal format*; for **Outlook.com** use *Settings → Calendar → Shared calendars → Publish a calendar* (the ICS link). The events are copied to the reader (read once; import again to update).
- **An .ics file** of the reader: the screen lists the ones found in the storage root and in `Download` (two levels deep).

Importing again does not duplicate events (same UID replaces). This is a **one-way copy**, not a live sync: there is no two-way Google or Microsoft sync for the calendar or for Tasks yet (it needs the account's OAuth client id; see chapter 18).

## Tasks synchronization (Microsoft To Do and Google Tasks)

The Tasks app has a **Sync** menu in the title row: **Sync now**, **Connect an account** (or **Switch account**) and **Disconnect**.

- **Microsoft To Do**: through Microsoft Graph, with the same sign-in by code as the mail. An Outlook account already added in the Mail app is used as it is, with no new sign-in.
- **Google Tasks**: through the Google Tasks API. Google does not allow its Tasks scope on the device code flow, so the app uses the **authorization code with PKCE**: the screen shows a **QR code**, you read it with your phone and sign in to Google; the page that opens fails (nothing listens at `http://127.0.0.1`, that is expected) and its address has the code: copy the code, or the whole address, and type it on the reader.
- **Needs a client id**, because the apps must be registered with each provider (see below).

How it works: the reader stays what the screens read (it works offline). Lists and tasks of the service appear as lists with the service's own names; a list you create on the reader stays only on the reader. What you change offline is kept (an outbox in the database) and **sent first** on the next sync, so your change wins over the service's version of the same task; then the service's lists and tasks are brought, and what was deleted at the service is removed from the reader. A failed item stays to be sent again and does not stop the others; a list that could not be read is never emptied.

What each service keeps: Microsoft To Do syncs title, notes, done, importance, due date and reminder; Google Tasks syncs title, notes, done and due date. **Steps, repetition, My Day and the e-mail a task came from stay only on the reader.** Reopening/completing and deleting are synchronized.

### The client ids (`applications/.pbgo/oauth.json`)

```json
{"microsoft": "<client id>", "google": "<client id>", "google_secret": "<client secret>"}
```

- **Microsoft**: register an app in Entra ID (portal.azure.com → App registrations), *public client / allow public client flows*; permissions `Tasks.ReadWrite`, `Mail.ReadWrite`, `Mail.Send`, `Calendars.ReadWrite`, `offline_access`, `User.Read`.
- **Google**: in a Google Cloud project, enable the **Google Tasks API**, create an **OAuth client of type Desktop app** and copy its id and secret (for a desktop app the secret is not confidential). While the app is in *Testing*, add your account as a test user; Google expires the refresh token of a testing app after 7 days, so publish it (even unverified, for yourself) to keep it signed in.
- Without the file, the Connect screen says where to put it.

Not done yet: two-way sync of the **calendar** with Google/Microsoft (today: import of .ics links, above).
