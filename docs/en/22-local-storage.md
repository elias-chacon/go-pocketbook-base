# 22. Local storage

[← Tasks: domain](21-tasks-domain.md) · [Index](README.md)

The `internal/storage` package is the local database the mail, tasks and calendar apps share: **one SQLite file** (`applications/.pbgo/data.db`)
with the cached messages, events and tasks. The screens read from the reader's own disk and **never wait for the network** (*offline-first*:
synchronization updates the database, not the screen).

## What it does

- It is **pure Go** (`modernc.org/sqlite`, decided in chapter 17): it builds with the SDK's Go 1.17 for ARM (verified) and the tests run on
  any computer, without a C compiler. The cost is about **5.5 MB** more in every app that uses it.
- It **opens with the settings of a small reader**: a page cache of about 2 MB, a rollback journal that is **truncated** (not WAL, because the
  reader's storage is a FAT volume, where the write-ahead log and its shared memory are not safe), foreign keys on and a single connection (one
  app runs at a time).
- **Migrations per module** (`Migrate`): each context (`mail`, `calendar`, `tasks`) has its steps numbered 1, 2, 3, without gaps. The ones not run
  yet execute in order, each in a transaction together with the version record, so **a failing step leaves nothing half done**. It is safe to call at
  every open. A database written by a **newer** program is refused (`ErrDowngrade`), instead of being touched by one that does not understand it.
- **Erasing an account** (`EraseAccount`, or `Erase`, which is the Accounts context's `Eraser` port): each context declares its tables that have an
  `account_id` column (`RegisterAccountTable`), and removing the account deletes its rows in all of them, **all or nothing**.
- `Check` runs SQLite's integrity check, and `Tables` lists the tables (for diagnostics).

The database does not know what a message or an event is: each context writes its own tables and queries on top of `DB()`.

## What has not been verified yet

**The real reader.** SQLite is verified on **emulated ARMv7** (all the storage tests pass under `qemu-arm` with the SDK's Go 1.17: script ARM, já removido),
but it has not run on the PocketBook itself, and the emulation cannot check the **FAT volume** the database will live on. It is the first thing to
validate when an app uses it; if it fails, the fallback is our own file behind the same port.

## The code

| Package | Holds |
|---|---|
| `internal/storage` | `Open`, `Migrate`, `Version`, `RegisterAccountTable`, `EraseAccount`/`Erase`, `Check`, `Tables` |

It depends only on the shared kernel and the driver; the architecture test (chapter 14) checks it.
