# 16. The installer and the releases

[← The Launcher](15-launcher.md) · [Index](README.md)

Nobody who just wants the apps should need Go, Docker or PowerShell. Each release publishes an **installer** for Windows,
Linux and macOS: one program, with an icon, that carries the apps inside itself and puts them on the PocketBook.

## For the person who just wants the apps

1. Open the [Releases](https://github.com/elias-chacon/go-pocketbook-base/releases) page and download the file for your
   computer: `pbgo-installer_<version>_windows_amd64.zip`, `…_linux_amd64.tar.gz`, `…_darwin_arm64.tar.gz` (Apple
   silicon) or `…_darwin_amd64.tar.gz` (Intel Mac). Unpack it.
2. Connect the PocketBook with the USB cable and choose the PC connection mode (*PC Link*) on the reader.
3. Double-click **pbgo-installer**. It finds the reader by itself, copies the apps (the Launcher and the others) and puts their
   icons on the reader's home screen. If more than one reader is connected it asks which one.
4. Eject the reader safely and unplug the cable: the icons appear in the Applications list, in the *General* group.

The installer speaks the computer's language (Portuguese or English). If it does not find the reader it says what to do, and
`pbgo-installer -drive F:\` (or a folder) installs on the drive you point at.

| System | What you may see | What to do |
|---|---|---|
| Windows | *"Windows protected your PC"* (the program is not signed) | **More info → Run anyway** |
| macOS | *"cannot be opened because the developer cannot be verified"* | right-click the file → **Open** → **Open** |
| Linux | nothing | `tar xzf` keeps the executable bit; run `./pbgo-installer` (from a file manager, "Run") |

## What it does

For every app it carries: it writes `applications/<id>.app` on the reader (through a temporary file, so an interrupted copy never
leaves half an app) and registers the app **the same way the app does on its first run**: the two icons, the entry in
`view.json` (with a backup) and the record in `applications/.pbgo` that the Launcher reads. The registration is the platform's
(`platform.RegisterOnDrive`), so there is one implementation: an app copied by hand still registers itself when it runs.

## The code (context `installer`)

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/installer/domain` | `Reader`, `Package`, `Outcome` and the ports `ReaderFinder`, `PackageSource`, `Deployer` |
| application | `internal/installer/application` | `Install`: find the readers, install every package (one failure does not stop the others) |
| infrastructure | `…/infrastructure/drives` | finds a PocketBook: drive letters on Windows, `/Volumes` on macOS, `/media`, `/run/media`, `/mnt` on Linux; a drive is a reader when it has the reader's `view.json` |
| | `…/infrastructure/embedded` | the apps inside the executable (`embed.FS` with `<id>.app` + `<id>.json`) |
| | `…/infrastructure/readerfs` | writes the app to the reader and calls the registration |
| | `…/infrastructure/locale` | the computer's language |
| presentation | `internal/installer/presentation` | the console conversation (waits for Enter at the end, so a double-clicked window does not vanish) |
| composition root | `cmd/pbinstaller` | wires everything, embeds `payload/`, knows each app's icon |

The context does not depend on the platform: the composition root hands it the registration function, and the architecture test
(chapter 14) keeps it that way.

## How a release is made

```powershell
.\build.ps1 -All          # the apps for the reader (dist/applications)
go run ./tools/release    # stages the payload and builds the installers into ./release
```

`tools/release` copies each `dist/applications/<id>.app` and its `apps/<id>/app.json` into `cmd/pbinstaller/payload`, builds the
installer for Windows (amd64), Linux (amd64) and macOS (amd64, arm64) with the version in the binary, gives the Windows one the
Launcher's icon (a resource file made by `github.com/akavel/rsrc`; without network it keeps the default icon), packs each as a zip
or tar.gz, copies the raw `<id>.app` files (the Launcher downloads those) and writes `SHA256SUMS`.

The workflow `.github/workflows/release.yml` does all of this when a tag `vMAJOR.MINOR.BUILD` is pushed (the tag must equal
`.version`) and publishes the files on the GitHub Release.

> **Status.** The installer is tested on the computer (finding, copying and registering use temporary folders), but it has not
> been run against a real reader through the cable, and the release workflow has not run yet.
