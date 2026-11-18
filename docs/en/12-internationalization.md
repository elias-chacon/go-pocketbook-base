# 12. Internationalization (i18n)

[← Troubleshooting](11-troubleshooting.md) · [Index](README.md)

Every app of this project is written in **Brazilian Portuguese (pt-BR)** and falls back to **English (en)**
automatically, through the reusable `internal/localization/i18n` package. A string that a language does not translate is
never blank: it comes from English, then from Portuguese, and as a last resort the key itself is shown (so a
forgotten translation is visible).

## The rules

| | |
|---|---|
| Primary language | `i18n.Primary` = `pt-BR`: what the app is written in and the default language |
| Fallback | `i18n.Fallback` = `en`: used for any key the current language lacks |
| Lookup order | current language → its base language (`pt-BR` → `pt`) → `en` → `pt-BR` → the key |
| Tags | `pt_BR`, `PT-br`, `pt_BR.UTF-8` are all understood (`i18n.Normalize`); `pt` finds the registered `pt-BR`, `en-GB` finds `en` |

## Files

One flat JSON file per language, named after it, embedded in the program with `go:embed` (no files to copy to the
reader):

```json
// locales/pt-BR.json
{ "greeting.title": "Olá!", "greeting.hello": "Bem-vinda(o), %s.", "items": "Você tem %d itens" }

// locales/en.json
{ "greeting.title": "Hello!", "greeting.hello": "Welcome, %s.", "items": "You have %d items", "farewell": "See you soon" }
```

Texts may contain `fmt` verbs (`%s`, `%d`), filled by the arguments of `T`. A text without arguments is returned
untouched, so a literal `%` is safe.

## Using it

```go
// examples/translation/translation.go
package translation

import (
	"embed"

	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads pt-BR (the primary language) and en (the fallback) from
// the embedded JSON files, one <language>.json per language.
func NewTranslator() *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // the files are embedded: failing here is a programming mistake
	}
	return tr
}

// GreetingScreen shows translated texts and a combo-box that changes the
// language. "farewell" exists only in English, so in pt-BR it falls back to it.
func GreetingScreen(tr *i18n.Bundle, name string) widget.Widget {
	title := widget.NewLabel("")
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	hello, items, bye := widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel("")
	lang := widget.NewComboBox("",
		widget.Option{Label: "Português", Value: "pt-BR"}, widget.Option{Label: "English", Value: "en"})
	lang.SetValue(tr.Lang())

	// Every visible text is set here, so changing the language is one call.
	refresh := func() {
		title.Text = tr.T("greeting.title")
		hello.Text = tr.T("greeting.hello", name)
		items.Text = tr.T("items", 3)
		bye.Text = tr.T("farewell")
		lang.Placeholder = tr.T("language")
	}
	refresh()
	tr.OnChange = refresh
	lang.OnChange = func(e *widget.Event) { tr.SetLang(lang.Value()) }

	return widget.NewColumn(12, title, hello, items, bye, lang)
}
```

| | |
|---|---|
| `i18n.New()` | an empty bundle whose language is pt-BR |
| `AddFS(fsys, dir)` | loads every `<language>.json` of a folder (an `embed.FS`) |
| `AddJSON(lang, data)` · `Add(lang, map)` | one language from JSON bytes / from a Go map (it merges with what is there) |
| `T(key, args...)` | the translated text |
| `SetLang(tag)` · `Lang()` · `Langs()` | choose and list languages; `OnChange` runs when the language changes |
| `Has(key)` · `Missing(lang)` | is a key defined; which keys a language still lacks |

## Changing the language at run time

Widgets hold plain strings, so after `SetLang` the texts must be set again. Keep every translated text in one
`refresh` function (as the example does) and call it from `OnChange`; or rebuild the page (with
[`Tabs`](06-components.md#tabs) it is rebuilt each time it is shown).

## Keeping a language complete

```go
if missing := tr.Missing("en"); len(missing) > 0 {
	t.Errorf("untranslated: %v", missing)
}
```

Put that in a test of your app: it lists the keys that pt-BR or en define and the given language does not.

[← Troubleshooting](11-troubleshooting.md) · [Index](README.md)
