# 24. The Bible app

The **Bible** app (`apps/bible`, context `internal/bible`) reads the Bible in many versions and languages, follows the reading
progress and shows the **verse of the day**. The versions come from the public [Midvash](https://api.midvash.com) API (no key, 119
versions in 69 languages), are **downloaded once** into the reader's SQLite database and are then read **offline**. It follows the
app contract (`apps/hello` is the blueprint) and has the same three tabs as Hello Go.

## Using it

| Tab | What it has |
|---|---|
| **Home** | the percentage of the Bible read (`0%` to `100%`), the **Verse of the day** card with its "I read it" box and a button to **continue reading** where you stopped |
| **Bible** | the selection bar (current version, language, version, search) and below it the books, the chapters, the verse numbers and the reader |
| **More** | **Settings**, **About** and **Exit** |

**First run.** With no version on the reader, Home offers "Download the Bible": one tap fetches the list of versions, downloads the
one of the reader's language (Almeida Livre in Portuguese, King James in English) and starts. Later, on the **Bible** tab, pick
another language and version; a version that is not on the reader yet asks before downloading (a few megabytes, in about 60 requests).

**Reading.** Book → chapter (the list shows the start of each chapter, or just the number, as the settings say) → the grid of verse
numbers, to jump straight to one, or **Read the chapter**. Reading is by pages (the reader's page keys turn them too); every verse
that **appears on the screen** counts as read, and the arrows at the top go to the previous or next chapter, even across books. The
list of books is in the original order and the **Alphabetical order** button toggles it.

**Verse of the day.** On special dates (Christmas and its eve, New Year, Epiphany, Palm Sunday, Good Friday, Easter, the Ascension and
Pentecost, with Easter computed for each year) the verse is the one of the date; on any other day it is drawn from the version in
use, the same all day and kept, so it does not change if you switch versions. Ticking "I read it" replaces the box by the done icon
and adds the verse to the progress. (The reader's screen is in grays: the "green" is the dark square with the check.)

**Settings.** Font size (24 to 56, in steps of 2, with a live preview), typeface (Go, Go Medium, Go Mono), high contrast (inverts the
colors), reading aloud, chapter previews and the language of the app (Portuguese or English; it starts in the reader's).

## What is kept

Everything in `applications/.pbgo/bible.db`, a SQLite file (the same `internal/storage` of the other apps, with the migrations of the
`bible` module):

| Table | Content |
|---|---|
| `bible_books`, `bible_versions` | the catalog (books with names in pt-BR and en; versions, with what was downloaded) |
| `bible_verses` | the text, **one verse per row**, key `(version, book, chapter, verse)` |
| `bible_read` | what was read, **by place only** `(book, chapter, verse)`, without the version |
| `bible_state` | the position (last book, chapter and verse) and the preferences |
| `bible_daily` | the verse chosen for each day (a month is kept) |

**The progress is unified.** Because `bible_read` does not know the version, reading John 3:16 in any translation counts for all of
them. The percentage is `read ÷ total` **of the version in use**, counting only the verses it has: a version with more or fewer books
or chapters changes the denominator in proportion (and what was read in another version only counts when the active one has that verse).

## Memory (the reader has 1 GB)

- The text is **never** loaded whole: the screen keeps the list of books and versions (a few KB) and, while reading, **one chapter**
  (176 verses at most). Every page is rebuilt when it is shown.
- A download brings **20 chapters per request** (`/v1/passages`), writes each batch in a transaction and lets the memory go; if it is
  interrupted, it resumes from the chapters that are missing.
- Search reads the rows **one at a time** and stops at 60 results, ignoring accents and case: the memory used does not depend on the
  size of the version.
- SQLite's page cache is ~2 MB (`internal/storage`).

## How it is built

| Layer | Package | What it has |
|---|---|---|
| domain | `internal/bible/domain` | `Book`, `Version`, `Ref`, `ChapterText`, `Preferences`, the `Canon` (the 66 books, to list before anything is downloaded), `FestiveVerse`, `Easter`, and the `Catalog`, `Library` and `Speaker` ports |
| application | `internal/bible/application` | `Bible`: refresh the catalog, download, read a chapter, count what was read, resume, verse of the day, search |
| infrastructure | `internal/bible/infrastructure` | `midvash` (the API over HTTP+JSON) and `sqlstore` (the text and the progress in the database) |
| presentation | `internal/bible/presentation` | the screens, built only from toolkit widgets |
| root | `apps/bible` | wires it up and picks the device parts: HTTP client, database, language, speech |

For reading, the toolkit got three generic additions (they know nothing about the Bible): the **ParagraphPager**
([chapter 6](06-components.md)), the typeface chosen with `gfx.SetFamily` and the negative screen with `Document.SetInvert`.

## What was verified and what is missing

- **Tested on the PC:** the domain, the use cases, the database (on disk and in memory), the API client against a fake server, the
  screens (every page fits the screen and every element gets its touch) and the full cycle in `apps/bible`: download → read → close →
  reopen with no network, with the position, the progress, the read verse and the settings in place.
- **Compiled and tested on ARMv7** (`.\scripts\arm-test.ps1`, Go 1.17 + qemu) and compiled with the reader's SDK.
- **The reader's language** (checked on a PB741 and in the SDK): the firmware keeps it in `global.cfg` in the user's storage
  (`/mnt/ext1/system/config/global.cfg`, `CONFIGPATH` of `inkview.h`), as `language=pt`. The app reads it the way the SDK offers
  (`GetGlobalConfig` + `ReadString`, `inkview.GlobalConfigString`), then the file itself, then `LANG`/`LC_*`, and last the language of the
  apps' shared settings. (`/ebrmain` is the firmware's read-only area; the settings are not there.)
- **Not tested on the device:** the speed of SQLite and of the download on a real PB741 and the memory use measured there.
- **Reading aloud:** the PocketBook SDK (6.3, `inkview.h`) has **no speech API for apps** (only the `PB_TTS_STATE` enum and an internal
  `hw_isttsplaying`). The firmware's own voices are on the reader (`system/tts/<voice>/`, e.g. `pt_br_rafael`, `pt_br_helena`,
  `en_us_julie`; `ttslanguage=`, `ttsspeed=` and `tts_volume=` in `global.cfg`): the **ReadSpeaker VT** engine (`libvtapi.so`,
  `librsttswrapper.so`, the `VT_LOADTTS_RS`/`VT_PLAYTTS_RS` functions), with no public header, license-checked and loaded by the reader
  application. Calling it from an app would be reverse engineering, so it was not done. The Settings option exists and the **Listen**
  button shows, but it only speaks if the reader has a speech program (`flite`, `espeak-ng` or `espeak`); without one the app says so
  instead of pretending. When there is a way on the device, only the `Speaker` in `apps/bible/device.go` changes.
- **Colors:** the PB741 canvas is 24 bits (a Kaleido color screen) but the toolkit renders and blits 8-bit grays
  (`inkview.Blit`), so the "green" of the done icon is a dark square. Colors would be a toolkit change (an RGB pipeline), not one of
  this app.
- **Different canons:** the download asks for the 1189 chapters of the 66 books; a version with chapters beyond those (e.g. `cpdv`,
  1197) or deuterocanonical books is left without the extras (the denominator already follows what was downloaded).
