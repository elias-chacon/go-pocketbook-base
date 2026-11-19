# 17. Architecture decisions: the e-mail, task list and calendar apps

[← The installer and the releases](16-installer.md) · [Index](README.md)

This is the decision record (ADR) of task **T51**, which precedes the three productivity apps (T58 to T70). Every decision has its
reason and, where it could be measured, the **evidence**. The choices were made without asking, so the work does not stop:
**they stand until you veto them**, and changing one is cheap because each sits behind a port (an interface) of the domain.

## Constraints that decide everything

- App code is **Go 1.17** (the PocketBook SDK's compiler): no generics, no `any`, no new standard library functions
  (`errors.Join`, `strings.Cut`…). A library that uses them **does not compile**.
- Memory: **1 GB**, one app at a time (the Launcher replaces the process with `execve`). Screens and lists cannot load everything.
- The development PC **has no `gcc`**: nothing that needs cgo can be tested locally (only in Docker). cgo stays confined to `inkview`.
- No physical keyboard (the system one only returns the text on confirm), no browser (authentication cannot open a page).

## Decisions

| # | Subject | Decision | Reason and evidence |
|---|---|---|---|
| 1 | Local storage (T54) | **Pure-Go SQLite**, `modernc.org/sqlite` v1.14.8, behind a `store` port | It built with the SDK's Go 1.17 for ARM in 19 s and runs on the PC without `gcc`. Cost: **+5.5 MB** per app that uses it (a test binary went from 1 MB to 6.8 MB). **Not verified:** running on ARM (the Docker here has no emulation), so the first step of T54 is to run it on the reader |
| 2 | Queries and search | SQL with indexes and simple `LIKE`/FTS; **always paginated** | Avoids loading the whole inbox; a small `PRAGMA cache_size` (about 2 MB) |
| 3 | IMAP (T59) | `github.com/emersion/go-imap` **v1.2.1** | Builds on Go 1.17/ARM. **v2** needs Go 1.18 and is out |
| 4 | SMTP (T59) | **standard library `net/smtp`**, with our own `smtp.Auth` for XOAUTH2 | `go-smtp` v0.26 **does not build** on Go 1.17 (it uses `errors.Join` and `strings.Cut`). `net/smtp` already does STARTTLS, PLAIN and LOGIN |
| 5 | MIME and mail (T58) | `github.com/emersion/go-message` v0.18.2 | Builds on Go 1.17/ARM; handles charsets and nested parts |
| 6 | iCalendar and recurrence (T66) | `github.com/emersion/go-ical` and `github.com/teambition/rrule-go` v1.8.2 | They build on Go 1.17/ARM (they come as dependencies of `go-webdav`). `arran4/golang-ical` needs Go 1.20 and is out |
| 7 | CalDAV and WebDAV (T68) | `github.com/emersion/go-webdav` v0.7.0 (package `caldav`) | Builds on Go 1.17/ARM (IMAP + MIME + CalDAV together take 3.3 MB) |
| 8 | Microsoft Graph (T60, T64, T68) | **plain HTTP and JSON** (`env.HTTP`, `encoding/json`), no SDK | Microsoft's SDK is huge and needs a new Go; the REST API is small |
| 9 | On-premises Exchange (T61) | **EWS** as SOAP/XML (`encoding/xml`) written by us | There is no compatible Go library; the slice we need (mail, calendar, tasks) is small |
| 10 | Authentication (T53) | **OAuth2 device code flow** (RFC 8628), app password as the alternative | The reader has no browser: it shows the code and the address and you authorize on your phone. Needs *client IDs* registered by you |
| 11 | Tokens and passwords (T57) | Encrypted at rest with **AES-GCM**; key derived from an **optional PIN** and, without one, from a device identifier | Without a PIN the cipher only protects against copying the file elsewhere; **it is not a vault** (the reader has no keystore). The screen must say so |
| 12 | Synchronization (T54) | **Offline-first**: the UI reads only from the `store`; a queue of operations syncs when there is a network (`Env.HTTP` connects the Wi-Fi) | The reader's Wi-Fi is slow to connect and goes away; the screen never waits for the network |
| 13 | Integration between apps (T55) | **Intents** (`PBGO_INTENT`, next to `PBGO_LAUNCHER`) and data shared in the `store` | One app at a time: no in-process communication, only data on disk and one app opening another |
| 14 | Accounts (T52) | A shared `accounts` context: one account serves mail, tasks and calendar | Avoids registering the same account three times |

## What this changes in the tasks

- **T54** starts with a *spike*: run SQLite on ARM on the reader before building the `store` on it. If it fails, the fallback is our own
  file (JSON records with an in-memory index) behind the same port.
- **T59** uses no SMTP library: sending is `net/smtp` plus XOAUTH2.
- **T66** uses `go-ical` and `rrule-go` instead of writing the parser and the recurrence expansion from scratch.
- **T53** can only be finished after you register the OAuth applications (Microsoft Entra ID and Google).

## How each decision was checked

The libraries were **really built** in the SDK container (Go 1.17.13, `GOARCH=arm`, `CGO_ENABLED=1`), not just read off the `go`
directives of their `go.mod`: `go-smtp`'s `go 1.13` directive did not prevent the build failure. To repeat it, create a module with
`go 1.17`, import the library and run `docker run --rm -v <folder>:/app go-pocketbook-base-sdk build .`.
