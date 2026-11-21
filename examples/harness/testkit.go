package harness

import (
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// NewDoc puts content in a Screen with a fake keyboard (it answers "7") and lays
// it out: the usual start of a test.
func NewDoc(content widget.Widget) (*widget.Document, *widget.FakeKeyboard) {
	kb := &widget.FakeKeyboard{Reply: "7"}
	doc := Screen(1404, 1872, kb, content)
	doc.Layout()
	return doc, kb
}

// TapAt touches the screen at p (design pixels): finger down, then up.
func TapAt(doc *widget.Document, p geom.Point) {
	doc.HandlePointer(widget.PointerDown, p.X*doc.Scale, p.Y*doc.Scale)
	doc.HandlePointer(widget.PointerUp, p.X*doc.Scale, p.Y*doc.Scale)
}

// Tap touches the center of w.
func Tap(doc *widget.Document, w widget.Widget) { TapAt(doc, w.Elem().Bounds.Center()) }

// Inked counts the dark pixels of the rendered screen: zero means nothing was drawn.
func Inked(doc *widget.Document) int {
	n := 0
	for _, p := range doc.Render().Pix {
		if p < 128 {
			n++
		}
	}
	return n
}

// Interactive returns the visible elements that react to touch.
func Interactive(doc *widget.Document) []widget.Widget {
	var out []widget.Widget
	doc.Walk(func(w widget.Widget) bool {
		if _, isScroll := w.(*widget.Scroll); isScroll {
			return true // a window onto other elements: its own center is theirs
		}
		if _, ok := w.(widget.PointerHandler); ok && w.Elem().Visible() && widget.InView(w) {
			out = append(out, w)
		}
		return true
	})
	return out
}
