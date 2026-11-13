// examples/screen.go
package examples

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/widget"
)

// Screen puts content inside a padded page and returns a ready-to-render
// document for a width x height screen. kb is the keyboard used by input
// fields: on the device pass the InkView keyboard, in tests a FakeKeyboard.
func Screen(width, height int, kb widget.Keyboard, content widget.Widget) *widget.Document {
	doc := widget.NewDocument(width, height)
	doc.Keyboard = kb

	page := widget.NewColumn(0, content)
	page.Style.Padding = geom.All(28)
	page.Style.Background = widget.Paper

	doc.SetRoot(page)
	return doc
}
