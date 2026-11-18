# 13. Apps and the contract

[← Internationalization](12-internationalization.md) · [Index](README.md)

Every app of this project (the Launcher and its plugins) lives in **one folder under `./apps/`** and follows the same
contract. `apps/hello` is the blueprint: it is a real, working app and the template for every new one.

## One folder, one file

```
apps/<id>/
  app.json     the manifest (embedded in the program)
  main.go      embeds app.json and calls appkit.Run
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

`appkit.ParseManifest` validates it and `TitleIn(lang)` picks the text like [i18n](12-internationalization.md) does.

## The lifecycle

1. `main` calls `appkit.Run(config, factory)`.
2. **First run:** the kit writes the icons, adds the app to the reader's desktop (`view.json`, with a backup) and writes
   a **record** `applications/.pbgo/<id>.json` (the manifest plus version, path of the file and of the icon): this is what the
   Launcher reads to know what is installed. It runs on every start, so the record always has the current version.
3. The kit reads the shared **settings** (`applications/.pbgo/settings.json`: today the language) and starts InkView.
4. Once the screen size is known the kit calls your **factory** with an `appkit.Env`: screen size, system keyboard,
   `Exit`, the first-run notice, the manifest, the language and the version. It returns an `appkit.App`, which only has to
   give its `*widget.Document`.
5. From then on the kit does the plumbing: touch, the system keyboard, **partial repaint** (full refresh every 10 updates
   to clear e-ink ghosting), and the keys: left and Menu close a popup, then a modal (`doc.HandleBack()`); keys the kit does not use go
   to the app if it implements `appkit.KeyHandler`.

```go
// apps/hello/main.go#main
func main() {
	err := appkit.Run(appkit.Config{
		Manifest: manifest,
		// An app may draw its own launcher icon; without Icon the kit draws one
		// from the title (appkit.DefaultIcon).
		Icon: func(size int, focused bool) image.Image { return ui.RenderIcon(size, focused) },
	}, newHello)
	if err != nil {
		log.Fatal(err)
	}
}
```

```go
// apps/hello/hello.go#newHello
// newHello is the appkit.Factory: the kit calls it once the screen size is
// known, with everything the app may need in env.
func newHello(env appkit.Env) appkit.App {
	return &hello{demo: demo.New(env.Width, env.Height, demo.Config{
		Version:  env.Version,
		Keyboard: env.Keyboard,
		Exit:     env.Exit,
		Notice:   env.Notice,
		Lang:     env.Lang,
		// The language chosen here becomes the language of every app.
		OnLangChange: func(lang string) { _ = appkit.SaveSettings(env.Root, appkit.Settings{Lang: lang}) },
	})}
}
```

The app's own `Key(code int) bool` method (in `hello.go`) is how it reacts to the keys the kit leaves alone.

## The language is shared

`appkit.LoadSettings(root).Lang` is the language chosen for the whole ecosystem; the kit hands it to your factory
(`env.Lang`). When the user changes it inside an app, save it with `appkit.SaveSettings` so every other app starts in it
(`apps/hello` does this in `OnLangChange`).

## Create an app

1. `cp -r apps/hello apps/myapp`
2. Edit `apps/myapp/app.json`: `"id": "myapp"`, titles and descriptions in pt-BR and en.
3. Replace the screen in `newHello` (rename it) and drop what you do not need.
4. `go test ./...`: the **contract test** (`apps/contract_test.go`) checks every folder of `apps/`: a valid manifest,
   the id equal to the folder, `main.go` embedding `app.json` and calling `appkit.Run`, and complete `locales/`.
5. `.\build.ps1 -App myapp` and copy `dist/applications/myapp.app` to the reader.

Write the texts in **Portuguese (pt-BR)** with English as the fallback, through [`internal/i18n`](12-internationalization.md).

[← Internationalization](12-internationalization.md) · [Index](README.md)
