# 15. The Launcher

[← Architecture (DDD)](14-architecture.md) · [Index](README.md)

`apps/launcher` is the App Launcher of the ecosystem: the home screen of the apps built on this base. It is an app like any
other (it follows the [contract](13-apps-and-the-contract.md)) and the composition root of the `internal/launcher` context.

> **Status.** The home screen (the grid of installed apps) and running an app are done. Search with the
> *Installed / Available* sections, the GitHub catalog and the settings screen come next.

## What it does today

- Lists the apps **installed** on the reader as a paged grid of **icon + title** (4 × 5 tiles per page). The apps are the ones
  that left a record in `applications/.pbgo` the first time they ran (see the contract); the Launcher does not list itself.
- Tapping a tile **runs the app**. The left and right keys turn the page (the arrows at the bottom do too).
- When the app asks to exit (the *Exit* menu, a Back key) the reader **comes back to the Launcher**; Home always goes to the
  system's home screen.
- Texts in Portuguese (pt-BR) with English as the fallback, in the language chosen in the shared settings.

## How an app is run, and how it comes back

The reader has one screen and one InkView session, so the Launcher does not run the app next to itself: it
**replaces itself with it** (`execve`), in the same process, and tells the app who started it through the environment variable
`PBGO_LAUNCHER` (the path of the Launcher). When the app asks to exit, the platform sees the variable and starts the Launcher
back, again in place. An app started from the home screen has no variable and just ends; pressing Home never goes back,
because that is the system's decision, not the app's. The Launcher never returns to itself (it inherits the variable from the app that
came back).

```
Launcher ──exec──► app          (PBGO_LAUNCHER=/…/launcher.app)
   ▲                 │
   └───────exec──────┘          when the app asks to exit
```

## The code (context `launcher`)

| Layer | Package | Holds |
|---|---|---|
| domain | `internal/launcher/domain` | `Tile`, `TilesFrom` (translated, sorted ignoring case and accents), `Grid` and paging, the `AppLauncher` port |
| application | `internal/launcher/application` | `Home`: the tiles of the installed apps, and opening one |
| infrastructure | `internal/launcher/infrastructure/processlauncher` | starts an app by replacing the process |
| presentation | `internal/launcher/presentation` | the home screen, from the toolkit's widgets (`Image`, `Grid`, `Label`…) |
| composition root | `apps/launcher` | wires the platform to the context; loads the BMP icons |

The icons are the BMPs each app wrote when it registered itself; when one is missing the Launcher draws the default icon (the
initial of the title).

## Trying it on the reader

```powershell
.\build.ps1 -All                       # builds every app of apps/ (one version bump)
.\scripts\install.ps1 -Drive F: -All    # copies them to the reader
```

`install.ps1` copies the apps **and registers them** (icons, the `view.json` entry and the record the Launcher reads: `tools/register`). Eject, **unplug the cable** and the icons show on the home screen; open the Launcher and it lists the others. An app copied by hand still registers itself on its first run.

[← Architecture (DDD)](14-architecture.md) · [Index](README.md)
