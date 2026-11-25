# 15. The Launcher

[← Architecture (DDD)](14-architecture.md) · [Index](README.md)

`apps/launcher` is the App Launcher of the ecosystem: the home screen of the apps built on this base. It is an app like any
other (it follows the [contract](13-apps-and-the-contract.md)) and the composition root of the `internal/launcher` context.

> **Status.** The home screen with every app of the catalog, running an app, install by tap, update by checksum, hide and remove are done in code (the
> catalog, the download, the update, the removal and the long press are not verified on the reader yet).

## What it does today

- Lists **every app of the ecosystem** as a paged grid of **icon + title** (4 × 5 tiles per page): the apps **installed** on the reader (the ones
  that left a record in `applications/.pbgo` the first time they ran) **and the apps that only the catalog has**, taken from the **latest GitHub
  Release** (`catalog.json`, kept on the reader in `applications/.pbgo/cache/catalog.json`, so they show at once and offline; the Launcher asks
  GitHub in the background when it opens and redraws when the answer comes). The Launcher does not list itself.
- A tap on an installed tile **runs the app**; a tap on one that is **not installed** (it says *Tap to install*) **installs it**: *Downloading…*,
  the zip, the size and SHA-256 check, `<id>.app` extracted into `applications/` through a temporary file and registered on the home screen (with the
  icon the app published as `icon.png`, or the default one). The left and right keys turn the page (the arrows at the bottom do too).
- A **long press (2 seconds)** on a tile opens the window of the app: **Update** (offered only when the app is installed and the **SHA-256 of its
  file differs from the `app_sha256` of the release**: the checksum decides, never the version), **Install** (when it is missing), **Hide from the home screen**
  (installed or not) and **Remove from the reader**, which after a confirmation deletes the app and everything it left: its entry in `view.json` and its
  icons, the program, its record and log, the files its manifest lists as `data` and, for apps installed by older versions whose record lists nothing, every file of `applications/.pbgo` named after the app (`<id>.…`, `<id>-…`, `<id>_…`, e.g. its database), its tables in the shared database (the file goes when no
  other app has tables in it) and the files it `shares` with other apps (the accounts) only when no other installed app shares them.
- When the app asks to exit (the *Exit* menu, a Back key) the reader **comes back to the Launcher**; Home always goes to the
  system's home screen.
- A **search field** filters the whole grid in real time (as the text changes) by title, description and id, ignoring case and accents.
- **Settings** also lists the apps: **Manage apps** (a `Table` of the visible apps, installed or not; select one and open its window, as the long press does)
  and **Hidden apps** (a `Table` of the hidden ones, with **Show the selected app again**). The hidden ids are kept in `launcher.json`.
- **Settings** also has **About** (what the Launcher is, its version and the repository of the releases) and **System information** (a `Table`: the Launcher version, the
  device model, the screen, the free and total storage, the folder of the apps, the language and how many apps are installed, listed and hidden). The settings
  cards do not fit the screen, so they are in a `Scroll` (drag the page or the bar on the right).
- **Settings** (the *Settings* button in the footer, or the Menu key; left leaves without saving): the **grid** (columns and rows, 1 to 8), the **icon size**
  (96 to 240), the **language** (shared by every app: the others read it when they start) and a **background image**, chosen among the pictures
  (PNG, JPG, GIF, BMP) the person copied to `applications/backgrounds` on the reader. The preferences are kept in `applications/.pbgo/launcher.json`; a
  picture is shrunk to the screen when loaded, and the tiles keep a white background so they stay readable over it.
- **The latest catalog comes with a progress bar**: as soon as the Launcher opens, a bar at the top of the grid follows the fetch (the release, `catalog.json`, then one step per icon; `domain.Progress`, read on the event loop every 0.4 s while the network runs in a goroutine) and goes away when the apps arrive. With no apps installed and no network the screen says to connect to Wi-Fi, it does not ask to copy files. Not yet seen on the reader.
- **Registering replaces the home screen entry**: every registration (the app at its own start, an install or update from the Launcher, the PC installer) rewrites the app's `view.json` entry whole, in the same place (path, title, icons), puts it back in the group if it had lost its place, and writes the icon files again when they differ, so what an older version left behind is not kept. The catalog's icon cache keeps one picture per app (`<id>.png` and the address it came from) and replaces it when the release publishes another, and drops the ones the catalog no longer lists. Not yet seen on the reader.
- **Installing and updating have a window**: Install/Update (from the manage window or a tap on a missing app) opens a window with the app name, a bar that follows the bytes of the download (`github.GetWithProgress` -> `domain.Progress`), and at the end what happened, the error included, until it is closed. The download runs in a goroutine and the event loop polls it every 0.3 s, so the screen is not frozen. Not yet seen on the reader.
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
| domain | `internal/launcher/domain` | `Tile`, `TilesFrom` (translated, sorted ignoring case and accents), `Grid` and paging, `AppsOf` (installed + catalog, sorted), `Visible`/`Hidden`, `Matches`/`FilterTiles`, `Listing` (with `AppSHA256`), `Preferences` (grid, icon size, background and hidden apps, always inside their bounds), the `AppLauncher`, `Catalog`, `Installer`, `Uninstaller` and `PreferencesStore` ports |
| application | `internal/launcher/application` | `Home`: every app as a tile (`All`), and opening one |
| infrastructure | `internal/launcher/infrastructure/processlauncher` | starts an app by replacing the process |
| | `…/infrastructure/github` | the `Catalog`: the latest release's `catalog.json` and the apps it lists (cached 5 min) |
| | `…/infrastructure/prefsfile` | the `PreferencesStore`: `applications/.pbgo/launcher.json` |
| | `…/infrastructure/backgrounds` | lists the pictures of `applications/backgrounds` |
| | `…/infrastructure/iconcache` | downloads the catalog's icons (PNG) and keeps them on the reader (`Cached` never uses the network) |
| | `…/infrastructure/catalogcache` | keeps the last catalog on the reader and gives it back when the network fails |
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
