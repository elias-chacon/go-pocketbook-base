// examples/navigation.go
package examples

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/widget"
)

// NewPagedApp builds a whole application: a menu bar that switches between the
// Login, Settings and Catalog pages. Pass the screen size and the keyboard.
func NewPagedApp(width, height int, kb widget.Keyboard) *widget.Document {
	books := [][]string{
		{"Dom Casmurro", "Machado de Assis", "256"},
		{"Vidas Secas", "Graciliano Ramos", "176"},
		{"Iracema", "José de Alencar", "160"},
	}
	pages := []struct {
		title   string
		content widget.Widget
	}{
		{"Login", NewLoginForm(func(string, bool) {}).Root},
		{"Settings", NewSettings().Root},
		{"Catalog", NewCatalog(books).Root},
	}

	body := widget.NewColumn(0)
	body.Style.FlexGrow = 1 // the page takes everything below the bar
	body.Style.Padding = geom.All(28)

	bar := widget.NewMenuBar()
	show := func(i int) {
		body.ClearChildren()
		body.AppendChild(pages[i].content)
		for j, m := range bar.Menus {
			m.Selected = j == i // the current page is drawn filled
		}
	}
	for i, p := range pages {
		i := i
		bar.Menus = append(bar.Menus, &widget.Menu{Label: p.title, OnSelect: func() { show(i) }})
	}

	doc := widget.NewDocument(width, height)
	doc.Keyboard = kb
	doc.SetRoot(widget.NewColumn(0, bar, body))
	show(0)
	return doc
}
