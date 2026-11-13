package widget

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Dropdown is a button that opens a menu of actions. Unlike a ComboBox it does
// not keep a value: each item runs its OnSelect.
type Dropdown struct {
	Base
	Text     string
	Items    []MenuItem
	Chosen   MenuItem // the last item picked
	OnSelect Handler
}

// NewDropdown creates a drop-down button.
func NewDropdown(text string, items ...MenuItem) *Dropdown {
	d := &Dropdown{Text: text, Items: items}
	d.Init(d)
	applyFieldStyle(&d.Style)
	d.Style.Bold = true
	return d
}

// Measure implements Widget.
func (d *Dropdown) Measure(avail float64) geom.Size {
	return geom.Size{W: geom.Min(triggerWidth(d.Text)+8, avail), H: lineHeight(FontBody)}
}

// Paint implements Widget.
func (d *Dropdown) Paint(c *gfx.Canvas) {
	paintTrigger(c, &d.Base, d.Text, d.fg(Ink), d.doc.PopupOpenFor(d))
}

// HandlePointer implements PointerHandler.
func (d *Dropdown) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	openMenu(d, d.Items, -1, d.Bounds, d.Bounds.W, func(i int) {
		d.Chosen = d.Items[i]
		if d.Chosen.OnSelect != nil {
			d.Chosen.OnSelect()
		}
		Fire(d.OnSelect, d)
	})
}
