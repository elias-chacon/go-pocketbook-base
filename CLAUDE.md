# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go base for **PocketBook e-reader apps** (Linux ARM): a from-scratch vector widget library (`internal/toolkit/widget`, drawn with `fogleman/gg`), a thin cgo binding to the reader's `libinkview` (`internal/device/inkview`), and an app that installs its own launcher icon on first run. Developed and tested on a **PB741 (InkPad Color 2, 1404x1872, firmware 6.x)**. The README and `docs/` are in Brazilian Portuguese / English; the code and comments are English.

## Commands

```sh
go test ./...                                   # everything (run this before declaring anything done)
go test ./internal/toolkit/widget -run TestTable -v     # a single test / package
go vet ./...                                    # also catches `unkeyed fields` on geom.Rect: use geom.R(x, y, w, h)
go run ./cmd/preview                            # render the demo pages to preview-<n>.png (-page N, -w, -h)
go run ./tools/docgen                           # redraw docs/components/*.png from the real widgets
go run ./tools/docsync                          # refill code blocks in docs/ from examples/  (-check to only verify)
go run ./tools/bumpversion build|minor|major    # edit .version
.\build.ps1 [-App name]                         # Windows build -> dist/applications/<app>.app (bumps BUILD)
make dist [APP=name]                            # same on Linux/macOS/WSL
.\scripts\sonar.ps1                             # tests with -coverpkg + SonarQube scan (needs SONAR_GLOBAL_TOKEN)
.\scripts\install.ps1 -Drive F: [-All]          # developer copy of the built .app(s) to a USB-mounted reader
go run ./tools/appicons                         # write apps/<id>/icon.png from the icons the apps draw in code (a test fails when stale)
go run ./tools/coverage                         # coverage that counts, from coverage.out (-badge file.json writes the shields.io badge)
go run ./tools/release                          # stage the payload, build the installers + SHA256SUMS into ./release (after build.ps1 -All)
go run ./cmd/pbinstaller [-drive F:\]           # the installer for people (needs the staged payload)
```

Only `internal/device/inkview` needs the device; on the desktop it compiles to stubs (build tags `linux && arm && cgo`), so all UI logic is testable on the PC. `apps/*` only build for the reader, through Docker.

## Architecture (DDD, enforced)

The repo is organized by **bounded contexts** under `internal/`, with one dependency rule that `internal/architecture` checks (`go test ./internal/architecture`): dependencies point inwards (infrastructure → application → domain), every context may use the kernel, a context never reaches into another's insides, and nothing in `internal/` imports `apps/`, `examples/`, `tools/` or `cmd/`. A new top-level folder in `internal/` must be declared in `architecture.Contexts` (and its rules in `architecture.Rules`). The map and the glossary are the package doc of `internal/architecture`; docs chapter 14 explains it.

