package demo

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/widget"
)

var (
	pagePadding   = geom.XY(28, 28)
	footerPadding = geom.XY(14, 28)
	twoColumns    = []widget.Track{widget.FrTrack(1), widget.FrTrack(1)}
)

// field is a caption above a widget.
func field(caption string, w widget.Widget) *widget.Panel {
	l := widget.NewLabel(caption)
	l.Style.FontSize, l.Style.Color = widget.FontSmall, widget.Muted
	return widget.NewColumn(8, l, w)
}

// hint is small secondary text.
func hint(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Style.FontSize, l.Style.Color = widget.FontSmall, widget.Muted
	return l
}

// page lays cards out in two columns.
func twoColPage(cards ...widget.Widget) widget.Widget {
	return widget.NewGrid(24, twoColumns, cards...)
}

// span makes a grid child take both columns.
func span(w widget.Widget) widget.Widget {
	w.Elem().Style.GridColumnSpan = 2
	return w
}

// withID sets the element ID (the JS id attribute) and returns the widget.
func withID(w widget.Widget, id string) widget.Widget {
	w.Elem().ID = id
	return w
}
