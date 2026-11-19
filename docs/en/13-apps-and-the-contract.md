# 13. Apps and the contract

[← Internationalization](12-internationalization.md) · [Index](README.md)

Every app of this project (the Launcher and its plugins) lives in **one folder under `./apps/`** and follows the same
contract. `apps/hello` is the blueprint: it is a real, working app and the template for every new one.

## Opening another app: intents

The reader runs one app at a time, so apps never talk to each other while running. An app that wants another to do something calls
`env.Send(appID, "tasks.create", data)`: the platform replaces the running program with that app (as the Launcher does) and hands it an
**intent**. The other app finds it in `env.Intent` when it starts (`Action`, `From` and a small `Data` map: pass ids and let it read
the rest from the shared store; an intent is bounded to 48 KB because it travels in the environment). When the app that was opened asks
to exit, the reader **goes back to the app that asked** (not to the Launcher), and the intent is removed from the environment so the app
it returns to does not take it for a request of its own. Home always leaves.

An app declares the actions it handles in `app.json`, `"intents": ["tasks.create", "tasks.show"]`, each named `app.verb`; `Send` fails
with `platform.ErrNoSuchApp` or `platform.ErrNotHandled`, without leaving the app, when the target is not installed or does not handle it.

## One folder, one file

```
apps/<id>/
  app.json     the manifest (embedded in the program)
  icon.png     (optional) the icon, 128x128: the release publishes it so the Launcher shows the app before it is
               installed, and the installers use it as the icon on the home screen
  main.go      embeds app.json and calls platform.Run
  *.go         the app: a factory that builds the screen
  locales/     (optional) translations, pt-BR and en
```

Building it produces **a single file**, `<id>.app`. Installing is copying that file to the reader's `applications`
folder; nothing else. An app is installed, updated and run on its own: the Launcher is a convenience, not a requirement.

## The manifest (`app.json`)

```json
{
  "id": "hello",
  "title":       { "pt-BR": "Hello Go", "en": "Hello Go" },
  "description": { "pt-BR": "Vitrine de todos os componentes…", "en": "Showcase of every component…" },
  "group": "@General"
}
```

| Field | Rule |
|---|---|
| `id` | lower-case letters, digits, `-` or `_`, starting with a letter; **equal to the folder name** (it names `<id>.app`) |
| `title`, `description` | texts by language; **pt-BR is required** (the primary language), `en` is the fallback |
| `group` | desktop group of the icon (default `@General`) |

`platform.ParseManifest` validates it and `TitleIn(lang)` picks the text like [i18n](12-internationalization.md) does.

## The lifecycle

1. `main` calls `platform.Run(config, factory)`.
2. **First run:** the platform writes the icons, adds the app to the reader's desktop (`view.json`, with a backup) and writes
   a **record** `applications/.pbgo/<id>.json` (the manifest plus version, path of the file and of the icon): this is what the
   Launcher reads to know what is installed. It runs on every start, so the record always has the current version.
3. The platform reads the shared **settings** (`applications/.pbgo/settings.json`: today the language) and starts InkView.
4. Once the screen size is known the platform calls your **factory** with an `platform.Env`: screen size, system keyboard,
   `Exit`, the first-run notice, the manifest, the language, the version and `HTTP`, the network client that works on the reader (its Linux has no `resolv.conf` and no IPv6, which the standard client needs; `internal/device/network`). Use `env.HTTP` for any request. It returns an `platform.App`, which only has to
   give its `*widget.Document`.
5. From then on the platform does the plumbing: touch, the system keyboard, **partial repaint** (full refresh every 10 updates
   to clear e-ink ghosting), and the keys: left and Menu close a popup, then a modal (`doc.HandleBack()`); keys the platform does not use go
   to the app if it implements `platform.KeyHandler`.

```go
// apps/hello/main.go#main
func main() {
	err := platform.Run(platform.Config{
		Manifest: manifest,
		// An app may draw its own launcher icon; without Icon the platform draws one
		// from the title (platform.DefaultIcon).
		Icon: func(size int, focused bool) image.Image { return showcase.RenderIcon(size, focused) },
	}, newHello)
	if err != nil {
		log.Fatal(err)
	}
}
```

```go
// apps/hello/hello.go#newHello
// newHello is the platform.Factory: the platform calls it once the screen size
// is known, with everything the app may need in env.
func newHello(env platform.Env) platform.App {
	return &hello{demo: showcase.New(env.Width, env.Height, showcase.Config{
		Version:  env.Version,
		Keyboard: env.Keyboard,
		Exit:     env.Exit,
		Notice:   env.Notice,
		Lang:     env.Lang,
		// The language chosen here becomes the language of every app.
		OnLangChange: func(lang string) { _ = env.Settings.Save(platform.Settings{Lang: lang}) },
	})}
}
```

The app's own `Key(key platform.Key) bool` method (in `hello.go`) is how it reacts to the keys the platform leaves alone.

## Where the platform's code lives

`internal/platform` is a bounded context with three layers (see [Architecture (DDD)](14-architecture.md)):
**domain** (`Manifest`, `InstalledApp`, `Settings` and the ports), **application** (`Runtime`, `FirstRun`) and
**infrastructure** (`jsonfiles`, `reader`). An app imports only the package `platform`, which wires them.

## The language is shared

`env.Settings.Load().Lang` is the language chosen for the whole ecosystem; the platform hands it to your factory
(`env.Lang`). When the user changes it inside an app, save it with `env.Settings.Save` so every other app starts in it
(`apps/hello` does this in `OnLangChange`).

## Create an app

1. `cp -r apps/hello apps/myapp`
2. Edit `apps/myapp/app.json`: `"id": "myapp"`, titles and descriptions in pt-BR and en.
3. Replace the screen in `newHello` (rename it) and drop what you do not need.
4. `go test ./...`: the **contract test** (`apps/contract_test.go`) checks every folder of `apps/`: a valid manifest,
   the id equal to the folder, `main.go` embedding `app.json` and calling `platform.Run`, complete `locales/` and, if there is an
   `icon.png`, a 128x128 PNG. The apps of this repository draw their icon in code; `go run ./tools/appicons` writes the file from
   that drawing (a test fails when it is out of date).
5. `.\build.ps1 -App myapp` and copy `dist/applications/myapp.app` to the reader.

Write the texts in **Portuguese (pt-BR)** with English as the fallback, through [`internal/localization/i18n`](12-internationalization.md).

[← Internationalization](12-internationalization.md) · [Index](README.md)
