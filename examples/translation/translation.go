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
