package widget

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// TextArea is a multi-line field (<textarea>): the text wraps and the box is
// Rows lines tall. Touching it opens the system keyboard.
type TextArea struct {
	Base
	Value       string
	Placeholder string
	Rows        int
	MaxLength   int
	ReadOnly    bool
	OnChange    Handler
}

// NewTextArea creates a text area showing rows lines.
func NewTextArea(placeholder string, rows int) *TextArea {
	t := &TextArea{Placeholder: placeholder, Rows: rows}
	t.init(t)
	applyFieldStyle(&t.Style)
	t.Style.Padding = geom.XY(16, 22)
	return t
}

// Measure implements Widget.
func (t *TextArea) Measure(avail float64) geom.Size {
	return geom.Size{W: geom.Min(avail, 360), H: lineHeight(FontBody) * float64(t.Rows)}
}

// Paint implements Widget.
func (t *TextArea) Paint(c *gfx.Canvas) {
	r := t.ContentRect()
	text, color := t.Value, Ink
	if text == "" {
		text, color = t.Placeholder, Muted
	}
	ts := t.Style.text(FontBody)
	ts.Color, ts.Ellipsis = t.fg(color), false
	lh := lineHeight(ts.Size)
	c.Clip(r, 0)
	for i, ln := range wrapText(text, r.W, ts.Size, ts.Bold) {
		c.Text(ln, geom.R(r.X, r.Y+float64(i)*lh, r.W, lh), ts)
	}
	c.ResetClip()
	paintFocusRing(c, &t.Base)
}

// HandlePointer implements PointerHandler.
func (t *TextArea) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick || t.ReadOnly || t.doc == nil {
		return
	}
	t.doc.RequestText(t.self, KeyboardRequest{
		Title: titleOf(&t.Base, t.Placeholder), Initial: t.Value, MaxLen: t.MaxLength,
		Done: func(text string, ok bool) {
			if ok {
				t.Value = truncateRunes(text, t.MaxLength)
				fire(t.OnChange, t)
			}
		},
	})
}
