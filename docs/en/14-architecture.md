# 14. Architecture (DDD)

[← Apps and the contract](13-apps-and-the-contract.md) · [Index](README.md)

The repository follows **Domain-Driven Design**, applied with pragmatism: the code is divided into **bounded
contexts**, each with its own vocabulary and layers, and **one dependency rule** that a test enforces. The map and the
rule are written as code in `internal/architecture`; this page explains them.

## The bounded contexts

| Context | What it is | Packages | May use |
|---|---|---|---|
| **kernel** | the shared kernel: tiny value objects every context needs | `geom` (Rect, Point, Insets), `version` | nothing |
| **toolkit** | the UI toolkit, the core of the library | `gfx` (vector drawing), `widget` (Document, elements, layout, touch) | kernel |
| **device** | the e-reader as an outside system (anti-corruption layer) | `inkview` (cgo binding), `desktop` (home screen registration) | kernel, toolkit |
| **localization** | languages | `i18n` (pt-BR first, English fallback) | kernel |
| **platform** | what an app of the project *is* | `domain` · `application` · `infrastructure` · the facade `platform` | kernel, localization, toolkit, device |
| **launcher** | the App Launcher ([chapter 15](15-launcher.md)) | `domain` · `application` · `infrastructure` · `presentation` | kernel, localization, toolkit, and the platform's domain for its vocabulary |
| **showcase** | the content of the hello app | `showcase` | kernel, localization, toolkit |

Outside `internal/`: `apps/<id>` are **composition roots** (one per app: they wire the contexts together and may use
any of them), `examples/` are documentation samples and `tools/`, `cmd/` are development tools.

## Layers inside a context: the platform

```
infrastructure ──► application ──► domain
 (adapters)        (use cases)      (rules + ports)
```

| Layer | Holds | Example |
|---|---|---|
| **domain** | the rules and the vocabulary, with no I/O, plus the **ports** (interfaces the domain needs) | `Manifest.Validate`, `Manifest.TitleIn(lang)`, `InstalledApp`, `Settings`; ports `AppRegistry`, `SettingsStore`, `DesktopRegistrar` |
| **application** | the **use cases** that orchestrate the domain and the toolkit through ports | `Runtime` (connect the reader's events to an app's document), `FirstRun` (register an app and record it) |
| **infrastructure** | the **adapters** that implement the ports with the real world | `jsonfiles` (JSON files under `applications/.pbgo`), `reader` (InkView, the system keyboard, the home screen) |

A port is stated in the domain's own words and implemented outside: `domain.AppRegistry` says "save an installed app, list
the installed apps"; `jsonfiles.Apps` does it with files; a test does it with a slice. That is why `Runtime` and
`FirstRun` are tested on the PC with fakes and need neither a reader nor a disk.

The package `platform` is the context's **composition root**: it builds the adapters, hands them to the use cases and
re-exports what an app needs, so an app has a single import.

## Ubiquitous language

| Term | Meaning |
|---|---|
| **Document** | the tree of elements of one screen; the aggregate root of the toolkit |
| **Element** | a node of the Document (button, input, panel…); implements `widget.Widget` |
| **Design pixel** | the unit of every measure (reference width 1404), scaled by the Document |
| **App** | a program of the ecosystem: one folder under `apps/`, one `<id>.app` file |
| **Manifest** | the app's identity, in `app.json` (id, titles, description) |
| **InstalledApp** | an App that has run on this reader: manifest + version + files |
| **Settings** | preferences shared by every App (today the language) |
| **Port** / **Adapter** | an interface the domain defines / its implementation in infrastructure |
| **Plugin** | an App that the Launcher lists, installs and runs |

## The dependency rule

Dependencies point **inwards**. A context never reaches into another's insides, the domain depends on nothing but
the kernel and the language module, and nothing in `internal/` imports `apps/`, `examples/`, `tools/` or `cmd/`.
The exact table is `architecture.Rules`; `go test ./internal/architecture` reads every import of the repository and fails
with a message like:

```
breaks the dependency rule (see internal/architecture): internal/platform/domain/settings.go imports internal/toolkit/widget
```

A second test fails when a new folder appears directly under `internal/` without being declared in
`architecture.Contexts`: a new context is a decision, taken on purpose and documented.

## Where does my code go?

| I am writing… | It goes in |
|---|---|
| a widget, a layout feature, a style property | `internal/toolkit/widget` (drawing primitives in `gfx`) |
| a rule about what an app is (validation, which text for a language) | `platform/domain` |
| a step that combines rules to do something (run an app, register it) | `platform/application` |
| reading or writing files, talking to InkView, the network | `platform/infrastructure` (a raw binding: `device`) |
| the screen of an app | the app's folder `apps/<id>` |
| a tiny value used by every context | `internal/kernel` (keep it tiny) |
| text the user reads | the app's `locales/*.json`, pt-BR first (see [i18n](12-internationalization.md)) |

## The samples follow the same map

`examples/` is organized by the chapter, that is the context, that each sample teaches:

```
examples/components/    chapter 6: one file per component (button.go, table.go, modal.go…)
examples/recipes/       chapter 8: complete screens (login, settings, catalog, order…)
examples/custom/        chapter 9: your own element
examples/translation/   chapter 12: i18n with JSON files
examples/harness/       what the samples and their tests share: Screen, SavePNG, the test kit
```

## Adding a context or an app

- **An app:** `cp -r apps/hello apps/<id>` ([Apps and the contract](13-apps-and-the-contract.md)); the contract test checks it.
- **A context:** create `internal/<name>/` with the layers it needs (`domain`, `application`, `infrastructure`, and a
  presentation layer for screens), add its rules to `architecture.Rules` and the folder to `architecture.Contexts`, and
  describe it in the documentation of `internal/architecture`. The Launcher is one: `internal/launcher`, with
  `apps/launcher` as its composition root.

[← Apps and the contract](13-apps-and-the-contract.md) · [Index](README.md)
