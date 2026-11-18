# 2. Installation

[← Getting started](01-getting-started.md) · [Index](README.md) · [Next: Build and versioning →](03-build-and-versioning.md)

## Development environment

| Tool | Version | Used for |
|---|---|---|
| Go | 1.17+ (any newer works on your PC) | tests, preview, tools |
| Git | any | clone and commits |
| Docker | any recent | building for the reader (ARM + PocketBook SDK) |
| WSL 2 (Windows only) | optional | Docker inside Linux, if you do not use Docker Desktop |

> **Go 1.17 rule.** The toolchain inside the Docker image is Go 1.17.13 (the last one that works with
> the SDK's old linker). Code that goes into the app must not use generics (`[T any]`), `any`, or
> newer library functions. Your PC can run newer Go, but write the app code in the 1.17 dialect.

### Check your setup

```sh
go version          # go1.17 or newer
docker version      # (or: wsl -e docker version)
go test ./...       # everything passes
```

## Installing the app on the reader

PocketBook has **no installer**: an app is just a file in the `applications` folder. This project goes
one step further and the app **installs its own icon**.

### 1. Copy the file

1. Build it ([Build and versioning](03-build-and-versioning.md)) — you get one file,
   `dist/applications/hello.app`.
2. Connect the reader by USB and choose **PC link**; it shows up as a drive (for example `F:`).
3. Copy the file to `<drive>/applications/`, or let the script do it:

   ```powershell
   .\scripts\install.ps1 -Drive F:              # the demo (hello)
   .\scripts\install.ps1 -Drive F: -App myapp   # your own app
   ```


4. Eject the reader safely.

### 2. Open it once

Open **Applications** on the reader and tap the app (it is listed by its file name, for example
*hello*). On this **first run** the app:

1. draws its two icons (normal and "pressed", 128×128, 8-bit BMP) with the same vector code as the
   screens and saves them in `applications/icons/`;
2. registers itself in the reader's desktop configuration, adding an entry for it and listing it in
   the **General** group;
3. shows a message at the bottom of the screen telling you what it did.

The desktop configuration file is `system/profiles/default/config/desktop/view.json`
(`system/config` is a link to the same folder). Before changing it the app saves a **backup**
(`view.json.bak-<date-time>`), checks that the new text is valid JSON and replaces the file in one step.
Running the app again changes nothing; if an entry from an older version has a wrong path, it fixes it.

### 3. Unplug the cable

The home screen only reads its configuration again when the USB cable is unplugged (or the reader
restarts). After that the icon **Hello Go** is in the *General* group of the Applications screen.

```text
copy the .app  →  open it once  →  unplug USB  →  icon on the home screen
```

## Updating

Copy the new `.app` over the old one. The app notices it is already registered and leaves the
configuration alone. The footer of the demo shows the running version (`vMAJOR.MINOR.BUILD`), so you
can confirm which build is on the reader.

## Uninstalling

1. Delete `applications/hello.app` and `applications/icons/hello.bmp` / `hello_f.bmp`.
2. Remove the entry from the desktop: restore the backup
   `system/profiles/default/config/desktop/view.json.bak-<date-time>` over `view.json`, or delete the
   `U_hello` entries by hand (one in `"applications"`, one in the `"apps"` list of a group).
3. Unplug the cable so the desktop reloads.

> The first backup of a reader that never ran the app is the original factory file, so it is the one to
> restore if you want everything exactly as it was.

## What is written on the reader

| Path | What | When |
|---|---|---|
| `applications/<name>.app` | the app | you copy it |
| `applications/icons/<name>.bmp`, `<name>_f.bmp` | launcher icons | first run |
| `system/profiles/default/config/desktop/view.json` | desktop entry | first run |
| `system/profiles/default/config/desktop/view.json.bak-*` | backup before the change | first run |
| `applications/hello.log` | event log for debugging (the demo only) | every run |

Next: [Build and versioning →](03-build-and-versioning.md)
