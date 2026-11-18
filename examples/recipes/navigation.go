// examples/recipes/navigation.go
package recipes

import (
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// NewPagedApp builds a whole application: tabs that switch between the Login,
// Settings and Catalog pages. Only the current page lives in memory; what the
// user typed or chose is kept in the Tabs store and put back when the page is
// shown again (every element that must be remembered has an ID).
func NewPagedApp(width, height int, kb widget.Keyboard) *widget.Document {
	books := [][]string{
		{"Dom Casmurro", "Machado de Assis", "256"},
		{"Vidas Secas", "Graciliano Ramos", "176"},
		{"Iracema", "José de Alencar", "160"},
	}
	tabs := widget.NewTabs(nil,
		&widget.Tab{Title: "Login", Build: func() widget.Widget {
			f := NewLoginForm(func(string, bool) {})
			f.User.ID, f.Remember.ID = "user", "remember" // the password is never remembered
			return f.Root
		}},
		&widget.Tab{Title: "Settings", Build: func() widget.Widget {
			s := NewSettings()
			s.WiFi.ID, s.Brightness.ID, s.Language.ID, s.Topics.ID = "wifi", "brightness", "language", "topics"
			return s.Root
		}},
		// TabKeepAlive keeps the whole page (and its data) in memory instead.
		&widget.Tab{Title: "Catalog", Mode: widget.TabKeepAlive, Build: func() widget.Widget { return NewCatalog(books).Root }})
	tabs.Body.Style.Padding = geom.All(28) // the body already takes everything below the bar

	doc := widget.NewDocument(width, height)
	doc.Keyboard = kb
	doc.SetRoot(tabs)
	return doc
}
