package widget

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Option is one choice of a ComboBox.
type Option struct{ Label, Value string }

// ComboBox is a field that keeps one chosen option from a list (<select>).
type ComboBox struct {
	Base
	Options     []Option
	Index       int // selected option, -1 for none
	Placeholder string
	OnChange    Handler
}

// NewComboBox creates a combo box with nothing selected.
func NewComboBox(placeholder string, options ...Option) *ComboBox {
	cb := &ComboBox{Options: options, Index: -1, Placeholder: placeholder}
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
	items := make([]MenuItem, len(cb.Options))
	for i, o := range cb.Options {
		items[i] = MenuItem{Label: o.Label, Value: o.Value}
	}
	openMenu(cb, items, cb.Index, cb.Bounds, cb.Bounds.W, func(i int) {
		cb.Index = i
		Fire(cb.OnChange, cb)
	})
}
