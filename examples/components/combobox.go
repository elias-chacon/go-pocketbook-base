package components

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/widget"
)

// ComboBoxExample keeps one option from a list.
func ComboBoxExample() widget.Widget {
	city := widget.NewComboBox("Choose a city",
		widget.Option{Label: "São Paulo", Value: "sp"},
		widget.Option{Label: "Rio de Janeiro", Value: "rj"},
		widget.Option{Label: "Curitiba", Value: "cwb"})
	city.OnChange = func(e *widget.Event) { fmt.Println("city:", city.Value()) }
	return city
}

// SearchComboBoxExample is a combo box for a long list: it opens a window with a search field and
// the options a page at a time, on the page of the option chosen.
func SearchComboBoxExample() widget.Widget {
	var options []widget.Option
	for _, name := range []string{"Afrikaans", "Alemão", "Árabe", "Búlgaro", "Chinês", "Coreano", "Dinamarquês", "Espanhol", "Finlandês",
		"Francês", "Grego", "Hebraico", "Inglês", "Italiano", "Japonês", "Latim", "Polonês", "Português", "Russo", "Sueco"} {
		options = append(options, widget.Option{Label: name, Value: name})
	}
	language := widget.NewSearchComboBox("Language", options...)
	language.Title, language.SearchPlaceholder, language.CancelText = "Pick a language", "Search", "Close"
	language.SetValue("Português") // the window opens on the page that has it
	language.OnChange = func(e *widget.Event) { fmt.Println("language:", language.Value()) }
	return language
}
