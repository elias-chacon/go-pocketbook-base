# 11. Troubleshooting

[← Testing](10-testing.md) · [Index](README.md)

Problems seen while building this project on a PB741, with the cause and the fix.

## On the reader

| Symptom | Cause | Fix |
|---|---|---|
| The icon does not appear after the first run | The home screen only reloads its configuration when the USB cable is unplugged (or on restart) | Unplug the cable (the first-run message says so) |
| The icon shows but tapping it returns to the home screen | The desktop entry used a **relative** launch path (`applications/x.app`) | Use the current build: it writes the absolute `/mnt/ext1/applications/x.app` and **repairs** an old entry on the next run (open the app from the Applications list) |
| The name appears under an empty icon | The icon files are missing (`applications/icons/`) | Open the app again: it recreates the icons |
| The picture is shifted down ~137 px, and the bottom touches the edge | The system panel is on and InkView offsets the canvas | Call `inkview.DisablePanel()` in `Init` (the template does) |
| The Exit/Quit does nothing | `CloseApp()` alone did not end the app | `inkview.Exit()` calls `CloseApp()` **and** `LeaveInkViewMain()` |
| A tap lands "a bit off" the drawing | Drawing and touch used different coordinates | Do not scale or offset by hand: use design pixels and `doc.HandlePointer` with the device coordinates |
| Typing does nothing | No keyboard was given to the document | `doc.Keyboard = env.Keyboard` (the platform gives it to your factory, see `apps/hello`) |
| Dragging a slider is slow | The e-ink redraws the whole screen | The platform coalesces repaints, skips move events and sends only the changed areas to the panel (partial refresh) |
| A symbol (✓, →) shows as a box | The built-in fonts lack that glyph | Draw an icon (`c.Icon`) or use another symbol |
| Text ends with "…" | It does not fit its box | Give the element more width (`FlexGrow`, `Width`) or shorter text |
| The app does nothing at all when opened | It crashed before drawing | Read `applications/hello.log` (create `applications/.pbgo/debug` for the full trace) |

### Reading the event log

Every app writes `applications/<id>.log` (the platform sets it up). By default it holds only **errors** (`platform.Errorf`: a failed
download, a failed registration), and it starts again when it passes 64 KB. To trace everything, create an empty file
`applications/.pbgo/debug` on the reader (or run with `PBGO_DEBUG=1`): the log then also has every InkView event and the app's steps
(`platform.Logf`). Connect the reader, open the file and look for `main start` (a new run), `info before/after` (screen and panel state)
and `register:` (the first-run result). Never log a password, a token or the text of a message.

## Building

| Symptom | Cause | Fix |
|---|---|---|
| `docker` prints its **help** and no `.app` is made | A PowerShell profile function named `docker` that drops the arguments (for example `function docker() { wsl -- docker }`) | `build.ps1` ignores functions and aliases; or fix the profile: `function docker { wsl docker @args }` |
| `docker: command not found` on Windows | Docker lives only inside WSL | `build.ps1` falls back to `wsl -e docker` automatically |
| `apt-get` returns 404 for `stretch` | The SDK image is based on Debian stretch, which is archived | The `Dockerfile` already points to `archive.debian.org` |
| `ft2build.h: file not found` | The FreeType header is outside the default include path | Already handled in the cgo flags of `internal/device/inkview` |
| A path like `/mnt/d/...` becomes `C:/Program Files/Git/...` | Git Bash rewrites paths it passes to `wsl` | Run WSL commands from PowerShell, or use the `.ps1` scripts |
| `-coverprofile=coverage.out` fails in PowerShell | PowerShell splits the argument at the dot | Quote it: `"-coverprofile=coverage.out"` |
| The build fails on `[T any]` or `any` | The image's compiler is Go 1.17 | Do not use generics or newer library functions in app code |
| "The build finished but … was not created" | Docker ran but produced nothing | Read the Docker output above that message |

## Tests and documentation

| Symptom | Fix |
|---|---|
| `TestRepositoryDocsAreInSync` fails | A page of `docs/` is out of date: run `go run ./tools/docsync` |
| The component pictures look old | Run `go run ./tools/docgen` |
| "LF will be replaced by CRLF" warnings from Git | Harmless on Windows (line-ending conversion) |
| Sonar returns 401 | `SONAR_GLOBAL_TOKEN` is missing or wrong |

## Still stuck?

1. Run `go test ./...` — a failing test usually names the problem.
2. Render the screen to a PNG (`harness.SavePNG`) to see what the layout produced.
3. Print `element.Bounds` of the elements that look wrong.
4. Open an issue with the model, the firmware version and the contents of the log file.

[← Testing](10-testing.md) · [Index](README.md)
