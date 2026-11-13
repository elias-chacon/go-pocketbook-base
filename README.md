# go-pocketbook-base

Base template for writing **PocketBook (Linux ARM e-readers) apps in Go**, with a
UI drawn from scratch using vector graphics ([`fogleman/gg`](https://github.com/fogleman/gg)),
a self-registering launcher icon, and a reproducible Docker build. Fork it to start new apps.

The app opens a panel with "Hello, Golang!"; **Back** key or the on-screen **Exit** button closes it.

## How it works

| Piece | Where |
|---|---|
| Vector UI and launcher icon (pure Go, device independent) | `internal/ui` |
| Minimal cgo binding to `libinkview` (event loop, blit, refresh) | `internal/inkview` |
| First-run self-install (icons + desktop `view.json`) | `internal/install` |
| App entry point | `cmd/hello` |
| Desktop preview (renders the UI to PNG) | `cmd/preview` |
| SDK + patched Go toolchain image | `docker/Dockerfile` |

Rendering: the UI is rasterized by `gg` into an 8-bit grayscale buffer, then handed to
InkView with `DrawBitmap` + `FullUpdate`. The build targets the official
[PocketBook SDK](https://github.com/pocketbook/SDK_6.3.0) (branch `6.5`, `SDK-B288`) via
`arm-obreey-linux-gnueabi-clang`, `GOARCH=arm GOARM=7`, cgo, `-linkview`.

## Installation = copy one file

PocketBook has no installer, so the app installs itself. Copy `hello.app` to
`<device>/applications/` and open it from **Applications**. On its **first run** it:

1. renders its two launcher icons (normal + focused, 128x128 8-bit BMP) with the same vector
   code as the UI, into `applications/icons/`;
2. adds itself to the desktop config
   `system/profiles/default/config/desktop/view.json` (`system/config` is a link to it): a
   `U_<name>` definition in `"applications"` (relative paths, `icon` as `{ "path": ... }`, the
   format documented in the file's own `_comment`) and the key in the `"apps"` array of the
   `@General` group.

`view.json` is backed up (`view.json.bak-<timestamp>`) before being changed, the patched text is
validated as JSON, the file is replaced atomically, and the step is idempotent (nothing happens
if the app is already listed). To remove the icon, restore the backup or delete the `U_<name>`
entries. Tested against the real `view.json` of a PB741 (`internal/install/testdata`).

## Develop on the desktop

```sh
go test ./...                                         # includes the view.json patching tests
go run ./cmd/preview -w 1404 -h 1872                  # preview.png at PB741 size
go run ./cmd/preview -status "first run message"      # preview with a status line
```

## Build the app (needs Docker)

```sh
make dist          # Linux/macOS/WSL
.\build.ps1        # Windows PowerShell (uses Docker Desktop or Docker inside WSL)
```

Output: the single file `dist/applications/hello.app`. `.\scripts\install.ps1 -Drive F:` just
copies it to the device.

> **Target device:** developed and tested on a **PocketBook PB741-N-WW** (InkPad Color 2):
> 7.8" e-ink, 1404x1872, 24-bit canvas, firmware 6.x, with a 137 px system panel at the bottom.
> With the system panel enabled (the default, `panelType=1`) InkView reserves 137 px and offsets
> the app canvas, so a full-screen drawing shows up ~137 px off-center (confirmed with a photo).
> `Init` calls `inkview.DisablePanel()` (`SetPanelType(0)`) so the canvas is the whole screen;
> `Init` also logs the screen state before/after (`applications/hello.log`).
>
> The home screen picks up the new icon when it reloads its config, e.g. after unplugging the USB
> cable.

> **Status / caveats:** tested on a PB741 only. The cgo binding follows the SDK and the
> [`dennwc/inkview`](https://github.com/dennwc/inkview) reference. `cmd/hello` writes an event log
> (`applications/hello.log`) for on-device debugging; remove the `DebugLog` line when done.
> Other models/firmwares may use a different `view.json` layout; please open issues with yours.

## Creating a new app from this base

Copy `cmd/hello` to `cmd/<name>`, set the title/group in its `main.go`, implement `inkview.App`,
draw your UI in `internal/ui`, and run `make dist APP=<name>`. The icon and the desktop
registration come for free on first run.

## License

MIT
