package widget

import (
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// Option is one choice of a ComboBox.
type Option struct{ Label, Value string }

// ComboBox is a field that keeps one chosen option from a list (<select>). A short list opens as
// a menu under the field. A list that does not fit the screen opens the same popup, but with a
// search field on top (any part of the text, ignoring case and accents) and the options a page at
// a time with previous and next buttons, because e-ink does not scroll; it opens on the page of the
// chosen option. Nothing has to be set for that: it depends only on how many options there are.
type ComboBox struct {
	Base
	Options     []Option
	Index       int // selected option, -1 for none
	Placeholder string
	OnChange    Handler
	// SearchPlaceholder is the hint of the search field of a long list; NoMatchText what shows when
	// the search matches nothing.
	SearchPlaceholder, NoMatchText string
}

// NewComboBox creates a combo box with nothing selected.
func NewComboBox(placeholder string, options ...Option) *ComboBox {
	cb := &ComboBox{Options: options, Index: -1, Placeholder: placeholder, SearchPlaceholder: "Search", NoMatchText: "Nothing found"}
	cb.Init(cb)
	applyFieldStyle(&cb.Style)
	return cb
}

// Value is the selected option's value ("" when none): JS select.value.
func (cb *ComboBox) Value() string {
	if cb.Index < 0 || cb.Index >= len(cb.Options) {
		return ""
	}
	return cb.Options[cb.Index].Value
}

// SetValue selects the option with the given value.
func (cb *ComboBox) SetValue(v string) {
	cb.Index = -1
	for i, o := range cb.Options {
		if o.Value == v {
			cb.Index = i
		}
	}
}

func (cb *ComboBox) label() (text string, color gfx.Color) {
	if cb.Index >= 0 && cb.Index < len(cb.Options) {
		return cb.Options[cb.Index].Label, Ink
	}
	return cb.Placeholder, Muted
}

// Measure implements Widget: wide enough for the longest option.
func (cb *ComboBox) Measure(avail float64) geom.Size {
	w := triggerWidth(cb.Placeholder)
	for _, o := range cb.Options {
		w = geom.Max(w, triggerWidth(o.Label))
	}
	return geom.Size{W: geom.Min(w, avail), H: lineHeight(FontBody)}
}

// Paint implements Widget.
func (cb *ComboBox) Paint(c *gfx.Canvas) {
	text, color := cb.label()
	paintTrigger(c, &cb.Base, text, cb.fg(color), cb.doc.PopupOpenFor(cb))
}

// HandlePointer implements PointerHandler.
func (cb *ComboBox) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	d := cb.doc
	if d == nil || len(cb.Options) == 0 {
		return
	}
	room := geom.Max(d.H-cb.Bounds.Bottom(), cb.Bounds.Y) - popupGap
	if float64(len(cb.Options))*menuRowH+12 > room {
		if rows := comboRows(room); rows >= 1 {
			p := newComboPopup(cb, rows, comboWidth(cb, d.W))
			d.OpenPopup(cb, p, cb.Bounds, geom.Size{W: p.Style.Width.V, H: p.Style.Height.V})
			return
		}
	}
	items := make([]MenuItem, len(cb.Options))
	for i, o := range cb.Options {
		items[i] = MenuItem{Label: o.Label, Value: o.Value}
	}
	openMenu(cb, items, cb.Index, cb.Bounds, cb.Bounds.W, cb.choose)
}

// choose selects option i and tells OnChange.
func (cb *ComboBox) choose(i int) {
	cb.Index = i
	Fire(cb.OnChange, cb)
}

// comboWidth is how wide a paged popup is: the field's width, or the longest option when that is
// wider, and never wider than the screen.
func comboWidth(cb *ComboBox, screen float64) float64 {
	w := cb.Bounds.W
	for _, o := range cb.Options {
		w = geom.Max(w, gfx.TextWidth(o.Label, FontBody, false)+2*menuPadX)
	}
	return geom.Min(w, screen)
}
