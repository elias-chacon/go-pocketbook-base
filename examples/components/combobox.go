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

// LongComboBoxExample is a combo box for a long list: no change is needed, it opens the same popup
// with a search field and the options a page at a time, on the page of the option chosen.
func LongComboBoxExample() widget.Widget {
	var options []widget.Option
	for _, name := range []string{"Afrikaans", "Alemão", "Árabe", "Armênio", "Basco", "Búlgaro", "Catalão", "Chinês", "Coreano", "Croata", "Dinamarquês", "Eslovaco", "Esloveno", "Espanhol", "Estoniano", "Finlandês", "Francês", "Galego", "Georgiano", "Grego", "Hebraico", "Hindi", "Holandês", "Húngaro", "Indonésio", "Inglês", "Irlandês", "Islandês", "Italiano", "Japonês", "Latim", "Letão", "Lituano", "Norueguês", "Persa", "Polonês", "Português", "Romeno", "Russo", "Sérvio", "Sueco", "Tailandês", "Tcheco", "Turco", "Ucraniano", "Vietnamita"} {
		options = append(options, widget.Option{Label: name, Value: name})
	}
	language := widget.NewComboBox("Language", options...)
	language.SearchPlaceholder, language.NoMatchText = "Search", "Nothing found"
	language.SetValue("Português") // the popup opens on the page that has it
	language.OnChange = func(e *widget.Event) { fmt.Println("language:", language.Value()) }
	return language
}