- **`internal/kernel`**: shared kernel, depends on nothing. `geom` is Rect/Point/Insets, `version` is MAJOR.MINOR.BUILD.
- **`internal/toolkit`**: the UI library. **`widget`**: `Document` owns one element tree: `Layout()` -> `Render()`/`Update()` (grayscale `*image.Gray`) -> `HandlePointer()`. Every element embeds `Base` (ID, ClassName, Style, Hidden, Disabled, OnClick, Bounds...) and implements `Widget` (`Elem`, `Measure`, `Paint`); interactive ones also implement `PointerHandler`. One layout engine (`layout.go`) serves everything: a block is a vertical flex; measuring and arranging are pure functions over the tree. Custom elements outside the package must call `Base.Init(self)` and use `widget.Fire`. **`gfx`** draws in design units (`Canvas`, fonts, text, icons, images).
- **Coordinates are "design pixels"** (reference width 1404). `Document.Scale` converts to the real screen and touch is converted back, so drawing and touch share one space. Never offset or scale by hand.
- **Touch** (`dispatch.go`): down picks the deepest interactive, non-disabled element; move/up go to it (slider drag); a click is synthesized on release inside it, then `OnClick` bubbles up the tree. Modals and popups are drawn above; a touch outside a popup only closes it, a modal takes every touch. Text entry goes through the injected `Keyboard` interface (`FakeKeyboard` in tests/preview, InkView's keyboard on the device).
- **`internal/device`**: `inkview` (the only cgo; stubs on the desktop) `network` (the HTTP client for the reader's Linux, which has no resolv.conf and no IPv6: apps use `Env.HTTP`) and `desktop` (writes 128x128 8-bit BMP icons and text-patches the desktop `view.json`: backup, JSON validation, atomic replace, idempotent, repairs old relative paths; tested against a real firmware file in `testdata/`).
- **`internal/localization/i18n`**: pt-BR is primary, English the fallback (key -> language -> en -> pt-BR -> the key). Texts live in embedded `locales/*.json`.
- **`internal/platform`**: what an app is (`platform.RegisterOnDrive` does the first-run registration from the PC, for the installer). `domain` (Manifest, InstalledApp, Settings + ports), `application` (Runtime = device lifecycle with partial repaint, FirstRun), `infrastructure` (`jsonfiles` under `applications/.pbgo`, `reader` = InkView adapter + home screen registrar), and the package `platform` (composition root, the only import an app needs).
- **`internal/launcher`** is the Launcher context (`domain`: Tile/Grid/paging + the `AppLauncher` port; `application`: `Home`; `infrastructure/processlauncher`: starts an app with `execve`; `infrastructure/github`: the catalog (latest release's `catalog.json` + one `<id>.zip` per app); `infrastructure/appinstaller`: downloads the zip, checks SHA-256 and registers an app, setting `PBGO_LAUNCHER` so the app returns to the Launcher when it asks to exit; `presentation`: the screens). `apps/launcher` is its composition root.
- **`internal/installer`** is the Installer context (`domain`: Reader/Package/Outcome + ports; `application`: `Install`; `infrastructure`: `drives`, `embedded`, `readerfs`, `locale`; `presentation`: the console). `cmd/pbinstaller` is its composition root: it embeds `cmd/pbinstaller/payload` (staged by `tools/release`, not committed) and is released for Windows/Linux/macOS by `.github/workflows/release.yml` (a `v*` tag runs `ci.yml` first, then publishes one Release with every app, the installers and install commands in the notes; `ci.yml` is separate: vet, tests on Linux, coverage badge on `main`, apps built for the reader). It does not import the platform: the composition root passes it `platform.RegisterOnDrive`.
- **`internal/showcase`** is the hello app's content, shared by `apps/hello`, `cmd/preview` and tests (translated, with a language switch).
- **`apps/<id>`** are composition roots: `main.go` embeds `app.json` and calls `platform.Run`; `apps/hello` is the blueprint and `apps/contract_test.go` checks every app. They build to `dist/applications/<id>.app`.
- **`examples/`** holds the code samples of `docs/`, by context (`components`, `recipes`, `custom`, `translation`, `harness`); `tools/docsync` copies them into fenced blocks whose first line is `// examples/<package>/<file>.go` (the whole file) or `// examples/<package>/<file>.go#<Func>`, and `TestRepositoryDocsAreInSync` fails if a page is stale.

## Non-obvious constraints

- **Go 1.17 dialect** for anything that goes into the app (the Docker toolchain is 1.17.13): no generics, no `any`, no newer stdlib functions. Tools/tests run on newer Go but keep to 1.17 anyway.
- **Disable the system panel** (`inkview.DisablePanel()` in Init): otherwise InkView offsets the canvas ~137 px and drawing looks shifted.
- **Exit** must call both `CloseApp()` and `LeaveInkViewMain()`.
- **Desktop entry** (`system/profiles/default/config/desktop/view.json`; `system/config` is a link): the `path` of `U_<app>` must be **absolute**, icons may be relative, and the key must also be in a group's `apps` array. The home screen only reloads it when the USB cable is unplugged.
- **Fonts** (Go Regular/Bold) lack glyphs like "✓": draw an icon with `Canvas.Icon` instead.
- **Layout gaps to know**: no scrolling, `Pct` only for width, one page must fit the screen (the demo test fails on overflow).
- **Touch-accuracy rule**: a touch at the center of every interactive element must reach that element; `internal/showcase/demo_test.go` and `examples/components/components_test.go` enforce it, plus a 56 px minimum target.

## Environment quirks (Windows + WSL)

- Docker exists only inside WSL; `build.ps1` falls back to `wsl -e docker`. The user's PowerShell profile defines a broken `docker` function (drops arguments), so scripts only accept a real `docker` executable.
- Git Bash rewrites `/mnt/d/...` paths passed to `wsl`: run WSL commands through `pwsh -File x.ps1`. In PowerShell quote `"-coverprofile=coverage.out"`.
- Long multi-file shell heredocs sometimes fail to parse in the Bash tool; write files with the editor tools instead. Editing text files with `perl -pi` on UTF-8 content can double-encode it; use Node or the editor tools. Files checked out by Git are CRLF, so normalize `\r\n` before string matching.
- SonarQube runs on the Windows host (`localhost:9000`, token in `SONAR_GLOBAL_TOKEN`); containers reach it through the WSL default-route IP (handled by the scripts).

## Working rules for this repo (from the maintainer)

- Versioning is `.version` = `MAJOR.MINOR.BUILD` (starts 1.0.0): every build bumps BUILD, a **finished task** bumps MINOR (BUILD -> 0), only big changes bump MAJOR. Tasks are the checkbox list in the README.
- **When a task is finished**: review and update the docs (`docs/en` and `docs/pt-br`, README tasks/architecture/links, `docgen` pictures, `docsync` + `-check`), mark the task, bump MINOR, run tests (and Sonar when code changed; the goal is 0 smells, quality gate OK), then **commit without being asked**. Do not push. Commit messages end with the `Co-Authored-By` trailer.
- **Never create git worktrees.**
- Apply SOLID/KISS/DRY pragmatically. Say plainly when something is untested on the device, and ask before changing anything on a connected reader outside its `applications/` folder (diagnostic code must be announced and removed).
