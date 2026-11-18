package widget

import (
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// Shared helpers for the interactive controls.

// iconStroke is the line width of icons drawn inside controls.
const iconStroke = 4.0

// fg returns normal, or the disabled color when the element is disabled.
func (b *Base) fg(normal gfx.Color) gfx.Color {
	if b.Disabled {
		return Disabled
	}
	return normal
}

// focused reports whether this element's text entry is open.
func (b *Base) focused() bool { return b.doc != nil && b.doc.focus == b.self }

// applyFieldStyle gives an element the look of a form field: bordered,
// rounded, at least one touch target tall.
func applyFieldStyle(s *Style) {
	s.Background, s.BorderColor = Paper, Ink
	s.BorderWidth, s.BorderRadius = BorderThin, Radius
	s.MinHeight = ControlHeight
	s.Padding = geom.XY(0, 22)
}

// paintFocusRing thickens the border of the field being edited.
func paintFocusRing(c *gfx.Canvas, b *Base) {
	if b.focused() {
		inner := b.Bounds.Inset(geom.All(BorderThin))
		c.StrokeRoundRect(inner, geom.Max(0, b.Style.BorderRadius-BorderThin), BorderThin, Ink)
	}
}

// paintTrigger draws the "text + chevron" content of a field that opens a
// popup (ComboBox, Dropdown). open flips the chevron.
func paintTrigger(c *gfx.Canvas, b *Base, text string, color gfx.Color, open bool) {
	r := b.ContentRect()
	chevron := geom.Rect{X: r.Right() - 40, Y: r.Y, W: 40, H: r.H}
	ts := b.Style.text(FontBody)
	ts.Color = color
	c.Text(text, geom.Rect{X: r.X, Y: r.Y, W: r.W - 48, H: r.H}, ts)
	kind := gfx.IconChevronDown
	if open {
		kind = gfx.IconChevronUp
	}
	c.Icon(kind, geom.Rect{X: chevron.X, Y: chevron.Y + (chevron.H-28)/2, W: 28, H: 28}, iconStroke, b.fg(Ink))
}

// triggerWidth is the natural width of a trigger's content.
func triggerWidth(text string) float64 {
	return gfx.TextWidth(text, FontBody, false) + 48
}
