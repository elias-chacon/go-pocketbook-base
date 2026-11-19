# 15. The Launcher

[← Architecture (DDD)](14-architecture.md) · [Index](README.md)

`apps/launcher` is the App Launcher of the ecosystem: the home screen of the apps built on this base. It is an app like any
other (it follows the [contract](13-apps-and-the-contract.md)) and the composition root of the `internal/launcher` context.

> **Status.** The home screen, running an app, the search and the GitHub catalog with one-tap install are done in code (the
> catalog and the download are not verified on the reader yet). The settings screen comes next.

## What it does today

- Lists the apps **installed** on the reader as a paged grid of **icon + title** (4 × 5 tiles per page). The apps are the ones
  that left a record in `applications/.pbgo` the first time they ran (see the contract); the Launcher does not list itself.
- Tapping a tile **runs the app**. The left and right keys turn the page (the arrows at the bottom do too).
- When the app asks to exit (the *Exit* menu, a Back key) the reader **comes back to the Launcher**; Home always goes to the
  system's home screen.
- A **search field** filters in real time (as the text changes) by title, description and id, ignoring case and accents. While
  searching, the answer has two sections: `-- Installed --` (the matching tiles, 3 rows per page) and `-- Available --` (the
  apps of the catalog that match and are not installed, up to 4 rows and "… and N more"). Without a catalog (or without
  network) the second section says why and the installed ones still answer.
- **Install with one tap.** The *Available* rows come from the **latest GitHub Release** of the repository (`elias-chacon/go-pocketbook-base`):
  every app is a `<id>.zip` (the `<id>.app` and its `app.json`) and a `catalog.json` lists them with manifest, size and SHA-256, so the list
  costs one small download. Tapping a row shows *Downloading…*, downloads the zip, checks its size and SHA-256, extracts the `<id>.app` into
  `applications/`, registers it on the home screen (with the
  icon the app published as `icon.png`, or the default one) and moves it to *Installed*. Each *Available* row shows that icon (downloaded once, kept in
  `applications/.pbgo/cache/icons`).
  The list is kept for 5 minutes. It needs Wi-Fi on; without it (or without a release) the section shows the error.
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
| domain | `internal/launcher/domain` | `Tile`, `TilesFrom` (translated, sorted ignoring case and accents), `Grid` and paging, `Matches`/`FilterTiles`/`FilterListings`, `Listing`, `SearchResult`, the `AppLauncher`, `Catalog` and `Installer` ports |
| application | `internal/launcher/application` | `Home`: the tiles of the installed apps, and opening one; `Search`: installed and available apps for a query |
| infrastructure | `internal/launcher/infrastructure/processlauncher` | starts an app by replacing the process |
| | `…/infrastructure/github` | the `Catalog`: the latest release's `catalog.json` and the apps it lists (cached 5 min) |
| | `…/infrastructure/iconcache` | downloads the catalog's icons (PNG) and keeps them on the reader |
| | `…/infrastructure/appinstaller` | the `Installer`: download the zip, check size and SHA-256, extract `<id>.app` through a temporary file, register |
| presentation | `internal/launcher/presentation` | the home screen, from the toolkit's widgets (`Image`, `Grid`, `Label`…) |
| composition root | `apps/launcher` | wires the platform to the context; loads the BMP icons |

The icons are the BMPs each app wrote when it registered itself; when one is missing the Launcher draws the default icon (the
initial of the title).

## Trying it on the reader

```powershell
.\build.ps1 -All                       # builds every app of apps/ (one version bump)
.\scripts\install.ps1 -Drive F: -All    # copies them to the reader
```

Each app registers itself the first time it runs (open it once from the *Applications* list), or use the [installer](16-installer.md), which registers them all while it copies. Then open the Launcher: it lists the others.

[← Architecture (DDD)](14-architecture.md) · [Index](README.md)
